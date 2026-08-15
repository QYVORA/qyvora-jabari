package models

import (
	"encoding/json"
	"testing"
)

func TestSeverityParseAndWeights(t *testing.T) {
	if ParseSeverity("High") != SeverityHigh {
		t.Error("ParseSeverity(High) mismatch")
	}
	if ParseSeverity("bogus") != SeverityInformational {
		t.Error("ParseSeverity(bogus) should default to informational")
	}
	if SeverityCritical.Weights() <= SeverityMedium.Weights() {
		t.Error("critical weight must exceed medium weight")
	}
}

func TestPermissionRisk(t *testing.T) {
	if PermissionRisk("android.permission.CAMERA") != "dangerous" {
		t.Error("CAMERA should classify as dangerous")
	}
	if PermissionRisk("android.permission.INTERNET") != "normal" {
		t.Error("INTERNET should classify as normal")
	}
	if PermissionRisk("com.example.CUSTOM") != "unknown" {
		t.Error("unknown permissions should classify as unknown")
	}
}

func TestRiskLevel(t *testing.T) {
	if RiskS1.Rank() != 1 || RiskS4.Rank() != 4 {
		t.Error("risk ranks are wrong")
	}
	if !RiskS3.RequiresRiskyFlag() || !RiskS4.RequiresRiskyFlag() {
		t.Error("S3/S4 must require the risky flag")
	}
	if RiskS1.RequiresRiskyFlag() || RiskS2.RequiresRiskyFlag() {
		t.Error("S1/S2 must not require the risky flag")
	}
}

func TestFindingAddEvidence(t *testing.T) {
	f := &Finding{Title: "x"}
	f.AddEvidence(Evidence{Kind: KindDevice, Source: "s"})
	if len(f.Evidence) != 1 {
		t.Error("AddEvidence did not append")
	}
}

func TestHashContentDeterministic(t *testing.T) {
	if HashContent([]byte("abc")) != HashContent([]byte("abc")) {
		t.Error("hash must be deterministic")
	}
	if HashContent([]byte("abc")) == HashContent([]byte("abd")) {
		t.Error("different content must hash differently")
	}
}

func TestSessionLifecycle(t *testing.T) {
	s := NewSession()
	if s.ID == "" {
		t.Error("NewSession must assign an ID")
	}
	f := &Finding{Title: "t"}
	s.AddFinding(f)
	if f.SessionID != s.ID {
		t.Error("AddFinding must attach the session ID")
	}
	if len(s.Findings) != 1 {
		t.Error("AddFinding must append")
	}
	s.Finish()
	if s.End.IsZero() {
		t.Error("Finish must record the end time")
	}
}

func TestTargetAuthorization(t *testing.T) {
	tgt := &Target{}
	if tgt.Authorized() {
		t.Error("default target must not be authorized")
	}
	tgt.Auth = Authorization{Granted: true}
	if !tgt.Authorized() {
		t.Error("granted target must be authorized")
	}
	if tgt.DisplayName() == "" {
		t.Error("DisplayName must never be empty")
	}
}

func TestDeviceSummary(t *testing.T) {
	d := &DeviceInfo{Manufacturer: "Acme", Model: "X1", AndroidVersion: "14", APILevel: "34"}
	got := d.Summary()
	if got != "Acme X1 Android 14 API 34" {
		t.Errorf("Summary = %q", got)
	}
	var nilD *DeviceInfo
	if nilD.Summary() != "no device information" {
		t.Error("nil device summary mismatch")
	}
}

func TestPocRunSerialization(t *testing.T) {
	s := NewSession()
	f := &Finding{ID: "fnd-1"}
	s.AddFinding(f)
	s.AddPoc(&PocRun{
		ID: "poc-1", Module: "android.run_as_debuggable", FindingID: f.ID,
		Status: PocProven, Risk: "medium", Summary: "proved",
		Evidence: []string{"run-as com.x id -> uid=1"},
	})
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Session
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back.Pocs) != 1 || !back.Pocs[0].Proven() || back.Pocs[0].Module != "android.run_as_debuggable" {
		t.Errorf("round-trip pocs = %+v", back.Pocs)
	}
}
