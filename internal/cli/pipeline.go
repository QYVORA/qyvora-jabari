package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/anomalyco/qyvora-jabari/internal/core"
	"github.com/anomalyco/qyvora-jabari/internal/evidence"
	"github.com/anomalyco/qyvora-jabari/internal/orchestration"
	"github.com/anomalyco/qyvora-jabari/internal/rules"
	"github.com/anomalyco/qyvora-jabari/internal/rules/builtin"
	"github.com/anomalyco/qyvora-jabari/internal/transport"
	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// newAssessmentEnv builds the shared environment for an assessment run:
// transport, session, evidence store, and a rule registry with the builtin
// rules registered. The returned cleanup function disconnects the transport.
func newAssessmentEnv(ctx context.Context, t *models.Target) (*core.Env, func(), error) {
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
	sess.Profile = cfg.GetString("profile")

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
// and persists the resulting session.
func runPipeline(ctx context.Context, profile orchestration.Profile) (*models.Session, error) {
	t, err := requireTarget()
	if err != nil {
		return nil, err
	}

	env, cleanup, err := newAssessmentEnv(ctx, t)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	pipe := orchestration.ForProfile(profile)
	if err := pipe.Run(ctx, env); err != nil {
		return nil, err
	}
	env.Session.Finish()

	path, err := persistSession(env.Session)
	if err != nil {
		log.Warn("persisting session: %v", err)
	} else {
		log.Info("session saved to %s", path)
	}
	return env.Session, nil
}

// elapsed helper for progress logging.
func elapsed(start time.Time) time.Duration {
	return time.Since(start).Round(time.Millisecond)
}

// nowUTC returns the current time in UTC, the canonical session timestamp.
func nowUTC() time.Time { return time.Now().UTC() }
