package reporting

import (
	"io"
	"strings"
	"time"

	"github.com/QYVORA/qyvora-jabari/internal/banner"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// renderTerminal prints a compact summary suitable for stdout. The header is
// the canonical brand banner (internal/banner), not hand-written wordmark
// text.
func renderTerminal(w io.Writer, s *models.Session) error {
	writef(w, "%s\n", strings.TrimRight(banner.Art, "\n"))
	writef(w, "\n")

	writef(w, "Session\n")
	writef(w, "  ID:      %s\n", s.ID)
	writef(w, "  Target:  %s\n", s.TargetID)
	writef(w, "  Profile: %s\n", s.Profile)
	writef(w, "  Started: %s\n", s.Start.Format(time.RFC3339))
	if s.RiskScore > 0 {
		writef(w, "  Risk:    %d/100 (%s)\n", s.RiskScore, s.RiskLevel)
	}
	writef(w, "\n")

	writef(w, "Findings\n")
	counts := tally(s)
	writef(w, "  %-14s %d\n", "Critical", counts.Critical)
	writef(w, "  %-14s %d\n", "High", counts.High)
	writef(w, "  %-14s %d\n", "Medium", counts.Medium)
	writef(w, "  %-14s %d\n", "Low", counts.Low)
	writef(w, "  %-14s %d\n", "Informational", counts.Informational)
	writef(w, "  %-14s %d\n", "Total", counts.Tally())
	writef(w, "\n")

	// A one-line list of the highest-severity findings so the summary is not
	// just numbers.
	var flagged int
	for _, f := range s.Findings {
		if f == nil {
			continue
		}
		if f.Severity == models.SeverityCritical || f.Severity == models.SeverityHigh {
			if flagged == 0 {
				writef(w, "Key findings\n")
			}
			writef(w, "  [%s] %s (%s)\n", strings.ToUpper(string(f.Severity)), f.Title, f.Status)
			flagged++
		}
	}
	if flagged == 0 {
		writef(w, "No critical or high findings.\n")
	}

	if len(s.Pocs) > 0 {
		writef(w, "\n")
		writef(w, "PoC runs\n")
		var proven, notProven, skipped, errors int
		for _, p := range s.Pocs {
			if p == nil {
				continue
			}
			switch p.Status {
			case models.PocProven:
				proven++
			case models.PocNotProven:
				notProven++
			case models.PocSkipped:
				skipped++
			default:
				errors++
			}
		}
		writef(w, "  %-14s %d\n", "Proven", proven)
		writef(w, "  %-14s %d\n", "Not proven", notProven)
		writef(w, "  %-14s %d\n", "Skipped", skipped)
		writef(w, "  %-14s %d\n", "Errors", errors)
		for _, p := range s.Pocs {
			if p == nil || p.Status != models.PocProven {
				continue
			}
			writef(w, "  [PROVEN] %s %s\n", p.Module, p.Summary)
		}
	}
	return nil
}
