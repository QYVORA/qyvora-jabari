package reporting

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/anomalyco/qyvora-jabari/internal/banner"
	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

func sampleSession() *models.Session {
	s := models.NewSession()
	s.ID = "sess-test"
	s.TargetID = "tgt-test"
	s.Profile = "standard"
	s.AddFinding(&models.Finding{
		ID:             "fnd-1",
		Title:          "ADB Unauthenticated Access",
		Category:       "device-exposure",
		Severity:       models.SeverityCritical,
		Confidence:     models.ConfidenceConfirmed,
		Status:         models.StatusConfirmed,
		RuleID:         "AND-003",
		Description:    "ADB without device-side authorization.",
		Recommendation: "Disable ADB on production devices.",
		Evidence: []models.Evidence{{
			Kind:    models.KindConfiguration,
			Source:  "ro.adb.secure",
			Content: "0",
		}},
	})
	return s
}

func render(t *testing.T, format Format) string {
	t.Helper()
	var buf bytes.Buffer
	w := &Writer{Format: format, Out: &buf}
	if err := w.Render(context.Background(), sampleSession()); err != nil {
		t.Fatalf("Render(%s): %v", format, err)
	}
	return buf.String()
}

func TestRenderTerminal(t *testing.T) {
	out := render(t, FormatTerminal)
	for _, want := range []string{"Critical", "1", "ADB Unauthenticated Access"} {
		if !strings.Contains(out, want) {
			t.Errorf("terminal output missing %q", want)
		}
	}
	// The report header must be the canonical brand banner, not custom text.
	if !strings.Contains(out, banner.Art) {
		t.Error("terminal output missing the canonical brand banner")
	}
}

func TestRenderJSON(t *testing.T) {
	out := render(t, FormatJSON)
	if !strings.Contains(out, `"id": "sess-test"`) {
		t.Errorf("json output missing session id")
	}
}

func TestRenderMarkdown(t *testing.T) {
	out := render(t, FormatMarkdown)
	for _, want := range []string{"# Android Security Assessment Report", "## Findings", "ADB Unauthenticated Access", "**Recommendation:**"} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown output missing %q", want)
		}
	}
}

func TestRenderHTML(t *testing.T) {
	out := render(t, FormatHTML)
	for _, want := range []string{"<!DOCTYPE html>", "<title>Android Security Assessment Report</title>", "ro.adb.secure"} {
		if !strings.Contains(out, want) {
			t.Errorf("html output missing %q", want)
		}
	}
}

func TestParseFormatUnknown(t *testing.T) {
	if _, err := ParseFormat("exe"); err == nil {
		t.Error("ParseFormat(exe) should fail")
	}
}
