package utilities

import "testing"

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"com.example.App": "com-example-App",
		"my file(1).apk":  "my-file-1-apk",
		"already-ok":      "already-ok",
		"":                "",
	}
	for in, want := range cases {
		if got := SanitizeName(in); got != want {
			t.Errorf("SanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("short", 10); got != "short" {
		t.Errorf("Truncate short = %q", got)
	}
	if got := Truncate("a very long string", 10); got != "a very ..." {
		t.Errorf("Truncate long = %q, want %q", got, "a very ...")
	}
}

func TestIndent(t *testing.T) {
	in := "line1\nline2"
	want := "  line1\n  line2"
	if got := Indent(in, "  "); got != want {
		t.Errorf("Indent = %q, want %q", got, want)
	}
	if got := Indent("", "  "); got != "" {
		t.Errorf("Indent empty = %q, want empty", got)
	}
}

func TestPluralize(t *testing.T) {
	if got := Pluralize(1, "finding", "findings"); got != "1 finding" {
		t.Errorf("Pluralize(1) = %q", got)
	}
	if got := Pluralize(3, "finding", "findings"); got != "3 findings" {
		t.Errorf("Pluralize(3) = %q", got)
	}
}
