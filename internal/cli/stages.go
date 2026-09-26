package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/QYVORA/qyvora-jabari/internal/analysis"
	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/internal/discovery"
	"github.com/QYVORA/qyvora-jabari/internal/enumeration"
	"github.com/QYVORA/qyvora-jabari/internal/events"
	"github.com/QYVORA/qyvora-jabari/internal/orchestration"
	"github.com/QYVORA/qyvora-jabari/internal/poc"
	"github.com/QYVORA/qyvora-jabari/internal/validation"
)

// stageRunner is the signature shared by single-stage command handlers.
type stageRunner func(ctx context.Context) error

// newStageCmd builds a command that runs exactly one pipeline stage against
// the current target.
func newStageCmd(use, short string, run stageRunner) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context())
		},
	}
}

// runSingleStage builds an assessment environment for the current target and
// executes one stage, persisting and rendering the result. The JSONL event
// stream bound by newAssessmentEnv is used when --events is configured.
func runSingleStage(ctx context.Context, stage core.Stage) error {
	t, err := requireTarget()
	if err != nil {
		return err
	}
	profile := orchestration.Profile(cfg.GetString("profile"))
	env, cleanup, err := newAssessmentEnv(ctx, t, profile)
	if err != nil {
		return err
	}
	defer cleanup()

	if env.Events != nil {
		env.Events.Info("jabari", events.ScanStarted, map[string]any{
			"profile":     string(profile),
			"target_id":   t.ID,
			"target_type": string(t.Type),
			"stage":       stage.Name(),
		})
	}

	if err := stage.Run(ctx, env); err != nil {
		if env.Events != nil {
			env.Events.Fail("jabari", events.Error, map[string]any{"message": err.Error()})
		}
		return err
	}
	env.Session.Finish()

	if path, err := persistSession(env.Session); err == nil {
		log.Info("session saved to %s", path)
		if env.Events != nil {
			env.Events.Info("jabari", events.ReportGenerated, map[string]any{
				"path":   path,
				"format": "json",
			})
		}
	}

	if env.Events != nil {
		env.Events.Info("jabari", events.ScanCompleted, map[string]any{
			"findings":   len(env.Session.Findings),
			"risk_score": env.Session.RiskScore,
			"risk_level": env.Session.RiskLevel,
			"stages":     env.Session.Stages,
			"stage":      stage.Name(),
		})
	}
	return renderSession(ctx, env.Session)
}

// runDiscover implements "jabari discover".
func runDiscover(ctx context.Context) error {
	return runSingleStage(ctx, &discovery.Stage{})
}

// runEnumerate implements "jabari enumerate".
func runEnumerate(ctx context.Context) error {
	return runSingleStage(ctx, &enumeration.Stage{})
}

// runAnalyze implements "jabari analyze". It runs discovery first so rules
// have device data to evaluate even if the target was never discovered.
func runAnalyze(ctx context.Context) error {
	t, err := requireTarget()
	if err != nil {
		return err
	}
	profile := orchestration.Profile(cfg.GetString("profile"))
	env, cleanup, err := newAssessmentEnv(ctx, t, profile)
	if err != nil {
		return err
	}
	defer cleanup()

	if env.Events != nil {
		env.Events.Info("jabari", events.ScanStarted, map[string]any{
			"profile":     string(profile),
			"target_id":   t.ID,
			"target_type": string(t.Type),
			"stage":       "analyze",
		})
	}

	if t.Device == nil {
		if err := (&discovery.Stage{}).Run(ctx, env); err != nil {
			if env.Events != nil {
				env.Events.Fail("jabari", events.Error, map[string]any{"message": err.Error()})
			}
			return err
		}
	}
	if err := (&analysis.Stage{}).Run(ctx, env); err != nil {
		if env.Events != nil {
			env.Events.Fail("jabari", events.Error, map[string]any{"message": err.Error()})
		}
		return err
	}
	env.Session.Finish()
	if path, err := persistSession(env.Session); err == nil {
		log.Info("session saved to %s", path)
		if env.Events != nil {
			env.Events.Info("jabari", events.ReportGenerated, map[string]any{
				"path":   path,
				"format": "json",
			})
		}
	}
	if env.Events != nil {
		env.Events.Info("jabari", events.ScanCompleted, map[string]any{
			"findings":   len(env.Session.Findings),
			"risk_score": env.Session.RiskScore,
			"risk_level": env.Session.RiskLevel,
			"stages":     env.Session.Stages,
			"stage":      "analyze",
		})
	}
	return renderSession(ctx, env.Session)
}

// runValidate implements "jabari validate".
func runValidate(ctx context.Context) error {
	return runSingleStage(ctx, &validation.Stage{})
}

// pocFlags are shared by the standalone "poc" command.
var pocFlags struct {
	highRisk     bool
	moduleFilter []string
}

// pocFilter builds the engine module filter from the flag value. An empty
// filter means every registered module runs.
func pocFilter() map[string]bool {
	if len(pocFlags.moduleFilter) == 0 {
		return nil
	}
	out := make(map[string]bool, len(pocFlags.moduleFilter))
	for _, id := range pocFlags.moduleFilter {
		out[id] = true
	}
	return out
}

// newPocCmd implements "jabari poc": run proof-of-concept modules against the
// current target. The stage enforces target authorization itself and returns
// exit code 3 when the gate is not satisfied.
func newPocCmd() *cobra.Command {
	cmd := newStageCmd("poc", "Run proof-of-concept modules against the current target", runPoc)
	cmd.Flags().BoolVar(&pocFlags.highRisk, "poc-high-risk", false,
		"allow PoC modules that change device state (e.g. android.exported_activity)")
	cmd.Flags().StringSliceVar(&pocFlags.moduleFilter, "poc-module", nil,
		"run only the named PoC module(s); repeatable")
	return cmd
}

// runPoc implements the standalone poc stage command.
func runPoc(ctx context.Context) error {
	return runSingleStage(ctx, &poc.Stage{AllowHighRisk: pocFlags.highRisk, ModuleFilter: pocFilter()})
}
