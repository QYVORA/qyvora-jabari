package models

import (
	"testing"
	"time"
)

func dedupTestFinding(ruleID, title, pkg string) *Finding {
	return &Finding{
		ID:         NewID("fnd"),
		Title:      title,
		Category:   "application-security",
		RuleID:     ruleID,
		Severity:   SeverityHigh,
		Confidence: ConfidenceMedium,
		Status:     StatusDetected,
		Timestamp:  time.Now().UTC(),
		Attributes: map[string]string{"package": pkg},
	}
}

func TestFingerprintStableAndDistinct(t *testing.T) {
	a := dedupTestFinding("AND-001", "Debuggable application", "com.example.app")
	b := dedupTestFinding("AND-001", "Debuggable application", "com.example.app")
	if a.Fingerprint() != b.Fingerprint() {
		t.Fatalf("identical findings have different fingerprints: %q vs %q", a.Fingerprint(), b.Fingerprint())
	}

	// Different package = different issue.
	c := dedupTestFinding("AND-001", "Debuggable application", "com.other.app")
	if a.Fingerprint() == c.Fingerprint() {
		t.Fatal("findings for different packages must not share a fingerprint")
	}
	// Different rule = different issue even for the same condition wording.
	d := dedupTestFinding("AND-002", "Debuggable application", "com.example.app")
	if a.Fingerprint() == d.Fingerprint() {
		t.Fatal("findings from different rules must not share a fingerprint")
	}
	// Attribute insertion order must not affect the fingerprint.
	e := dedupTestFinding("AND-001", "Debuggable application", "com.example.app")
	e.Attributes["exported"] = "true"
	f := dedupTestFinding("AND-001", "Debuggable application", "com.example.app")
	f.Attributes["exported"] = "true"
	if e.Fingerprint() != f.Fingerprint() {
		t.Fatal("attribute insertion order changed the fingerprint")
	}
}

func TestSessionDeduplicatesIdenticalFindings(t *testing.T) {
	s := NewSession()
	first := dedupTestFinding("AND-001", "Debuggable application", "com.example.app")
	second := dedupTestFinding("AND-001", "Debuggable application", "com.example.app")

	s.AddFinding(first)
	s.AddFinding(second)

	if len(s.Findings) != 1 {
		t.Fatalf("session kept %d findings, want 1 (duplicate not collapsed)", len(s.Findings))
	}
	if s.Findings[0].ID != first.ID {
		t.Fatalf("kept duplicate %s, want first occurrence %s", s.Findings[0].ID, first.ID)
	}
	for _, f := range s.Findings {
		if f.SessionID != s.ID {
			t.Fatalf("finding %s has SessionID %q, want %q", f.ID, f.SessionID, s.ID)
		}
	}
}

func TestSessionKeepsGenuinelyDifferentFindings(t *testing.T) {
	s := NewSession()
	s.AddFinding(dedupTestFinding("AND-001", "Debuggable application", "com.example.app"))
	s.AddFinding(dedupTestFinding("AND-001", "Debuggable application", "com.other.app"))
	s.AddFinding(dedupTestFinding("AND-002", "Exported component", "com.example.app"))
	s.AddFinding(dedupTestFinding("AND-002", "Backup allowed", "com.example.app"))

	if len(s.Findings) != 4 {
		t.Fatalf("session kept %d findings, want 4 (over-deduplication)", len(s.Findings))
	}
}

func TestMergeEvidenceDedupesByHashAndUpgradesConfidence(t *testing.T) {
	ev := Evidence{Kind: KindConfiguration, Source: "dumpsys", Content: "debuggable=true"}
	ev.Hash = HashContent([]byte(ev.Content))

	low := dedupTestFinding("AND-001", "Debuggable application", "com.example.app")
	low.Confidence = ConfidenceLow
	low.AddEvidence(ev)

	dup := dedupTestFinding("AND-001", "Debuggable application", "com.example.app")
	dup.Confidence = ConfidenceConfirmed
	dup.AddEvidence(ev) // same hash — must not be appended twice

	high := dedupTestFinding("AND-001", "Debuggable application", "com.example.app")
	high.Confidence = ConfidenceConfirmed
	unique := Evidence{Kind: KindManifest, Source: "AndroidManifest.xml", Content: "android:debuggable=\"true\""}
	unique.Hash = HashContent([]byte(unique.Content))
	high.AddEvidence(unique)

	s := NewSession()
	s.AddFinding(low)
	s.AddFinding(dup)
	s.AddFinding(high)

	if len(s.Findings) != 1 {
		t.Fatalf("session kept %d findings, want 1", len(s.Findings))
	}
	merged := s.Findings[0]
	if got := len(merged.Evidence); got != 2 {
		t.Fatalf("merged finding carries %d evidence items, want 2 (hash-deduped)", got)
	}
	if merged.Confidence != ConfidenceConfirmed {
		t.Fatalf("merged confidence = %q, want confirmed (strongest wins)", merged.Confidence)
	}
}
