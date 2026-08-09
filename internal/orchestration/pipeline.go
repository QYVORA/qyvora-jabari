// Package orchestration coordinates the assessment pipeline. It owns the
// stage list, profile selection, and per-stage progress reporting so the CLI
// stays thin.
package orchestration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anomalyco/qyvora-jabari/internal/analysis"
	"github.com/anomalyco/qyvora-jabari/internal/core"
	"github.com/anomalyco/qyvora-jabari/internal/discovery"
	"github.com/anomalyco/qyvora-jabari/internal/enumeration"
	"github.com/anomalyco/qyvora-jabari/internal/risk"
	"github.com/anomalyco/qyvora-jabari/internal/validation"
)

// Profile selects which stages run and how deep the assessment goes. The
// profile names mirror the documented set.
type Profile string

const (
	ProfileQuick       Profile = "quick"
	ProfileStandard    Profile = "standard"
	ProfileDeep        Profile = "deep"
	ProfileApplication Profile = "application"
	ProfileDevice      Profile = "device"
	ProfileNetwork     Profile = "network"
	ProfileCompliance  Profile = "compliance"
	ProfileResearch    Profile = "research"
)

// Profiles lists every supported profile in documentation order.
var Profiles = []Profile{
	ProfileQuick, ProfileStandard, ProfileDeep, ProfileApplication,
	ProfileDevice, ProfileNetwork, ProfileCompliance, ProfileResearch,
}

// IsValid reports whether name is a known profile.
func IsValid(name string) bool {
	for _, p := range Profiles {
		if string(p) == name {
			return true
		}
	}
	return false
}

// ErrNoStages is returned when a pipeline is executed without stages.
var ErrNoStages = errors.New("orchestration pipeline has no stages")

// Pipeline runs a fixed sequence of stages against a shared Env.
type Pipeline struct {
	stages []core.Stage
}

// NewPipeline returns an empty pipeline.
func NewPipeline() *Pipeline { return &Pipeline{} }

// Add appends a stage to the pipeline.
func (p *Pipeline) Add(s core.Stage) { p.stages = append(p.stages, s) }

// Stages returns the current stage list.
func (p *Pipeline) Stages() []core.Stage { return p.stages }

// Run executes every stage in order, recording stage names on the session.
// Context cancellation aborts the remaining stages.
func (p *Pipeline) Run(ctx context.Context, env *core.Env) error {
	if len(p.stages) == 0 {
		return ErrNoStages
	}
	for _, stage := range p.stages {
		if err := ctx.Err(); err != nil {
			return err
		}
		if env.Session != nil {
			env.Session.Stages = append(env.Session.Stages, stage.Name())
		}
		start := time.Now()
		err := stage.Run(ctx, env)
		reportStage(env, stage.Name(), start, err)
		if err != nil {
			if env.Session != nil {
				env.Session.Errors = append(env.Session.Errors,
					fmt.Sprintf("%s: %v", stage.Name(), err))
			}
			return fmt.Errorf("stage %s: %w", stage.Name(), err)
		}
	}
	return nil
}

// reportStage logs stage progress and invokes the optional terminal reporter.
func reportStage(env *core.Env, name string, start time.Time, err error) {
	if env.Log != nil {
		elapsed := time.Since(start).Round(time.Millisecond)
		if err != nil {
			env.Log.Error("stage %s failed after %s: %v", name, elapsed, err)
		} else {
			env.Log.Debug("stage %s completed in %s", name, elapsed)
		}
	}
}

// ForProfile builds the stage list for a profile.
//
//   - quick:       discovery, analysis, risk, reporting (no app inventory)
//   - device:      discovery, enumeration (device only), analysis, risk
//   - application: discovery, enumeration (apps), analysis, risk
//   - standard/deep/compliance/research/network: full pipeline
//
// Reporting is appended by the CLI so the report format stays a CLI concern.
func ForProfile(p Profile) *Pipeline {
	pipe := NewPipeline()
	pipe.Add(&discovery.Stage{})
	switch p {
	case ProfileQuick:
		// quick skips application enumeration to stay fast.
	case ProfileApplication:
		pipe.Add(&enumeration.Stage{})
	case ProfileDevice:
		pipe.Add(&enumeration.Stage{})
	default:
		pipe.Add(&enumeration.Stage{})
	}
	pipe.Add(&analysis.Stage{})
	pipe.Add(&validation.Stage{})
	pipe.Add(&risk.Stage{})
	return pipe
}
