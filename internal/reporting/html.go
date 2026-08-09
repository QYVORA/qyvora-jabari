package reporting

import (
	"html"
	"io"
	"time"

	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// renderHTML writes a self-contained HTML report (no external assets).
func renderHTML(w io.Writer, s *models.Session) error {
	writef(w, "<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	writef(w, "<meta charset=\"utf-8\">\n")
	writef(w, "<title>Android Security Assessment Report</title>\n")
	writef(w, "<style>body{font-family:sans-serif;max-width:64rem;margin:2rem auto;padding:0 1rem;color:#1a1a2e}"+
		"h1{border-bottom:2px solid #1a1a2e}table{border-collapse:collapse;width:100%%}td,th{"+
		"text-align:left;padding:.35rem .6rem;border-bottom:1px solid #ddd}"+
		".critical{color:#b00020}.high{color:#d00000}.medium{color:#b26a00}.low,.informational{color:#555}"+
		"code{background:#f2f2f2;padding:.1rem .3rem}</style>\n</head>\n<body>\n")

	writef(w, "<h1>Android Security Assessment Report</h1>\n")
	writef(w, "<p>Session <code>%s</code> &middot; Target <code>%s</code> &middot; Profile <code>%s</code></p>\n",
		html.EscapeString(s.ID), html.EscapeString(s.TargetID), html.EscapeString(s.Profile))
	writef(w, "<p>Started %s", s.Start.Format(time.RFC3339))
	if !s.End.IsZero() {
		writef(w, " &middot; Ended %s", s.End.Format(time.RFC3339))
	}
	writef(w, "</p>\n")

	writef(w, "<h2>Executive Summary</h2>\n")
	counts := tally(s)
	writef(w, "<table><tr><th>Severity</th><th>Count</th></tr>\n")
	writef(w, "<tr><td>Critical</td><td>%d</td></tr>\n", counts.Critical)
	writef(w, "<tr><td>High</td><td>%d</td></tr>\n", counts.High)
	writef(w, "<tr><td>Medium</td><td>%d</td></tr>\n", counts.Medium)
	writef(w, "<tr><td>Low</td><td>%d</td></tr>\n", counts.Low)
	writef(w, "<tr><td>Informational</td><td>%d</td></tr>\n", counts.Informational)
	writef(w, "</table>\n")

	writef(w, "<h2>Findings</h2>\n")
	if len(s.Findings) == 0 {
		writef(w, "<p>No findings.</p>\n")
	}
	for _, f := range s.Findings {
		if f == nil {
			continue
		}
		writef(w, "<h3 class=\"%s\">%s</h3>\n", html.EscapeString(string(f.Severity)), html.EscapeString(f.Title))
		writef(w, "<p><strong>Severity:</strong> <span class=\"%s\">%s</span> &middot; "+
			"<strong>Confidence:</strong> %s &middot; <strong>Status:</strong> %s &middot; "+
			"<strong>Rule:</strong> %s</p>\n",
			html.EscapeString(string(f.Severity)), html.EscapeString(string(f.Severity)),
			html.EscapeString(string(f.Confidence)), html.EscapeString(string(f.Status)),
			html.EscapeString(f.RuleID))
		writef(w, "<p>%s</p>\n", html.EscapeString(f.Description))
		if f.Recommendation != "" {
			writef(w, "<p><strong>Recommendation:</strong> %s</p>\n", html.EscapeString(f.Recommendation))
		}
		if len(f.Evidence) > 0 {
			writef(w, "<p><strong>Evidence:</strong></p><ul>\n")
			for _, ev := range f.Evidence {
				if ev.Content != "" {
					writef(w, "<li><code>%s</code>: <code>%s</code></li>\n",
						html.EscapeString(ev.Source), html.EscapeString(ev.Content))
				}
			}
			writef(w, "</ul>\n")
		}
	}

	writef(w, "</body>\n</html>\n")
	return nil
}
