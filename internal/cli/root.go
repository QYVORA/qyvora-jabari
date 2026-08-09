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

	"github.com/anomalyco/qyvora-jabari/internal/config"
	errs "github.com/anomalyco/qyvora-jabari/internal/errors"
	"github.com/anomalyco/qyvora-jabari/internal/logger"
	"github.com/anomalyco/qyvora-jabari/internal/output"
	"github.com/anomalyco/qyvora-jabari/internal/target"
	"github.com/anomalyco/qyvora-jabari/internal/version"
	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

var (
	cfgFile   string
	verbose   bool
	quiet     bool
	outputFmt string
	jsonOut   bool
	timeout   time.Duration

	cfg     *viper.Viper
	log     *logger.Logger
	printer *output.Printer
	targets *target.Manager
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
}

// Execute runs the root command and returns the process exit code. It never
// calls os.Exit itself so callers control process termination. The command
// context is bound to SIGINT/SIGTERM so assessments cancel cleanly.
func Execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
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
	return 0
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default $HOME/.qyvora-jabari/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress non-error output")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "table", "output format: table, json, text, yaml")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output in JSON format (shorthand for --output json)")
	rootCmd.PersistentFlags().DurationVarP(&timeout, "timeout", "t", 30*time.Second, "default timeout for device operations")

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.AddCommand(newTargetCmd())
	rootCmd.AddCommand(newAssessCmd())
	rootCmd.AddCommand(newStageCmd("discover", "Identify the current target", runDiscover))
	rootCmd.AddCommand(newStageCmd("enumerate", "Inventory applications on the current target", runEnumerate))
	rootCmd.AddCommand(newStageCmd("analyze", "Evaluate rules against the current target", runAnalyze))
	rootCmd.AddCommand(newStageCmd("validate", "Confirm detected findings on the current target", runValidate))
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
	format := outputFmt

	if jsonOut || cfg.GetBool("json") {
		format = "json"
	} else if cfg.IsSet("output") {
		if v, ok := cfg.Get("output").(string); ok {
			format = v
		}
	}

	parsed, err := output.ParseFormat(format)
	if err != nil {
		log.Warn("invalid output format %q, using table", format)
		parsed = output.FormatTable
	}
	printer.SetFormat(parsed)
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
