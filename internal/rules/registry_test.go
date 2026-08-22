package rules

import (
	"context"
	"testing"

	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// countingRule is a test rule that records how many times it was evaluated.
type countingRule struct {
	calls int
}

func (r *countingRule) ID() string                { return "TEST-001" }
func (r *countingRule) Name() string              { return "test rule" }
func (r *countingRule) Category() string          { return "test" }
func (r *countingRule) Description() string       { return "test rule for registry" }
func (r *countingRule) Severity() models.Severity { return models.SeverityMedium }
func (r *countingRule) MitreRefs() []string       { return nil }
func (r *countingRule) Evaluate(_ context.Context, _ EvaluationContext) ([]models.Finding, error) {
	r.calls++
	return []models.Finding{
		{
			ID:          models.NewID("fnd"),
			Title:       "test finding",
			Category:    "test",
			Severity:    models.SeverityMedium,
			Confidence:  models.ConfidenceConfirmed,
			Status:      models.StatusDetected,
			Description: "produced by counting rule",
		},
	}, nil
}

func TestRegistryEvaluate(t *testing.T) {
	reg := NewRegistry()
	rule := &countingRule{}
	if err := reg.Register(rule); err != nil {
		t.Fatalf("Register: %v", err)
	}

	findings := reg.Evaluate(context.Background(), EvaluationContext{})
	if len(findings) != 1 {
		t.Fatalf("Evaluate returned %d findings, want 1", len(findings))
	}
	f := findings[0]
	if f.RuleID != "TEST-001" {
		t.Errorf("RuleID = %q, want TEST-001", f.RuleID)
	}
	if f.ID == "" {
		t.Error("finding ID is empty; registry should assign one")
	}
	if f.Timestamp.IsZero() {
		t.Error("finding timestamp is zero; registry should set it")
	}
}

func TestRegistryDuplicateRejected(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(&countingRule{}); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := reg.Register(&countingRule{}); err == nil {
		t.Error("duplicate registration should fail")
	}
}

func TestRegistryErrorTolerated(t *testing.T) {
	reg := NewRegistry()
	good := &countingRule{}
	bad := &failingRule{}
	if err := reg.Register(good); err != nil {
		t.Fatalf("Register good: %v", err)
	}
	if err := reg.Register(bad); err != nil {
		t.Fatalf("Register bad: %v", err)
	}

	findings := reg.Evaluate(context.Background(), EvaluationContext{})
	// One good finding plus one diagnostic for the failing rule.
	if len(findings) != 2 {
		t.Fatalf("Evaluate returned %d findings, want 2 (1 real + 1 diagnostic)", len(findings))
	}
	// Locate the diagnostic by its rule ID rather than by index, so the
	// assertion is order-independent. The registry guarantees deterministic
	// rule order, but the test must not depend on that ordering to be valid.
	var diagnostic *models.Finding
	for i := range findings {
		if findings[i].RuleID == "TEST-BAD" {
			diagnostic = &findings[i]
			break
		}
	}
	if diagnostic == nil {
		t.Fatal("no diagnostic finding for the failing rule was produced")
	}
	if diagnostic.Status != models.StatusInformational {
		t.Errorf("diagnostic status = %q, want informational", diagnostic.Status)
	}
}

func TestRegistryAllSortedByID(t *testing.T) {
	reg := NewRegistry()
	ids := []string{"TEST-002", "TEST-001", "TEST-010", "TEST-003"}
	for _, id := range ids {
		r := &failingRule{}
		r.id = id
		if err := reg.Register(r); err != nil {
			t.Fatalf("Register %s: %v", id, err)
		}
	}
	got := reg.All()
	if len(got) != len(ids) {
		t.Fatalf("All returned %d rules, want %d", len(got), len(ids))
	}
	want := []string{"TEST-001", "TEST-002", "TEST-003", "TEST-010"}
	for i := range want {
		if got[i].ID() != want[i] {
			t.Fatalf("All order = %q, want deterministic sorted order %q",
				[]string{got[0].ID(), got[1].ID(), got[2].ID(), got[3].ID()}, want)
		}
	}
}

type failingRule struct {
	countingRule
	id string
}

func (r *failingRule) ID() string {
	if r.id != "" {
		return r.id
	}
	return "TEST-BAD"
}

func (r *failingRule) Evaluate(_ context.Context, _ EvaluationContext) ([]models.Finding, error) {
	return nil, &evaluationFailure{}
}

type evaluationFailure struct{}

func (e *evaluationFailure) Error() string { return "boom" }
