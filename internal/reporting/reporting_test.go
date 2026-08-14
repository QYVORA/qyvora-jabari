package reporting

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/QYVORA/qyvora-jabari/internal/banner"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
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

func TestRenderYAML(t *testing.T) {
	out := render(t, FormatYAML)
	for _, want := range []string{"id: sess-test", "targetid: tgt-test", "ADB Unauthenticated Access"} {
		if !strings.Contains(out, want) {
			t.Errorf("yaml output missing %q", want)
		}
	}
	// YAML output must not leak ANSI escape sequences.
	if strings.Contains(out, "\x1b[") {
		t.Error("yaml output contains ANSI escape sequences")
	}
}

func TestParseFormatAliases(t *testing.T) {
	for alias, want := range map[string]Format{
		"terminal": FormatTerminal,
		"table":    FormatTerminal,
		"text":     FormatTerminal,
		"json":     FormatJSON,
		"markdown": FormatMarkdown,
		"html":     FormatHTML,
		"yaml":     FormatYAML,
	} {
		got, err := ParseFormat(alias)
		if err != nil {
			t.Errorf("ParseFormat(%q): %v", alias, err)
			continue
		}
		if got != want {
			t.Errorf("ParseFormat(%q) = %q, want %q", alias, got, want)
		}
	}
}

func TestParseFormatUnknown(t *testing.T) {
	if _, err := ParseFormat("exe"); err == nil {
		t.Error("ParseFormat(exe) should fail")
	}
}

func TestRenderJSONDeterministic(t *testing.T) {
	s := sampleSession()
	renderOnce := func() string {
		t.Helper()
		var buf bytes.Buffer
		w := &Writer{Format: FormatJSON, Out: &buf}
		if err := w.Render(context.Background(), s); err != nil {
			t.Fatalf("Render(json): %v", err)
		}
		return buf.String()
	}
	first := renderOnce()
	second := renderOnce()
	if first != second {
		t.Error("JSON render of the same session is not deterministic across runs")
	}
	// Machine-readable output must never contain ANSI escape sequences.
	if strings.Contains(first, "\x1b[") {
		t.Error("JSON output contains ANSI escape sequences")
	}
}

// TestRenderJSONCarriesRiskScore verifies that a persisted risk score
// survives through the report pipeline into the machine-readable JSON.
func TestRenderJSONCarriesRiskScore(t *testing.T) {
	s := sampleSession()
	s.RiskScore = 87
	s.RiskLevel = "critical"
	var buf bytes.Buffer
	w := &Writer{Format: FormatJSON, Out: &buf}
	if err := w.Render(context.Background(), s); err != nil {
		t.Fatalf("Render(json): %v", err)
	}
	out := buf.String()
	for _, want := range []string{`"risk_score": 87`, `"risk_level": "critical"`} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON output missing %q", want)
		}
	}
}

// TestRenderTerminalCarriesRiskScore verifies the score is surfaced in the
// human-readable terminal report.
func TestRenderTerminalCarriesRiskScore(t *testing.T) {
	s := sampleSession()
	s.RiskScore = 62
	s.RiskLevel = "high"
	var buf bytes.Buffer
	w := &Writer{Format: FormatTerminal, Out: &buf}
	if err := w.Render(context.Background(), s); err != nil {
		t.Fatalf("Render(terminal): %v", err)
	}
	if !strings.Contains(buf.String(), "62/100 (high)") {
		t.Error("terminal report missing the risk score line")
	}
}
