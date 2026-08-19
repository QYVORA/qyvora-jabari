package builtin

import (
	"context"
	"testing"
	"time"

	"github.com/QYVORA/qyvora-jabari/internal/rules"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
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

func TestAllRulesProduceEvidence(t *testing.T) {
	ctx := context.Background()
	ec := rules.EvaluationContext{
		Device: &models.DeviceInfo{
			DebugState:       "1",
			SecurityPatch:    "2023-01-01",
			RoAdbSecure:      "0",
			Rooted:           true,
			APILevel:         "28",
			AndroidVersion:   "9",
			BuildFingerprint: "test/test-keys",
			SystemProperties: map[string]string{"ro.build.tags": "test-keys"},
		},
		Apps: []models.Application{{
			PackageName:   "com.test.app",
			AllowBackup:   true,
			UsesCleartext: true,
			Debuggable:    true,
			Permissions:   []string{"android.permission.CAMERA", "android.permission.ACCESS_FINE_LOCATION", "android.permission.READ_CONTACTS", "android.permission.RECORD_AUDIO"},
		}},
	}

	for _, rule := range []rules.Rule{
		debuggableProductionRule,
		outdatedPatchRule,
		adbUnauthRule,
		rootedDeviceRule,
		backupEnabledRule,
		cleartextTrafficRule,
		debuggableAppRule,
		outdatedAndroidRule,
		testKeysRule,
		excessivePermsRule,
	} {
		findings, err := rule.Evaluate(ctx, ec)
		if err != nil {
			t.Fatalf("rule %s returned error: %v", rule.ID(), err)
		}
		for _, f := range findings {
			if len(f.Evidence) == 0 {
				t.Errorf("rule %s produced finding %q with no evidence", rule.ID(), f.Title)
			}
			if f.Impact == "" {
				t.Errorf("rule %s produced finding %q with no impact description", rule.ID(), f.Title)
			}
		}
	}
}

func TestManifestPropertyConfidenceIsMedium(t *testing.T) {
	ctx := context.Background()
	ec := rules.EvaluationContext{
		Device: &models.DeviceInfo{Rooted: true},
		Apps: []models.Application{{
			PackageName:   "com.test.app",
			AllowBackup:   true,
			UsesCleartext: true,
			Debuggable:    false,
			Permissions:   []string{"android.permission.CAMERA", "android.permission.ACCESS_FINE_LOCATION", "android.permission.READ_CONTACTS", "android.permission.RECORD_AUDIO"},
		}},
	}

	// AND-004 (rooted) should be medium confidence
	findings, _ := rootedDeviceRule.Evaluate(ctx, ec)
	if len(findings) == 0 {
		t.Fatal("AND-004 should fire for rooted device")
	}
	if findings[0].Confidence != models.ConfidenceMedium {
		t.Errorf("AND-004 confidence = %s, want Medium", findings[0].Confidence)
	}

	// AND-005 (allowBackup) should be medium confidence
	findings, _ = backupEnabledRule.Evaluate(ctx, ec)
	if len(findings) == 0 {
		t.Fatal("AND-005 should fire for allowBackup=true")
	}
	if findings[0].Confidence != models.ConfidenceMedium {
		t.Errorf("AND-005 confidence = %s, want Medium", findings[0].Confidence)
	}

	// AND-006 (cleartext) should be medium confidence
	findings, _ = cleartextTrafficRule.Evaluate(ctx, ec)
	if len(findings) == 0 {
		t.Fatal("AND-006 should fire for cleartext=true")
	}
	if findings[0].Confidence != models.ConfidenceMedium {
		t.Errorf("AND-006 confidence = %s, want Medium", findings[0].Confidence)
	}

	// AND-010 (excessive perms) should be medium confidence
	findings, _ = excessivePermsRule.Evaluate(ctx, ec)
	if len(findings) == 0 {
		t.Fatal("AND-010 should fire for 4+ dangerous perms")
	}
	if findings[0].Confidence != models.ConfidenceMedium {
		t.Errorf("AND-010 confidence = %s, want Medium", findings[0].Confidence)
	}
}

func TestAND001DoesNotFireOnProductionDevice(t *testing.T) {
	ctx := context.Background()
	ec := rules.EvaluationContext{
		Device: &models.DeviceInfo{DebugState: "0"},
	}
	findings, err := debuggableProductionRule.Evaluate(ctx, ec)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("AND-001 should not fire on production device, got %d findings", len(findings))
	}
}

func TestAND003DoesNotFireWhenSecured(t *testing.T) {
	ctx := context.Background()
	ec := rules.EvaluationContext{
		Device: &models.DeviceInfo{RoAdbSecure: "1"},
	}
	findings, err := adbUnauthRule.Evaluate(ctx, ec)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("AND-003 should not fire when ADB is secured, got %d findings", len(findings))
	}
}

func TestAND003DoesNotFireOnMissingData(t *testing.T) {
	ctx := context.Background()
	ec := rules.EvaluationContext{
		Device: &models.DeviceInfo{RoAdbSecure: ""},
	}
	findings, err := adbUnauthRule.Evaluate(ctx, ec)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("AND-003 should not fire on missing data, got %d findings", len(findings))
	}
}

func TestAND005DoesNotFireWhenBackupDisabled(t *testing.T) {
	ctx := context.Background()
	ec := rules.EvaluationContext{
		Apps: []models.Application{{PackageName: "com.test", AllowBackup: false}},
	}
	findings, err := backupEnabledRule.Evaluate(ctx, ec)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("AND-005 should not fire when backup disabled, got %d findings", len(findings))
	}
}

func TestAND006DoesNotFireWithoutCleartext(t *testing.T) {
	ctx := context.Background()
	ec := rules.EvaluationContext{
		Apps: []models.Application{{PackageName: "com.test", UsesCleartext: false}},
	}
	findings, err := cleartextTrafficRule.Evaluate(ctx, ec)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("AND-006 should not fire without cleartext, got %d findings", len(findings))
	}
}

func TestAND007DoesNotFireOnNonDebuggableApp(t *testing.T) {
	ctx := context.Background()
	ec := rules.EvaluationContext{
		Apps: []models.Application{{PackageName: "com.test", Debuggable: false}},
	}
	findings, err := debuggableAppRule.Evaluate(ctx, ec)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("AND-007 should not fire on non-debuggable app, got %d findings", len(findings))
	}
}

func TestAND008DoesNotFireOnCurrentDevice(t *testing.T) {
	ctx := context.Background()
	ec := rules.EvaluationContext{
		Device: &models.DeviceInfo{APILevel: "34", AndroidVersion: "14"},
	}
	findings, err := outdatedAndroidRule.Evaluate(ctx, ec)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("AND-008 should not fire on current device, got %d findings", len(findings))
	}
}

func TestAND010DoesNotFireWithFewPermissions(t *testing.T) {
	ctx := context.Background()
	ec := rules.EvaluationContext{
		Apps: []models.Application{{
			PackageName: "com.test",
			Permissions: []string{"android.permission.CAMERA"},
		}},
	}
	findings, err := excessivePermsRule.Evaluate(ctx, ec)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("AND-010 should not fire with <4 dangerous perms, got %d findings", len(findings))
	}
}
