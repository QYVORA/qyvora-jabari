package poc

import (
	"context"
	"strings"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// runAsDebuggable proves that a debuggable application (or a debuggable build
// with ro.debuggable=1) lets an external caller execute commands inside the
// app's uid context via `run-as`. The check is read-only: `run-as <pkg> id`
// prints the app's uid and never modifies the device.
type runAsDebuggable struct{}

const (
	runAsModuleID   = "android.run_as_debuggable"
	runAsModuleName = "Debuggable App Run-As Proof"
)

func init() {
	Register(&runAsDebuggable{})
}

func (m *runAsDebuggable) Meta() ModuleMeta {
	return ModuleMeta{
		ID:          runAsModuleID,
		Name:        runAsModuleName,
		Category:    "application",
		Risk:        RiskMedium,
		Description: "Proves a debuggable app (or ro.debuggable=1 build) allows an external caller to execute commands as the app via run-as.",
		Preconditions: "A finding for ro.debuggable=1 or a debuggable app, and an installed app " +
			"that run-as can enter.",
	}
}

func (m *runAsDebuggable) Eligible(_ context.Context, env *core.Env, f *models.Finding) bool {
	if env.Transport == nil {
		return false
	}
	if f.RuleID == "AND-001" || f.RuleID == "AND-007" {
		return true
	}
	return evidenceHasProp(f, "ro.debuggable", "1")
}

func (m *runAsDebuggable) Run(ctx context.Context, env *core.Env, f *models.Finding) (*Result, error) {
	pkgs := candidateApps(env, f)
	if len(pkgs) == 0 {
		return notProven(runAsModuleID, f.ID, string(RiskMedium),
			"no installed app identified to escalate through"), nil
	}
	for _, pkg := range pkgs {
		resp, err := execShell(ctx, env, "run-as", pkg, "id")
		if err != nil {
			// run-as on an unauthorized package returns an error; keep trying
			// other candidates.
			continue
		}
		out := strings.TrimSpace(string(resp.Stdout))
		if resp.OK() && strings.Contains(out, "uid=") {
			return proven(runAsModuleID, f.ID, string(RiskMedium),
				"run-as "+pkg+" executed as the application user",
				[]string{"run-as " + pkg + " id -> " + out}), nil
		}
	}
	return notProven(runAsModuleID, f.ID, string(RiskMedium),
		"run-as could not execute as any candidate app"), nil
}
