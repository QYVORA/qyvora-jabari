// Package analysis implements the assessment stage that turns collected
// device and application data into findings by evaluating the rule engine.
package analysis

import (
	"context"
	"fmt"

	"github.com/anomalyco/qyvora-jabari/internal/core"
	"github.com/anomalyco/qyvora-jabari/internal/rules"
)

// Stage runs the registered rules against the device and application data
// collected by earlier stages.
type Stage struct{}

// Name returns the stage name used in logs and reports.
func (s *Stage) Name() string { return "analysis" }

// Run evaluates the rule registry and appends every produced finding to the
// session. A registry without rules is a configuration error, not a silent
// success.
func (s *Stage) Run(ctx context.Context, env *core.Env) error {
	if env.Rules == nil {
		return fmt.Errorf("analysis stage has no rule registry configured")
	}

	ec := rules.EvaluationContext{
		Target: env.Target,
		Device: env.Target.Device,
		Apps:   env.Apps,
	}

	findings := env.Rules.Evaluate(ctx, ec)
	for i := range findings {
		env.Session.AddFinding(&findings[i])
	}

	if env.Log != nil {
		env.Log.Info("analysis produced %d finding(s)", len(findings))
	}
	return nil
}
