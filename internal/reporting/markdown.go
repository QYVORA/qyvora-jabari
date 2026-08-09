package reporting

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// renderJSON writes the full session as indented JSON. JSON is the canonical
// machine-readable format and the foundation for ecosystem integration.
func renderJSON(w io.Writer, s *models.Session) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}

// renderMarkdown writes a human-readable Markdown report.
func renderMarkdown(w io.Writer, s *models.Session) error {
	bw := &mdWriter{w: w}
	bw.title("Android Security Assessment Report")
	bw.line()
	bw.kv("Session", s.ID)
	bw.kv("Target", s.TargetID)
	bw.kv("Profile", s.Profile)
	bw.kv("Started", s.Start.Format(time.RFC3339))
	if !s.End.IsZero() {
		bw.kv("Ended", s.End.Format(time.RFC3339))
	}

	bw.section("Executive Summary")
	counts := tally(s)
	bw.kv("Findings", strconv.Itoa(counts.Tally()))
	bw.kv("Critical", strconv.Itoa(counts.Critical))
	bw.kv("High", strconv.Itoa(counts.High))
	bw.kv("Medium", strconv.Itoa(counts.Medium))
	bw.kv("Low", strconv.Itoa(counts.Low))
	bw.kv("Informational", strconv.Itoa(counts.Informational))

	bw.section("Findings")
	if len(s.Findings) == 0 {
		bw.text("No findings.")
	}
	for _, f := range s.Findings {
		if f == nil {
			continue
		}
		bw.heading3(f.Title)
		bw.kv("Severity", string(f.Severity))
		bw.kv("Confidence", string(f.Confidence))
		bw.kv("Status", string(f.Status))
		bw.kv("Rule", f.RuleID)
		bw.kv("Description", f.Description)
		if f.Impact != "" {
			bw.kv("Impact", f.Impact)
		}
		if f.Recommendation != "" {
			bw.kv("Recommendation", f.Recommendation)
		}
		if len(f.Evidence) > 0 {
			bw.text("**Evidence:**")
			for _, ev := range f.Evidence {
				if ev.Content != "" {
					bw.codeBlock(ev.Source + ": " + ev.Content)
				}
			}
		}
	}
	return nil
}

func tally(s *models.Session) models.SeverityCounts {
	var c models.SeverityCounts
	for _, f := range s.Findings {
		if f == nil {
			continue
		}
		switch f.Severity {
		case models.SeverityCritical:
			c.Critical++
		case models.SeverityHigh:
			c.High++
		case models.SeverityMedium:
			c.Medium++
		case models.SeverityLow:
			c.Low++
		default:
			c.Informational++
		}
	}
	return c
}

// mdWriter is a tiny helper for emitting Markdown without an external
// dependency.
type mdWriter struct {
	w io.Writer
}

func (m *mdWriter) title(t string)    { writef(m.w, "# %s\n\n", t) }
func (m *mdWriter) section(s string)  { writef(m.w, "## %s\n\n", s) }
func (m *mdWriter) heading3(s string) { writef(m.w, "### %s\n", s) }
func (m *mdWriter) text(s string)     { writef(m.w, "%s\n\n", s) }
func (m *mdWriter) line()             { writef(m.w, "\n") }
func (m *mdWriter) kv(k, v string)    { writef(m.w, "- **%s:** %s\n", k, v) }

func (m *mdWriter) codeBlock(content string) {
	writef(m.w, "```\n%s\n```\n\n", content)
}

func writef(w io.Writer, format string, args ...any) {
	_, _ = io.WriteString(w, fmt.Sprintf(format, args...))
}
