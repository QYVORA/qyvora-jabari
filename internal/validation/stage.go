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

	"github.com/anomalyco/qyvora-jabari/internal/core"
	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// Stage confirms Detected findings whose evidence can be re-checked by a
// read-only property query.
type Stage struct{}

// Name returns the stage name used in logs and reports.
func (s *Stage) Name() string { return "validation" }

// Run re-checks property-based evidence on a live transport. Findings whose
// evidence has no live property (for example static APK findings) are left at
// their current status.
func (s *Stage) Run(ctx context.Context, env *core.Env) error {
	if env.Transport == nil || env.Session == nil {
		return nil
	}

	var confirmed, unchanged int
	for _, f := range env.Session.Findings {
		if f.Status != models.StatusDetected {
			continue
		}
		properties := propertyEvidence(f)
		if len(properties) == 0 {
			continue
		}
		ok := true
		for _, prop := range properties {
			resp, err := env.Transport.Execute(ctx, models.Request{
				Command: "shell",
				Args:    []string{"getprop", prop.key},
			})
			if err != nil {
				ok = false
				break
			}
			if strings.TrimSpace(string(resp.Stdout)) != prop.value {
				ok = false
				break
			}
		}
		if ok {
			f.Status = models.StatusConfirmed
			f.Confidence = models.ConfidenceConfirmed
			confirmed++
		} else {
			unchanged++
		}
	}

	if env.Log != nil {
		env.Log.Info("validation confirmed %d finding(s), left %d as detected", confirmed, unchanged)
	}
	return nil
}

type propRef struct {
	key   string
	value string
}

// propertyEvidence extracts "ro.*" system property references from a
// finding's evidence. Only configuration evidence with a property-style
// source is eligible for live re-validation.
func propertyEvidence(f *models.Finding) []propRef {
	var out []propRef
	for _, ev := range f.Evidence {
		if ev.Kind != models.KindConfiguration {
			continue
		}
		key := strings.TrimSpace(ev.Source)
		if !strings.HasPrefix(key, "ro.") {
			continue
		}
		out = append(out, propRef{key: key, value: strings.TrimSpace(ev.Content)})
	}
	return out
}
