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

func TestAND007DebuggableApp(t *testing.T) {
	reg := newRegistry(t)
	findings := reg.Evaluate(context.Background(), rules.EvaluationContext{
		Apps: []models.Application{{PackageName: "com.example.app", Debuggable: true}},
	})
	if findByRuleID(findings, "AND-007") == nil {
		t.Fatal("AND-007 did not fire for debuggable app")
	}
}

func TestAND008OutdatedAndroid(t *testing.T) {
	reg := newRegistry(t)
	findings := reg.Evaluate(context.Background(), rules.EvaluationContext{
		Device: &models.DeviceInfo{AndroidVersion: "10", APILevel: "29"},
	})
	if findByRuleID(findings, "AND-008") == nil {
		t.Fatal("AND-008 did not fire for API 29")
	}

	reg = newRegistry(t)
	findings = reg.Evaluate(context.Background(), rules.EvaluationContext{
		Device: &models.DeviceInfo{AndroidVersion: "14", APILevel: "34"},
	})
	if findByRuleID(findings, "AND-008") != nil {
		t.Error("AND-008 fired for API 34")
	}
}

func TestAND009TestKeys(t *testing.T) {
	reg := newRegistry(t)
	findings := reg.Evaluate(context.Background(), rules.EvaluationContext{
		Device: &models.DeviceInfo{
			SystemProperties: map[string]string{"ro.build.tags": "test-keys"},
		},
	})
	f := findByRuleID(findings, "AND-009")
	if f == nil {
		t.Fatal("AND-009 did not fire for test-keys")
	}
	if f.Severity != models.SeverityHigh {
		t.Errorf("severity = %q, want high", f.Severity)
	}

	reg = newRegistry(t)
	findings = reg.Evaluate(context.Background(), rules.EvaluationContext{
		Device: &models.DeviceInfo{
			BuildFingerprint: "acme/x1/x1:14/TQ3A/user/release-keys",
		},
	})
	if findByRuleID(findings, "AND-009") != nil {
		t.Error("AND-009 fired for a release-keys build")
	}
}

func TestAND010ExcessivePermissions(t *testing.T) {
	reg := newRegistry(t)
	perms := []string{
		"android.permission.CAMERA",
		"android.permission.RECORD_AUDIO",
		"android.permission.ACCESS_FINE_LOCATION",
		"android.permission.READ_CONTACTS",
		"android.permission.READ_SMS",
	}
	findings := reg.Evaluate(context.Background(), rules.EvaluationContext{
		Apps: []models.Application{{PackageName: "com.example.app", Permissions: perms}},
	})
	f := findByRuleID(findings, "AND-010")
	if f == nil {
		t.Fatal("AND-010 did not fire for 5 dangerous permissions")
	}
	if f.Attributes["package"] != "com.example.app" {
		t.Errorf("package attribute = %q", f.Attributes["package"])
	}

	reg = newRegistry(t)
	findings = reg.Evaluate(context.Background(), rules.EvaluationContext{
		Apps: []models.Application{{PackageName: "com.example.app", Permissions: []string{
			"android.permission.CAMERA", "android.permission.INTERNET",
		}}},
	})
	if findByRuleID(findings, "AND-010") != nil {
		t.Error("AND-010 fired for only 1 dangerous permission")
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
