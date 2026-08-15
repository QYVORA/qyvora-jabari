// Package poc implements the framework-native proof-of-concept stage. It is
// the offense-validation counterpart to the analysis and validation stages:
// after rules flag a suspected weakness, a PoC module attempts to demonstrate
// it on the live device and records the outcome as structured evidence.
//
// Every module executes through the session's transport (adb) and is gated by
// target authorization plus an explicit opt-in. High-risk modules (anything
// that changes device state, such as launching an activity) are skipped unless
// the operator explicitly allows them. Nothing here is ever run by default in
// an assessment; the stage only runs when requested.
package poc

import (
	"context"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// Risk is how disruptive a PoC module is. High-risk modules change device
// state and require explicit opt-in.
type Risk string

const (
	// RiskLow modules are strictly read-only probes.
	RiskLow Risk = "low"
	// RiskMedium modules may perform harmless actions such as executing a
	// command as an app that is already debuggable.
	RiskMedium Risk = "medium"
	// RiskHigh modules change device state (for example launching an
	// activity on screen) and require explicit opt-in.
	RiskHigh Risk = "high"
)

// ModuleMeta is the static metadata of a PoC module.
type ModuleMeta struct {
	// ID is the dotted module identifier, e.g. "android.run_as_debuggable".
	ID string
	// Name is a short human-readable title.
	Name string
	// Category groups modules, e.g. "device" or "application".
	Category string
	// Risk is the module's disruptiveness.
	Risk Risk
	// Description explains what the module proves and how.
	Description string
	// Preconditions notes what must be true for the module to prove anything.
	Preconditions string
}

// Result is the outcome of one module execution against one finding.
type Result struct {
	// Module is the module id that produced the result.
	Module string
	// FindingID is the finding the module targeted.
	FindingID string
	// Status is proven, not-proven, skipped or error.
	Status models.PocStatus
	// Summary is a human-readable verdict.
	Summary string
	// Evidence lists the raw proof captured during the run.
	Evidence []string
	// Risk is the module risk level at run time.
	Risk string
	// Err carries the failure when Status is error.
	Err error
}

// Module is one proof-of-concept execution unit. Modules must be safe,
// reversible, and honor context cancellation.
type Module interface {
	// Meta returns the module's static metadata.
	Meta() ModuleMeta
	// Eligible reports whether the module can attempt to prove the finding
	// (precondition check). It must not touch the device.
	Eligible(ctx context.Context, env *core.Env, f *models.Finding) bool
	// Run executes the PoC against the live target and returns the result.
	Run(ctx context.Context, env *core.Env, f *models.Finding) (*Result, error)
}
