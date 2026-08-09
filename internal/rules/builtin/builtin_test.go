package builtin

import (
	"context"
	"testing"
	"time"

	"github.com/anomalyco/qyvora-jabari/internal/rules"
	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

func newRegistry(t *testing.T) *rules.Registry {
	t.Helper()
	reg := rules.NewRegistry()
	if err := Register(reg); err != nil {
		t.Fatalf("Register builtin: %v", err)
	}
	return reg
}

func findByRuleID(findings []models.Finding, id string) *models.Finding {
	for i := range findings {
		if findings[i].RuleID == id {
			return &findings[i]
		}
	}
	return nil
}

func TestAND001DebuggableDevice(t *testing.T) {
	reg := newRegistry(t)
	findings := reg.Evaluate(context.Background(), rules.EvaluationContext{
		Device: &models.DeviceInfo{DebugState: "1"},
	})
	f := findByRuleID(findings, "AND-001")
	if f == nil {
		t.Fatal("AND-001 did not fire for ro.debuggable=1")
	}
	if f.Severity != models.SeverityHigh {
		t.Errorf("severity = %q, want high", f.Severity)
	}
}

func TestAND001NonDebugDevice(t *testing.T) {
	reg := newRegistry(t)
	findings := reg.Evaluate(context.Background(), rules.EvaluationContext{
		Device: &models.DeviceInfo{DebugState: "0"},
	})
	if findByRuleID(findings, "AND-001") != nil {
		t.Error("AND-001 fired for ro.debuggable=0")
	}
}

func TestAND002OutdatedPatch(t *testing.T) {
	reg := newRegistry(t)
	old := time.Now().AddDate(0, -12, 0).Format("2006-01-02")
	findings := reg.Evaluate(context.Background(), rules.EvaluationContext{
		Device: &models.DeviceInfo{SecurityPatch: old},
	})
	if findByRuleID(findings, "AND-002") == nil {
		t.Errorf("AND-002 did not fire for patch %s", old)
	}
}

func TestAND002CurrentPatch(t *testing.T) {
	reg := newRegistry(t)
	recent := time.Now().Format("2006-01-02")
	findings := reg.Evaluate(context.Background(), rules.EvaluationContext{
		Device: &models.DeviceInfo{SecurityPatch: recent},
	})
	if findByRuleID(findings, "AND-002") != nil {
		t.Error("AND-002 fired for a current patch level")
	}
}

func TestAND003ADBSecureDisabled(t *testing.T) {
	reg := newRegistry(t)
	findings := reg.Evaluate(context.Background(), rules.EvaluationContext{
		Device: &models.DeviceInfo{RoAdbSecure: "0"},
	})
	f := findByRuleID(findings, "AND-003")
	if f == nil {
		t.Fatal("AND-003 did not fire for ro.adb.secure=0")
	}
	if f.Severity != models.SeverityCritical {
		t.Errorf("severity = %q, want critical", f.Severity)
	}
}

func TestAND005BackupEnabled(t *testing.T) {
	reg := newRegistry(t)
	findings := reg.Evaluate(context.Background(), rules.EvaluationContext{
		Apps: []models.Application{{PackageName: "com.example.app", AllowBackup: true}},
	})
	f := findByRuleID(findings, "AND-005")
	if f == nil {
		t.Fatal("AND-005 did not fire for allowBackup=true")
	}
	if f.Attributes["package"] != "com.example.app" {
		t.Errorf("package attribute = %q", f.Attributes["package"])
	}
}

func TestRulesTolerateNoData(t *testing.T) {
	reg := newRegistry(t)
	findings := reg.Evaluate(context.Background(), rules.EvaluationContext{})
	for _, f := range findings {
		if f.Status == models.StatusInformational && f.RuleID == "" {
			t.Errorf("unexpected diagnostic finding with no rule: %s", f.Title)
		}
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings with empty context, got %d", len(findings))
	}
}
