// Package validation implements the assessment stage that confirms suspected
// issues with safe, non-destructive checks.
//
// The foundation validation pass re-reads the exact system properties that
// a finding's evidence references and only upgrades the finding to Confirmed
// when the live value still matches the recorded evidence. This keeps
// validation honest without touching the device beyond read-only queries.
// Runtime and offensive validation (Part B) extend this stage later.
package validation

import (
	"context"
	"strings"
	"sync"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// ValidateWorkersDefault bounds how many findings are re-checked in parallel.
// Each check is a separate adb getprop call, independent and safe to run
// concurrently; the cap keeps the adb server responsive during validation.
const ValidateWorkersDefault = 8

// Stage confirms Detected findings whose evidence can be re-checked by a
// read-only property query.
type Stage struct{}

// Name returns the stage name used in logs and reports.
func (s *Stage) Name() string { return "validation" }

// Run re-checks property-based evidence on a live transport. Findings whose
// evidence has no live property (for example static APK findings) are left at
// their current status. Each property re-read is an independent adb call, so
// eligible findings are checked concurrently (bounded) to cut wall time.
func (s *Stage) Run(ctx context.Context, env *core.Env) error {
	if env.Transport == nil || env.Session == nil {
		return nil
	}

	var (
		confirmed, unchanged int
		mu                   sync.Mutex
		wg                   sync.WaitGroup
		sem                  = make(chan struct{}, ValidateWorkersDefault)
	)

	for i := range env.Session.Findings {
		f := env.Session.Findings[i]
		if f.Status != models.StatusDetected {
			continue
		}
		properties := propertyEvidence(f)
		if len(properties) == 0 {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			ok := s.confirmProperties(ctx, env, properties)
			mu.Lock()
			if ok {
				f.Status = models.StatusConfirmed
				f.Confidence = models.ConfidenceConfirmed
				confirmed++
			} else {
				unchanged++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if env.Log != nil {
		env.Log.Info("validation confirmed %d finding(s), left %d as detected", confirmed, unchanged)
	}
	return nil
}

// confirmProperties verifies every referenced system property still matches
// its recorded evidence value on the live transport.
func (s *Stage) confirmProperties(ctx context.Context, env *core.Env, properties []propRef) bool {
	for _, prop := range properties {
		resp, err := env.Transport.Execute(ctx, models.Request{
			Command: "shell",
			Args:    []string{"getprop", prop.key},
		})
		if err != nil {
			return false
		}
		if strings.TrimSpace(string(resp.Stdout)) != prop.value {
			return false
		}
	}
	return true
}

type propRef struct {
	key   string
	value string
}

// propertyEvidence extracts system property references from a finding's
// evidence. Configuration evidence whose source matches a known re-checkable
// prefix is eligible for live re-validation.
func propertyEvidence(f *models.Finding) []propRef {
	allowed := func(source string) bool {
		if strings.HasPrefix(source, "ro.") {
			return true
		}
		if strings.HasPrefix(source, "android:") {
			return true
		}
		switch source {
		case "usesCleartextTraffic", "permissions", "root_access":
			return true
		}
		return false
	}
	var out []propRef
	for _, ev := range f.Evidence {
		if ev.Kind != models.KindConfiguration {
			continue
		}
		key := strings.TrimSpace(ev.Source)
		if !allowed(key) {
			continue
		}
		out = append(out, propRef{key: key, value: strings.TrimSpace(ev.Content)})
	}
	return out
}
