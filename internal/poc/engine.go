package poc

import (
	"context"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// Engine drives PoC modules against the session's findings. It is a thin
// executor: eligibility and the high-risk gate are applied here, and every
// executed module produces a structured Result that the stage persists.
type Engine struct {
	// AllowHighRisk permits modules that change device state. It must be
	// wired to explicit operator opt-in; the engine never defaults it on.
	AllowHighRisk bool
	// ModuleFilter, when non-empty, restricts execution to the given module
	// IDs (targeted runs). Empty means every registered module runs.
	ModuleFilter map[string]bool
}

// Run executes every eligible module against every eligible finding in the
// session and returns the collected results in deterministic order.
func (e *Engine) Run(ctx context.Context, env *core.Env) []Result {
	if env == nil || env.Session == nil || env.Transport == nil {
		return nil
	}
	var results []Result
	for _, f := range env.Session.Findings {
		if f == nil {
			continue
		}
		if f.Status == models.StatusFalsePositive || f.Status == models.StatusResolved {
			continue
		}
		for _, m := range List() {
			if err := ctx.Err(); err != nil {
				return results
			}
			meta := m.Meta()
			if len(e.ModuleFilter) > 0 && !e.ModuleFilter[meta.ID] {
				continue
			}
			if meta.Risk == RiskHigh && !e.AllowHighRisk {
				results = append(results, Result{
					Module:    meta.ID,
					FindingID: f.ID,
					Status:    models.PocSkipped,
					Risk:      string(meta.Risk),
					Summary:   "high-risk PoC requires explicit opt-in (poc.high_risk=true or --poc-high-risk)",
				})
				continue
			}
			if !m.Eligible(ctx, env, f) {
				continue
			}
			results = append(results, e.runModule(ctx, env, m, meta, f))
		}
	}
	return results
}

// runModule executes a single module and normalizes its outcome.
func (e *Engine) runModule(ctx context.Context, env *core.Env, m Module, meta ModuleMeta, f *models.Finding) Result {
	res, err := m.Run(ctx, env, f)
	if err != nil {
		return Result{
			Module:    meta.ID,
			FindingID: f.ID,
			Status:    models.PocError,
			Risk:      string(meta.Risk),
			Summary:   "PoC execution failed",
			Err:       err,
		}
	}
	if res == nil {
		res = &Result{
			Module:    meta.ID,
			FindingID: f.ID,
			Status:    models.PocNotProven,
			Summary:   "no proof captured",
		}
	}
	if res.Risk == "" {
		res.Risk = string(meta.Risk)
	}
	return *res
}

// proven builds a proven result with captured evidence.
func proven(module, findingID, risk, summary string, evidence []string) *Result {
	return &Result{
		Module: module, FindingID: findingID, Status: models.PocProven,
		Risk: risk, Summary: summary, Evidence: evidence,
	}
}

// notProven builds a negative result.
func notProven(module, findingID, risk, summary string) *Result {
	return &Result{
		Module: module, FindingID: findingID, Status: models.PocNotProven,
		Risk: risk, Summary: summary,
	}
}
