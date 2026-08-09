package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	errs "github.com/anomalyco/qyvora-jabari/internal/errors"
	"github.com/anomalyco/qyvora-jabari/internal/reporting"
	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// reportFlags are shared by the report command.
var reportFlags struct {
	format string
	list   bool
}

// newReportCmd builds "jabari report [session-id]", which renders a saved
// session. With no argument the most recent session in the report directory
// is rendered. With --list, saved sessions are enumerated instead.
func newReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report [session-id]",
		Short: "Render an assessment report",
		Long: `Render a saved assessment session in the configured output format
(terminal, json, markdown, html).

With no session id, the most recent session in the report directory is used.
Sessions are written by "jabari assess" and the stage commands.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if reportFlags.list {
				return listSessions()
			}
			path, err := sessionPath(args)
			if err != nil {
				return err
			}
			session, err := loadSession(path)
			if err != nil {
				return errs.NewExitError(1, "loading session: "+err.Error())
			}
			return renderSession(cmd.Context(), session)
		},
	}
	cmd.Flags().StringVarP(&reportFlags.format, "format", "f", "",
		"report format: terminal, json, markdown, html (default from config)")
	cmd.Flags().BoolVar(&reportFlags.list, "list", false,
		"list saved sessions instead of rendering")
	return cmd
}

// listSessions prints the saved session files newest-first.
func listSessions() error {
	files, err := sortedSessionFiles(reportDir())
	if err != nil {
		return errs.NewExitError(2, err.Error())
	}
	if len(files) == 0 {
		return errs.NewExitError(2, "no saved sessions in "+reportDir()+" (run an assessment first)")
	}
	for _, f := range files {
		fmt.Println(filepath.Base(f))
	}
	return nil
}

// renderSession writes a loaded session in the CLI's configured report
// format. Precedence: --format flag, --json flag, report.format config,
// terminal default.
func renderSession(ctx context.Context, s *models.Session) error {
	format := reporting.FormatTerminal
	switch {
	case reportFlags.format != "":
		parsed, err := reporting.ParseFormat(reportFlags.format)
		if err != nil {
			return errs.NewExitError(2, err.Error())
		}
		format = parsed
	case jsonOut || cfg.GetBool("json"):
		format = reporting.FormatJSON
	case cfg.IsSet("report.format"):
		if v, ok := cfg.Get("report.format").(string); ok && v != "" {
			if parsed, err := reporting.ParseFormat(v); err == nil {
				format = parsed
			}
		}
	}
	w := &reporting.Writer{Format: format, Out: os.Stdout}
	return w.Render(ctx, s)
}

// sessionPath resolves the session file for the report command.
func sessionPath(args []string) (string, error) {
	if len(args) == 1 {
		id := args[0]
		if path := filepath.Join(reportDir(), "session-"+id+".json"); fileExists(path) {
			return path, nil
		}
		return "", errs.NewExitError(2, "session "+id+" not found in "+reportDir())
	}
	latest, err := latestSessionFile(reportDir())
	if err != nil {
		return "", errs.NewExitError(2, err.Error())
	}
	return latest, nil
}

// latestSessionFile finds the most recently modified session JSON in dir.
func latestSessionFile(dir string) (string, error) {
	files, err := sortedSessionFiles(dir)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no saved sessions in %s (run an assessment first)", dir)
	}
	return files[0], nil
}

// sortedSessionFiles returns session JSON paths in dir, newest first.
func sortedSessionFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("no saved sessions in %s (run an assessment first)", dir)
	}
	var matches []string
	for _, e := range entries {
		if !e.IsDir() && len(e.Name()) > len("session-") &&
			e.Name()[:len("session-")] == "session-" && filepath.Ext(e.Name()) == ".json" {
			matches = append(matches, filepath.Join(dir, e.Name()))
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no saved sessions in %s (run an assessment first)", dir)
	}
	sort.Slice(matches, func(i, j int) bool {
		si, _ := os.Stat(matches[i])
		sj, _ := os.Stat(matches[j])
		return si.ModTime().After(sj.ModTime())
	})
	return matches, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
