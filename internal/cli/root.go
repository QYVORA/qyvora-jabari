// Package cli implements the jabari command-line interface. The same binary
// is also published as "androidsec". The package wires configuration,
// logging, output formatting, and the target manager together and exposes the
// assessment commands.
package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/QYVORA/qyvora-jabari/internal/config"
	errs "github.com/QYVORA/qyvora-jabari/internal/errors"
	"github.com/QYVORA/qyvora-jabari/internal/logger"
	"github.com/QYVORA/qyvora-jabari/internal/output"
	"github.com/QYVORA/qyvora-jabari/internal/reporting"
	"github.com/QYVORA/qyvora-jabari/internal/target"
	"github.com/QYVORA/qyvora-jabari/internal/version"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)
var updateFlag bool


var (
	cfgFile    string
	verbose    bool
	quiet      bool
	outputFmt  string
	jsonOut    bool
	eventsFlag string
	dryRun     bool
	timeout    time.Duration

	cfg     *viper.Viper
	log     *logger.Logger
	printer *output.Printer
	targets *target.Manager

	// initErr records a fatal configuration/flag-validation failure that
	// occurs inside cobra's OnInitialize hook, which cannot return an error.
	// Execute() turns it into a usage error (exit code 2).
	initErr error
)

const appDescription = `jabari is a terminal-first CLI for authorized Android security
assessment, attack-surface analysis, vulnerability validation, and
evidence-driven reporting across USB-connected and specified-network
Android targets.

Usage modes:
  jabari assess usb          assess a connected authorized device
  jabari assess ip <addr>    assess one specific authorized Android device
  jabari target usb          select a connected device as the current target
  jabari enumerate           enumerate the current target
  jabari analyze             run the rule engine against the current target
  jabari validate            confirm detected findings on the current target
  jabari poc                 run proof-of-concept modules against live findings
  jabari report              render the latest assessment report

Every assessment requires explicit target authorization. jabari is scoped,
reversible, logged, and intended for use only on systems you are authorized
to assess.`

var rootCmd = &cobra.Command{
	Use:           "jabari",
	Short:         "Authorized Android security assessment framework",
	Long:          appDescription,
	Version:       version.String(),
	SilenceUsage:  true,
	SilenceErrors: true,
	// Validate shared flag/config state before any command runs so an
	// invalid --output value is rejected as a usage error (exit code 2)
	// instead of executing the command first.
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if initErr != nil {
			return errs.NewExitError(2, initErr.Error())
		}
		// --events stdout owns stdout: report renders, dry-run text and
		// command output must route to stderr so stdout carries exactly the
		// JSONL stream. Combining the stream with a machine report format is
		// a usage error (exit 2) — a consumer cannot split two machine
		// streams on the same pipe.
		if eventsFlag == "stdout" {
			machine := printer.Format() != output.FormatTerminal
			if !machine {
				if f, err := resolveReportFormat(); err == nil {
					machine = f != reporting.FormatTerminal
				}
			}
			if machine {
				return errs.NewExitError(2,
					"cannot combine --events stdout with a machine report format; use --events stderr or --events <file>")
			}
			stdoutWriter = os.Stderr
			printer.SetWriter(os.Stderr)
			cmd.SetOut(os.Stderr)
		}
		return nil
	},
	// Running "jabari" with no subcommand drops into the interactive
	// Metasploit-style console. One-shot commands remain available both at
	// the shell and as console commands.
	// Unknown subcommand names are usage errors (exit 2), matching the flag
	// handling above; this validator replaces cobra's legacyArgs default so
	// the exit code stays under QYVORA contract control.
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) > 0 {
			return errs.NewExitError(2, fmt.Sprintf("unknown command %q (try 'jabari --help')", args[0]))
		}
		return nil
	},
}

// Execute runs the root command against os.Args and returns the process exit
// code. It never calls os.Exit itself so callers control process termination.
// The command context is bound to SIGINT/SIGTERM so assessments cancel
// cleanly.
func Execute() int {
	return ExecuteArgs(os.Args[1:])
}

// ExecuteArgs runs the root command with an explicit argument vector and
// returns the process exit code. It exists so tests can exercise exit-code
// behavior without spawning processes.
func ExecuteArgs(args []string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return ExecuteArgsContext(ctx, args)
}

// ExecuteArgsContext runs the root command with an explicit argument vector
// under a caller-supplied context and returns the process exit code.
//
// The interactive TUI needs this form. It runs commands in-process on its own
// goroutine and must be able to cancel a single execution without tearing down
// the process, so the work is driven by a context the caller owns rather than
// by process-wide signal handling. That distinction is what makes Ctrl+C cancel
// the operation instead of the interface.
func ExecuteArgsContext(ctx context.Context, args []string) int {
	rootCmd.SetArgs(args)

	// If --update is passed, route to the update subcommand regardless of
	// other positional arguments.
	for _, a := range args {
		if a == "--update" || a == "-update" || a == "--update=true" {
			rootCmd.SetArgs([]string{"update"})
			break
		}
	}

	rootCmd.SetContext(ctx)
	if err := rootCmd.Execute(); err != nil {
		var exitErr *errs.ExitError
		if errors.As(err, &exitErr) {
			fmt.Fprintln(os.Stderr, wrapErr(exitErr.Message))
			if exitErr.Cause != nil {
				fmt.Fprintln(os.Stderr, "  "+exitErr.Cause.Error())
			}
			return exitErr.Code
		}
		fmt.Fprintln(os.Stderr, wrapErr(err.Error()))
		return 1
	}
	if initErr != nil {
		fmt.Fprintln(os.Stderr, wrapErr(initErr.Error()))
		return 2
	}
	return 0
}

