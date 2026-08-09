// Package core defines the interfaces that connect the assessment stages into
// a pipeline. Keeping the pipeline contract here (rather than in any one
// stage) lets new stages be added without changing existing ones.
package core

import (
	"context"

	"github.com/spf13/viper"

	"github.com/anomalyco/qyvora-jabari/internal/evidence"
	"github.com/anomalyco/qyvora-jabari/internal/logger"
	"github.com/anomalyco/qyvora-jabari/internal/rules"
	"github.com/anomalyco/qyvora-jabari/internal/transport"
	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// Env is the shared context every pipeline stage receives. It bundles the
// target, session, transport, and the services stages may need (rules,
// evidence store, logger, config).
type Env struct {
	Target    *models.Target
	Session   *models.Session
	Transport transport.Transport
	Rules     *rules.Registry
	Evidence  *evidence.Store
	Log       *logger.Logger
	Config    *viper.Viper
	// Apps holds the application inventory collected by the enumeration
	// stage. It feeds the analysis stage's evaluation context.
	Apps []models.Application
}

// Stage is one step of the assessment pipeline (discovery, enumeration,
// analysis, validation, risk, reporting). Stages run in registration order
// and share the same Env.
type Stage interface {
	// Name is the human-readable stage name used in logs and reports.
	Name() string
	// Run executes the stage. Stages must honor context cancellation.
	Run(ctx context.Context, env *Env) error
}

// Reporter is the terminal-facing reporter interface. Assesses use it to
// print a per-stage progress summary as the pipeline runs.
type Reporter interface {
	// StageStarted is called before a stage begins.
	StageStarted(name string)
	// StageFinished is called after a stage completes, with an error if one
	// occurred.
	StageFinished(name string, err error)
}
