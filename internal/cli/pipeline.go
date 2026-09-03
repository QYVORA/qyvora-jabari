package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/internal/events"
	"github.com/QYVORA/qyvora-jabari/internal/evidence"
	"github.com/QYVORA/qyvora-jabari/internal/orchestration"
	"github.com/QYVORA/qyvora-jabari/internal/poc"
	"github.com/QYVORA/qyvora-jabari/internal/rules"
	"github.com/QYVORA/qyvora-jabari/internal/rules/builtin"
	"github.com/QYVORA/qyvora-jabari/internal/transport"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// newAssessmentEnv builds the shared environment for an assessment run:
// transport, session, evidence store, and a rule registry with the builtin
// rules registered. The returned cleanup function disconnects the transport.
func newAssessmentEnv(ctx context.Context, t *models.Target, profile orchestration.Profile) (*core.Env, func(), error) {
	tr, err := transport.NewForTarget(t, timeout)
	if err != nil {
		return nil, func() {}, err
	}

	// APK targets have no live transport; every other target type must
	// connect before stages run.
	if t.Type != models.TargetAPK {
		if err := tr.Connect(ctx); err != nil {
			return nil, func() {}, err
		}
	}

	evStore, err := evidence.New(reportDir())
	if err != nil {
		return nil, func() {}, fmt.Errorf("evidence store: %w", err)
	}

	reg := rules.NewRegistry()
	if err := builtin.Register(reg); err != nil {
		return nil, func() {}, fmt.Errorf("registering builtin rules: %w", err)
	}

	sess := models.NewSession()
	sess.TargetID = t.ID
	sess.Profile = string(profile)

	env := &core.Env{
		Target:    t,
		Session:   sess,
		Transport: tr,
		Rules:     reg,
		Evidence:  evStore,
		Log:       log,
		Config:    cfg,
	}

	cleanup := func() { _ = tr.Disconnect() }
	return env, cleanup, nil
}

// reportDir resolves the report output directory from configuration.
func reportDir() string {
	dir := cfg.GetString("report.dir")
	if dir == "" {
		dir = "reports"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Warn("creating report directory: %v", err)
	}
	return dir
}

// persistSession writes the finished session as JSON so reports can be
// re-rendered later. The session filename is deterministic per session ID.
func persistSession(s *models.Session) (string, error) {
	path := filepath.Join(reportDir(), "session-"+s.ID+".json")
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// loadSession reads a session JSON file into a Session.
func loadSession(path string) (*models.Session, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s models.Session
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// runPipeline executes a pipeline for a profile against the current target
// and persists the resulting session. When --events is configured, scan
// lifecycle and finding events are streamed as JSONL.
func runPipeline(ctx context.Context, profile orchestration.Profile) (*models.Session, error) {
	t, err := requireTarget()
	if err != nil {
		return nil, err
	}

	env, cleanup, err := newAssessmentEnv(ctx, t, profile)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Bind the optional JSONL event stream to this run before any stage
	// executes so the full lifecycle (scan.started .. scan.completed) is
	// captured.
	emitter, closeStream, err := newEventsEmitter(env.Session.ID)
	if err != nil {
		return nil, err
	}
	if closeStream != nil {
		defer func() { _ = closeStream() }()
	}
	env.Events = emitter
	if emitter != nil {
		emitter.Info("jabari", events.ScanStarted, map[string]any{
			"profile":     string(profile),
			"target_id":   t.ID,
			"target_type": string(t.Type),
		})
	}

	pipe := orchestration.ForProfile(profile)
	// Offline APK targets have no live transport: use the static-analysis
	// pipeline (no discovery/enumeration) instead of the device pipeline.
	if t.Type == models.TargetAPK {
		pipe = orchestration.ForAPKProfile(profile)
	}
	// The poc stage is opt-in: it runs only when explicitly requested on an
	// authorized target (authorization is enforced by the stage itself).
	if assessFlags.poc {
		pipe.Add(&poc.Stage{AllowHighRisk: assessFlags.pocHighRisk})
	}
	if err := pipe.Run(ctx, env); err != nil {
		if emitter != nil {
			emitter.Fail("jabari", events.Error, map[string]any{"message": err.Error()})
		}
		return nil, err
	}
	env.Session.Finish()

	path, err := persistSession(env.Session)
	if err != nil {
		log.Warn("persisting session: %v", err)
	} else {
		log.Info("session saved to %s", path)
		if emitter != nil {
			emitter.Info("jabari", events.ReportGenerated, map[string]any{
				"path":   path,
				"format": "json",
			})
		}
	}

	if emitter != nil {
		emitter.Info("jabari", events.ScanCompleted, map[string]any{
			"findings":   len(env.Session.Findings),
			"risk_score": env.Session.RiskScore,
			"risk_level": env.Session.RiskLevel,
			"stages":     env.Session.Stages,
		})
	}
	return env.Session, nil
}

// nowUTC returns the current time in UTC, the canonical session timestamp.
func nowUTC() time.Time { return time.Now().UTC() }
