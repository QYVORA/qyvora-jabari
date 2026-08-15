package poc

import (
	"context"
	"strings"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// exportedActivity proves that an exported activity can be launched from
// outside the application. It starts the first exported activity of the target
// package with `am start` and verifies the resulting process is running.
//
// Launching an activity changes on-screen state, so this module is high-risk
// and only runs with explicit operator opt-in.
type exportedActivity struct{}

const (
	exportedModuleID   = "android.exported_activity"
	exportedModuleName = "Exported Activity Launch Proof"
)

func init() {
	Register(&exportedActivity{})
}

func (m *exportedActivity) Meta() ModuleMeta {
	return ModuleMeta{
		ID:          exportedModuleID,
		Name:        exportedModuleName,
		Category:    "application",
		Risk:        RiskHigh,
		Description: "Proves an exported activity is launchable by an external caller via am start.",
		Preconditions: "An application finding naming a package whose inventory entry lists " +
			"activities. Requires explicit high-risk opt-in.",
	}
}

func (m *exportedActivity) Eligible(_ context.Context, env *core.Env, f *models.Finding) bool {
	if env.Transport == nil {
		return false
	}
	pkg := f.Attributes["package"]
	if pkg == "" {
		return false
	}
	app, ok := appByPackage(env, pkg)
	return ok && len(app.Activities) > 0
}

func (m *exportedActivity) Run(ctx context.Context, env *core.Env, f *models.Finding) (*Result, error) {
	pkg := f.Attributes["package"]
	app, ok := appByPackage(env, pkg)
	if !ok || len(app.Activities) == 0 {
		return notProven(exportedModuleID, f.ID, string(RiskHigh),
			"no activities known for "+pkg), nil
	}
	activity := app.Activities[0]
	component := componentName(pkg, activity)

	resp, err := execShell(ctx, env, "am", "start", "-n", component)
	if err != nil {
		return notProven(exportedModuleID, f.ID, string(RiskHigh),
			"am start failed ("+err.Error()+")"), nil
	}
	startOut := strings.TrimSpace(string(resp.Stdout))

	// Verify the process is running after launch; pidof is read-only.
	pidResp, err := execShell(ctx, env, "pidof", pkg)
	if err != nil {
		pidResp = models.Response{}
	}
	pid := strings.TrimSpace(string(pidResp.Stdout))
	if pid == "" {
		return notProven(exportedModuleID, f.ID, string(RiskHigh),
			component+" was launched but no process is running"), nil
	}
	return proven(exportedModuleID, f.ID, string(RiskHigh),
		component+" was launched externally (pid "+pid+")",
		[]string{"am start -n " + component + " -> " + startOut, "pidof " + pkg + " -> " + pid}), nil
}

// componentName normalizes an activity name into an am start component. A bare
// class name becomes ".Class" relative to the package; names that already
// carry a dot (fully-qualified or ".relative") pass through.
func componentName(pkg, activity string) string {
	activity = strings.TrimSpace(activity)
	if activity == "" {
		return pkg + "/."
	}
	if strings.Contains(activity, "/") {
		return activity
	}
	if strings.HasPrefix(activity, ".") || strings.HasPrefix(activity, pkg+".") {
		return pkg + "/" + activity
	}
	return pkg + "/." + activity
}
