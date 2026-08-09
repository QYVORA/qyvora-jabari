// Package reporting renders completed sessions into human-readable and
// machine-readable formats. Every report is derived from the session model so
// a session can be re-rendered into any format at any time.
package reporting

import (
	"context"
	"fmt"
	"io"

	"github.com/anomalyco/qyvora-jabari/internal/core"
	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// Format identifies a supported report format.
type Format string

const (
	// FormatTerminal prints a compact summary to stdout.
	FormatTerminal Format = "terminal"
	// FormatJSON emits the full session as JSON.
	FormatJSON Format = "json"
	// FormatMarkdown emits a Markdown report.
	FormatMarkdown Format = "markdown"
	// FormatHTML emits a self-contained HTML report.
	FormatHTML Format = "html"
)

// ParseFormat resolves a user-supplied format name.
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case FormatTerminal, FormatJSON, FormatMarkdown, FormatHTML:
		return Format(s), nil
	default:
		return "", fmt.Errorf("unsupported report format %q", s)
	}
}

// Writer renders a session to an io.Writer in a given format.
type Writer struct {
	Format Format
	Out    io.Writer
}

// Render writes the session report.
func (w *Writer) Render(ctx context.Context, s *models.Session) error {
	switch w.Format {
	case FormatJSON:
		return renderJSON(w.Out, s)
	case FormatMarkdown:
		return renderMarkdown(w.Out, s)
	case FormatHTML:
		return renderHTML(w.Out, s)
	default:
		return renderTerminal(w.Out, s)
	}
}

// Stage is the reporting pipeline stage. It renders the session using the
// configured writer once the pipeline has finished.
type Stage struct {
	Writer *Writer
}

// Name returns the stage name used in logs and reports.
func (s *Stage) Name() string { return "reporting" }

// Run renders the finished session. Without a writer the stage is a no-op so
// callers can run pipelines without producing reports.
func (s *Stage) Run(ctx context.Context, env *core.Env) error {
	if s.Writer == nil || env.Session == nil {
		return nil
	}
	return s.Writer.Render(ctx, env.Session)
}
