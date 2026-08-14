package output

import (
	"bytes"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"
)

func TestParseFormatAliases(t *testing.T) {
	for in, want := range map[string]Format{
		"terminal": FormatTerminal,
		"table":    FormatTerminal,
		"text":     FormatTerminal,
		"json":     FormatJSON,
		"yaml":     FormatYAML,
	} {
		got, err := ParseFormat(in)
		if err != nil {
			t.Errorf("ParseFormat(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseFormat(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseFormatUnknown(t *testing.T) {
	if _, err := ParseFormat("exe"); err == nil {
		t.Error("ParseFormat(exe) should fail")
	}
	if _, err := ParseFormat("markdown"); err == nil {
		t.Error("ParseFormat(markdown) should fail for the tools printer (not a supported printer format)")
	}
}

func TestPrinterYAMLValid(t *testing.T) {
	var buf bytes.Buffer
	p := New()
	p.SetWriter(&buf)
	p.SetFormat(FormatYAML)
	if p.Format() != FormatYAML {
		t.Fatalf("SetFormat(yaml) did not take effect: %q", p.Format())
	}
	p.Print(map[string]string{"tool": "jabari", "version": "dev"})

	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Error("yaml output contains ANSI escape sequences")
	}
	var decoded map[string]string
	if err := yaml.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("printed YAML is not valid YAML: %v\n%s", err, out)
	}
	if decoded["tool"] != "jabari" {
		t.Errorf("decoded tool = %q, want jabari", decoded["tool"])
	}
}

func TestPrinterJSONNoANSI(t *testing.T) {
	var buf bytes.Buffer
	p := New()
	p.SetWriter(&buf)
	p.SetFormat(FormatJSON)
	p.Print(map[string]string{"a": "b"})
	if strings.Contains(buf.String(), "\x1b[") {
		t.Error("JSON output contains ANSI escape sequences")
	}
}
