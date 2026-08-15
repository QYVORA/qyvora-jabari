package poc

import (
	"context"
	"strings"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// execShell runs a shell command on the live target through the session's
// transport. The command is delivered as a single adb "shell" invocation, so
// arguments are never shell-interpreted by the host.
func execShell(ctx context.Context, env *core.Env, args ...string) (models.Response, error) {
	return env.Transport.Execute(ctx, models.Request{Command: "shell", Args: args})
}

// candidateApps returns the ordered list of package names a debuggability PoC
// should try: the finding's own package first (when present), then any app
// flagged as debuggable, then every other installed app. Order is preserved
// and duplicates are dropped.
func candidateApps(env *core.Env, f *models.Finding) []string {
	var out []string
	seen := map[string]bool{}
	add := func(pkg string) {
		if pkg == "" || seen[pkg] {
			return
		}
		seen[pkg] = true
		out = append(out, pkg)
	}

	// The package the finding is about is the most likely candidate.
	if f != nil {
		add(f.Attributes["package"])
	}
	for _, app := range env.Apps {
		if app.Debuggable {
			add(app.PackageName)
		}
	}
	for _, app := range env.Apps {
		add(app.PackageName)
	}
	return out
}

// appByPackage returns the inventory entry for pkg, or the zero value.
func appByPackage(env *core.Env, pkg string) (models.Application, bool) {
	for _, app := range env.Apps {
		if app.PackageName == pkg {
			return app, true
		}
	}
	return models.Application{}, false
}

// evidenceHasProp reports whether the finding carries evidence with the given
// source and content value (property-style evidence from the analysis stage).
func evidenceHasProp(f *models.Finding, source, value string) bool {
	if f == nil {
		return false
	}
	for _, ev := range f.Evidence {
		if ev.Kind == models.KindConfiguration &&
			strings.TrimSpace(ev.Source) == source &&
			strings.TrimSpace(ev.Content) == value {
			return true
		}
	}
	return false
}
