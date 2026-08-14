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
	writef(w, "%s\n", repeat("─", 46))
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
	return nil
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
