package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/anomalyco/qyvora-jabari/internal/analysis"
	"github.com/anomalyco/qyvora-jabari/internal/core"
	"github.com/anomalyco/qyvora-jabari/internal/discovery"
	"github.com/anomalyco/qyvora-jabari/internal/enumeration"
	"github.com/anomalyco/qyvora-jabari/internal/validation"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context())
		},
	}
}

// runSingleStage builds an assessment environment for the current target and
// executes one stage, persisting and rendering the result.
func runSingleStage(ctx context.Context, stage core.Stage) error {
	t, err := requireTarget()
	if err != nil {
		return err
	}
	env, cleanup, err := newAssessmentEnv(ctx, t)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := stage.Run(ctx, env); err != nil {
		return err
	}
	env.Session.Finish()

	if path, err := persistSession(env.Session); err == nil {
		log.Info("session saved to %s", path)
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
	env, cleanup, err := newAssessmentEnv(ctx, t)
	if err != nil {
		return err
	}
	defer cleanup()

	if t.Device == nil {
		if err := (&discovery.Stage{}).Run(ctx, env); err != nil {
			return err
		}
	}
	if err := (&analysis.Stage{}).Run(ctx, env); err != nil {
		return err
	}
	env.Session.Finish()
	if path, err := persistSession(env.Session); err == nil {
		log.Info("session saved to %s", path)
	}
	return renderSession(ctx, env.Session)
}

// runValidate implements "jabari validate".
func runValidate(ctx context.Context) error {
	return runSingleStage(ctx, &validation.Stage{})
}
