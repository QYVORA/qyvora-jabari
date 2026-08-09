// Package enumeration implements the assessment stage that builds the
// application inventory of a target device. It answers "what exists on this
// Android device" for the applications dimension.
package enumeration

import (
	"context"
	"fmt"
	"strings"

	"github.com/anomalyco/qyvora-jabari/internal/core"
	"github.com/anomalyco/qyvora-jabari/internal/transport"
	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// DetailLimitDefault caps how many apps get the (slower) per-package detail
// pass during one assessment. Listing package names is cheap; dumpsys per
// package is not, so the limit keeps device assessments bounded.
const DetailLimitDefault = 100

// Stage enumerates installed applications through the transport.
type Stage struct {
	// DetailLimit overrides DetailLimitDefault when non-zero.
	DetailLimit int
}

// Name returns the stage name used in logs and reports.
func (s *Stage) Name() string { return "enumeration" }

// Run lists third-party packages and gathers metadata for each up to the
// detail limit. Packages beyond the limit are still counted but not expanded.
func (s *Stage) Run(ctx context.Context, env *core.Env) error {
	if env.Transport == nil {
		return transport.ErrNotConnected
	}

	limit := DetailLimitDefault
	if s.DetailLimit > 0 {
		limit = s.DetailLimit
	}

	resp, err := env.Transport.Execute(ctx, models.Request{
		Command: "shell",
		Args:    []string{"pm", "list", "packages", "-3"},
	})
	if err != nil {
		return fmt.Errorf("listing packages: %w", err)
	}

	packages := parsePackageList(string(resp.Stdout))
	if env.Log != nil {
		env.Log.Info("enumerated %d third-party packages", len(packages))
	}

	apps := make([]models.Application, 0, len(packages))
	for i, pkg := range packages {
		if err := ctx.Err(); err != nil {
			return err
		}
		if i >= limit {
			break
		}
		app, dumpErr := s.appDetail(ctx, env, pkg)
		if dumpErr != nil {
			// A single unreadable package should not abort the inventory;
			// record it as a package entry with just the name.
			app = models.Application{PackageName: pkg}
		}
		apps = append(apps, app)
	}

	env.Apps = apps
	env.Session.Apps = apps
	return nil
}

// appDetail fetches and parses the dumpsys package listing for one package.
func (s *Stage) appDetail(ctx context.Context, env *core.Env, pkg string) (models.Application, error) {
	resp, err := env.Transport.Execute(ctx, models.Request{
		Command: "shell",
		Args:    []string{"dumpsys", "package", pkg},
	})
	if err != nil {
		return models.Application{}, err
	}
	return parsePackageDump(string(resp.Stdout), pkg), nil
}

// parsePackageList extracts package names from "pm list packages" output,
// skipping the "package:" prefix on each matching line.
func parsePackageList(out string) []string {
	var pkgs []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package:") {
			pkgs = append(pkgs, strings.TrimPrefix(line, "package:"))
		}
	}
	return pkgs
}
