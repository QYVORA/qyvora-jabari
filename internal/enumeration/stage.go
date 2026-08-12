// Package enumeration implements the assessment stage that builds the
// application inventory of a target device. It answers "what exists on this
// Android device" for the applications dimension.
package enumeration

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/anomalyco/qyvora-jabari/internal/core"
	"github.com/anomalyco/qyvora-jabari/internal/transport"
	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// DetailLimitDefault caps how many apps get the (slower) per-package detail
// pass during one assessment. Listing package names is cheap; dumpsys per
// package is not, so the limit keeps device assessments bounded.
const DetailLimitDefault = 100

// DetailWorkersDefault is the default concurrency for the per-package detail
// pass. Every package is queried with its own adb subprocess, so the calls
// are independent and safe to run in parallel; the cap keeps the adb server
// and the device from being saturated.
const DetailWorkersDefault = 8

// Stage enumerates installed applications through the transport.
type Stage struct {
	// DetailLimit overrides DetailLimitDefault when non-zero.
	DetailLimit int
	// DetailWorkers overrides DetailWorkersDefault when non-zero.
	DetailWorkers int
}

// Name returns the stage name used in logs and reports.
func (s *Stage) Name() string { return "enumeration" }

// Run lists third-party packages and gathers metadata for each up to the
// detail limit. Packages beyond the limit are still counted but not expanded.
// The per-package detail pass runs concurrently (bounded by DetailWorkers)
// because each package is a separate, independent adb call.
func (s *Stage) Run(ctx context.Context, env *core.Env) error {
	if env.Transport == nil {
		return transport.ErrNotConnected
	}

	limit := DetailLimitDefault
	if s.DetailLimit > 0 {
		limit = s.DetailLimit
	}
	workers := DetailWorkersDefault
	if s.DetailWorkers > 0 {
		workers = s.DetailWorkers
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
	if len(packages) > limit {
		packages = packages[:limit]
	}

	apps := make([]models.Application, len(packages))
	if workers <= 1 {
		for i, pkg := range packages {
			if err := ctx.Err(); err != nil {
				return err
			}
			apps[i] = s.appDetailOrName(ctx, env, pkg)
		}
	} else {
		s.detailConcurrent(ctx, env, packages, apps, workers)
	}

	env.Apps = apps
	env.Session.Apps = apps
	return nil
}

// detailConcurrent fills apps with per-package metadata using a bounded pool
// of workers. Each package is written to its own slice slot, so no shared
// state needs locking beyond the worker accounting.
func (s *Stage) detailConcurrent(ctx context.Context, env *core.Env, packages []string, apps []models.Application, workers int) {
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i, pkg := range packages {
		if err := ctx.Err(); err != nil {
			return
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, pkg string) {
			defer wg.Done()
			defer func() { <-sem }()
			apps[i] = s.appDetailOrName(ctx, env, pkg)
		}(i, pkg)
	}
	wg.Wait()
}

// appDetailOrName returns the parsed detail for a package, or a name-only
// entry when the detail pass fails. A single unreadable package should not
// abort the inventory.
func (s *Stage) appDetailOrName(ctx context.Context, env *core.Env, pkg string) models.Application {
	app, err := s.appDetail(ctx, env, pkg)
	if err != nil {
		return models.Application{PackageName: pkg}
	}
	return app
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
