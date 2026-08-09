// Package rules implements the vulnerability rule engine. Security checks are
// modeled as independently testable Rule implementations instead of being
// hardcoded into analysis modules, so the framework can grow new checks
// without touching existing code.
package rules

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// EvaluationContext carries everything a rule may need to evaluate a target.
// Rules must tolerate sparse data: every field may be empty or nil, and a
// well-behaved rule simply declines to fire rather than panicking.
type EvaluationContext struct {
	// Target is the target under assessment.
	Target *models.Target
	// Device is the discovered device metadata, when available.
	Device *models.DeviceInfo
	// Apps is the installed-application inventory, when available.
	Apps []models.Application
}

// Rule is a single reusable security check.
type Rule interface {
	// ID is the stable unique identifier (for example "AND-001").
	ID() string
	// Name is a short human-readable title.
	Name() string
	// Category groups the rule (for example "application-security").
	Category() string
	// Description explains what the rule detects and why it matters.
	Description() string
	// Severity is the default severity of findings this rule produces.
	Severity() models.Severity
	// MitreRefs lists MITRE ATT&CK for Mobile technique IDs, if any.
	MitreRefs() []string
	// Evaluate runs detection logic and returns zero or more findings.
	Evaluate(ctx context.Context, ec EvaluationContext) ([]models.Finding, error)
}

// Registry holds the set of registered rules and evaluates them against an
// evaluation context. It is safe for concurrent use.
type Registry struct {
	mu    sync.RWMutex
	rules map[string]Rule
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{rules: map[string]Rule{}}
}

// Register adds a rule. It rejects duplicate identifiers so rules cannot
// silently shadow each other.
func (r *Registry) Register(rule Rule) error {
	if rule == nil {
		return fmt.Errorf("cannot register a nil rule")
	}
	if rule.ID() == "" {
		return fmt.Errorf("rule %s has an empty ID", rule.Name())
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.rules[rule.ID()]; exists {
		return fmt.Errorf("rule %s already registered", rule.ID())
	}
	r.rules[rule.ID()] = rule
	return nil
}

// Get returns a rule by ID.
func (r *Registry) Get(id string) (Rule, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rule, ok := r.rules[id]
	return rule, ok
}

// All returns every registered rule.
func (r *Registry) All() []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Rule, 0, len(r.rules))
	for _, rule := range r.rules {
		out = append(out, rule)
	}
	return out
}

// Evaluate runs every registered rule against the context and returns the
// collected findings. A rule error is recorded as a finding diagnostic rather
// than aborting the entire assessment, so one broken rule cannot stop the run.
func (r *Registry) Evaluate(ctx context.Context, ec EvaluationContext) []models.Finding {
	var findings []models.Finding
	for _, rule := range r.All() {
		found, err := rule.Evaluate(ctx, ec)
		if err != nil {
			findings = append(findings, models.Finding{
				ID:          models.NewID("fnd"),
				TargetID:    targetID(ec),
				Title:       fmt.Sprintf("rule %s failed to evaluate", rule.ID()),
				Category:    "diagnostic",
				Description: err.Error(),
				Severity:    models.SeverityLow,
				Confidence:  models.ConfidenceMedium,
				Status:      models.StatusInformational,
				RuleID:      rule.ID(),
				Timestamp:   time.Now().UTC(),
			})
			continue
		}
		for i := range found {
			f := &found[i]
			if f.ID == "" {
				f.ID = models.NewID("fnd")
			}
			if f.TargetID == "" {
				f.TargetID = targetID(ec)
			}
			if f.Severity == "" {
				f.Severity = rule.Severity()
			}
			if f.Status == "" {
				f.Status = models.StatusDetected
			}
			if f.Confidence == "" {
				f.Confidence = models.ConfidenceMedium
			}
			f.RuleID = rule.ID()
			f.MitreRefs = append(f.MitreRefs, rule.MitreRefs()...)
			if f.Timestamp.IsZero() {
				f.Timestamp = time.Now().UTC()
			}
			findings = append(findings, *f)
		}
	}
	return findings
}

func targetID(ec EvaluationContext) string {
	if ec.Target != nil {
		return ec.Target.ID
	}
	return ""
}