func init() {
	// The default action opens the interactive TUI. It is assigned here rather
	// than in the rootCmd literal because Go's initialisation dependency
	// analysis follows references through function bodies: runTUI reaches
	// rootCmd, so naming it inside rootCmd's own initialiser is a cycle, while
	// init() is exempt from that analysis.
	rootCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runTUI(cmd.Root(), cmd.Context())
	}
	rootCmd.AddCommand(commandTUI())
	rootCmd.AddCommand(newConsoleCommand())

	cobra.OnInitialize(initConfig)

	// Flag-parse failures (unknown flag, bad value) are usage errors and must
	// exit 2 per the shared QYVORA exit-code contract, not the runtime code 1.
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return errs.NewExitError(2, err.Error())
	})

	pf := rootCmd.PersistentFlags()
	pf.BoolVar(&updateFlag, "update", false, "update the CLI to the latest official release")
	pf.StringVarP(&cfgFile, "config", "c", "", "config file (default $HOME/.config/qyvora/jabari/config.yaml")
	pf.BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	pf.BoolVarP(&quiet, "quiet", "q", false, "suppress non-error output")
	pf.StringVarP(&outputFmt, "output", "o", "", "output format: terminal, json, markdown, html, yaml")
	pf.BoolVar(&jsonOut, "json", false, "output in JSON format (shorthand for --output json")
	pf.StringVar(&eventsFlag, "events", "", "emit a machine-readable JSONL event stream to stdout, stderr, or a file path")
	pf.BoolVar(&dryRun, "dry-run", false, "validate target and print the assessment plan without executing")

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newUpdatesCmd())
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.AddCommand(newCapabilitiesCmd())
	rootCmd.AddCommand(newToolsCmd())
	rootCmd.AddCommand(newTargetCmd())
	rootCmd.AddCommand(newAssessCmd())
	rootCmd.AddCommand(newStageCmd("discover", "Identify the current target", runDiscover))
	rootCmd.AddCommand(newStageCmd("enumerate", "Inventory applications on the current target", runEnumerate))
	rootCmd.AddCommand(newStageCmd("analyze", "Evaluate rules against the current target", runAnalyze))
	rootCmd.AddCommand(newStageCmd("validate", "Confirm detected findings on the current target", runValidate))
	rootCmd.AddCommand(newPocCmd())
	rootCmd.AddCommand(newReportCmd())

	rootCmd.SetVersionTemplate(fmt.Sprintf("jabari %s\n", version.String()))
}

// initConfig loads configuration and initializes the shared logger, printer,
// and target manager. A config file that cannot be parsed is fatal.
func initConfig() {
	v, err := config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}
	cfg = v

	initLogger()
	initPrinter()
	targets = target.NewManager()
}

func initLogger() {
	log = logger.New()
	log.SetLevel(logger.ParseLevel(cfg.GetString("log.level")))

	if verbose || cfg.GetBool("verbose") {
		log.SetVerbose(true)
	}
	if quiet || cfg.GetBool("quiet") {
		log.SetQuiet(true)
	}
}

func initPrinter() {
	printer = output.New()

	// Precedence: explicit --output/-o flag, then --json shorthand, then
	// config (json/output keys), then terminal. The flag default is empty so
	// a non-empty outputFmt always means the user passed it explicitly.
	format := "terminal"
	switch {
	case outputFmt != "":
		format = outputFmt
	case jsonOut:
		format = "json"
	case cfg.GetBool("json"):
		format = "json"
	case cfg.IsSet("output"):
		if v, ok := cfg.Get("output").(string); ok && v != "" {
			format = v
		}
	}

	parsed, err := output.ParseFormat(format)
	if err != nil {
		initErr = err
		return
	}
	printer.SetFormat(parsed)
	// ANSI color is a terminal-only nicety: disable it when stdout is not an
	// interactive device or when the caller opts out via NO_COLOR, so no
	// escape sequences leak into redirected or piped output.
	color.NoColor = !stdoutIsTerminal() || os.Getenv("NO_COLOR") != ""
}

// stdoutIsTerminal reports whether standard output is an interactive
// character device.
func stdoutIsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// requireTarget returns the current target or fails with a helpful message
// pointing at the target selection commands.
func requireTarget() (*models.Target, error) {
	t := targets.Current()
	if t == nil {
		return nil, errs.NewExitError(2, "no target selected; run 'jabari target usb' or 'jabari target ip <addr>' first")
	}
	if !t.Authorized() {
		return nil, errs.NewExitError(2, "current target is not authorized: "+t.DisplayName())
	}
	return t, nil
}

func wrapErr(msg string) string {
	return color.New(color.FgRed, color.Bold).Sprint("Error: ") + msg
}