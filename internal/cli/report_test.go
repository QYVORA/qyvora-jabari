package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/QYVORA/qyvora-jabari/internal/reporting"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// TestResolveReportFormatPrecedence verifies the documented precedence:
// report --format flag, then --output/-o flag, then --json, then config,
// then terminal default.
func TestResolveReportFormatPrecedence(t *testing.T) {
	reset := func() {
		reportFlags.format = ""
		outputFmt = ""
		jsonOut = false
	}
	reset()
	defer reset()

	if got, err := resolveReportFormat(); err != nil || got != reporting.FormatTerminal {
		t.Fatalf("default format = %q, %v; want terminal", got, err)
	}

	outputFmt = "json"
	if got, err := resolveReportFormat(); err != nil || got != reporting.FormatJSON {
		t.Fatalf("-o json format = %q, %v; want json", got, err)
	}
	outputFmt = ""

	jsonOut = true
	if got, err := resolveReportFormat(); err != nil || got != reporting.FormatJSON {
		t.Fatalf("--json format = %q, %v; want json", got, err)
	}
	jsonOut = false

	reportFlags.format = "markdown"
	outputFmt = "json"
	if got, err := resolveReportFormat(); err != nil || got != reporting.FormatMarkdown {
		t.Fatalf("--format precedence = %q, %v; want markdown", got, err)
	}

	reportFlags.format = "bogus"
	if _, err := resolveReportFormat(); err == nil {
		t.Error("resolveReportFormat(bogus) should fail")
	}
}

// TestRenderSessionJSONHonorsOutput verifies that renderSession with -o json
// emits a JSON document containing the session identifier.
func TestRenderSessionJSONHonorsOutput(t *testing.T) {
	oldStdout := stdoutWriter
	oldFmt := outputFmt
	var buf bytes.Buffer
	stdoutWriter = &buf
	outputFmt = "json"
	defer func() {
		stdoutWriter = oldStdout
		outputFmt = oldFmt
	}()

	s := models.NewSession()
	s.ID = "sess-json-test"
	if err := renderSession(context.Background(), s); err != nil {
		t.Fatalf("renderSession: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("renderSession(-o json) did not emit valid JSON: %v\n%s", err, buf.String())
	}
	if doc["id"] != "sess-json-test" {
		t.Errorf("JSON id = %v, want sess-json-test", doc["id"])
	}
}
