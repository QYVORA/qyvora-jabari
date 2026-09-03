// Package stage (in internal/poc) is split across files; this file holds the
// pipeline stage that drives the PoC engine and persists its results.
package poc

import (
	"context"
	"time"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	errs "github.com/QYVORA/qyvora-jabari/internal/errors"
	"github.com/QYVORA/qyvora-jabari/internal/events"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// Stage is the assessment pipeline stage that executes proof-of-concept
// modules against the live target. It is never part of a default profile;
// it only runs when explicitly requested (--poc or the "poc" command) and
// still requires target authorization.
type Stage struct {
	// AllowHighRisk overrides the high-risk gate (normally controlled by
	// poc.high_risk config). Tests set this directly.
	AllowHighRisk bool
	// ModuleFilter restricts the stage to specific module IDs.
	ModuleFilter map[string]bool
}

// Name returns the stage name used in logs and reports.
func (s *Stage) Name() string { return "poc" }

// Run executes the PoC engine against the session findings and persists every
// result as a session PocRun, updating finding exploitability and emitting
// module lifecycle events.
func (s *Stage) Run(ctx context.Context, env *core.Env) error {
	if env == nil || env.Session == nil {
		return nil
	}
	// The poc stage is offensive: it requires the explicit authorization gate
	// to be satisfied even when invoked directly.
	if env.Target == nil || !env.Target.Authorized() {
		return errs.NewExitError(3, "poc stage requires target authorization; re-run with --authorized")
	}
	// APK-only targets have no live device to prove anything on.
	if env.Target.Type == models.TargetAPK {
		return nil
	}

	eng := &Engine{AllowHighRisk: s.AllowHighRisk, ModuleFilter: s.ModuleFilter}
	if env.Config != nil && env.Config.GetBool("poc.high_risk") {
		eng.AllowHighRisk = true
	}

	results := eng.Run(ctx, env)
	var proven int
	for i := range results {
		res := &results[i]
		run := &models.PocRun{
			ID:         models.NewID("poc"),
			Module:     res.Module,
			FindingID:  res.FindingID,
			Status:     res.Status,
			Risk:       res.Risk,
			Summary:    res.Summary,
			Evidence:   append([]string(nil), res.Evidence...),
			StartedAt:  time.Now().UTC(),
			FinishedAt: time.Now().UTC(),
		}
		if res.Err != nil {
			run.Error = res.Err.Error()
		}
		env.Session.AddPoc(run)
		if res.Status == models.PocProven {
			proven++
		}
		setExploitability(env, res)
		emitRun(env, res)
	}

	if env.Log != nil {
		env.Log.Info("poc stage proved %d of %d run(s)", proven, len(results))
	}
	return nil
}

// setExploitability advances the exploitability ladder of the targeted
// finding based on the PoC outcome.
func setExploitability(env *core.Env, res *Result) {
	switch res.Status {
	case models.PocProven:
		for _, f := range env.Session.Findings {
			if f != nil && f.ID == res.FindingID {
				f.Exploitability = "exploited"
				return
			}
		}
	case models.PocNotProven:
		for _, f := range env.Session.Findings {
			if f != nil && f.ID == res.FindingID && f.Exploitability == "" {
				f.Exploitability = "dynamic"
				return
			}
		}
	}
}

// emitRun streams the module lifecycle events for one PoC execution.
func emitRun(env *core.Env, res *Result) {
	if env.Events == nil {
		return
	}
	data := map[string]any{
		"poc_module": res.Module,
		"finding_id": res.FindingID,
		"status":     string(res.Status),
		"risk":       res.Risk,
		"summary":    res.Summary,
	}
	env.Events.Info("jabari", events.ModuleStarted, map[string]any{
		"poc_module": res.Module,
		"finding_id": res.FindingID,
	})
	env.Events.Info("jabari", events.ModuleCompleted, data)
}
