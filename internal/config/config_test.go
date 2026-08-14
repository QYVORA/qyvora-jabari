package config

import (
	"testing"
)

func TestEnvBindingNestedKeys(t *testing.T) {
	t.Setenv("QYVORA_PROFILE", "deep")
	t.Setenv("QYVORA_REPORT_DIR", "/tmp/rpt")
	t.Setenv("QYVORA_LOG_LEVEL", "warn")

	v, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := v.GetString("profile"); got != "deep" {
		t.Errorf("profile = %q, want deep", got)
	}
	if got := v.GetString("report.dir"); got != "/tmp/rpt" {
		t.Errorf("report.dir = %q, want /tmp/rpt", got)
	}
	if got := v.GetString("log.level"); got != "warn" {
		t.Errorf("log.level = %q, want warn", got)
	}
}
