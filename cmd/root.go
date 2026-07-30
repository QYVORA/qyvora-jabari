package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/anomalyco/qyvora-jabari/internal/config"
	"github.com/anomalyco/qyvora-jabari/internal/logger"
	"github.com/anomalyco/qyvora-jabari/internal/output"
	"github.com/anomalyco/qyvora-jabari/internal/version"
)

var (
	cfgFile   string
	verbose   bool
	quiet     bool
	outputFmt string
	jsonOut   bool

	cfg      *viper.Viper
	log      *logger.Logger
	printer  *output.Printer
)

const appDescription = `QYVORA-Jabari is a terminal-first CLI application.

A professional-grade command-line tool for infrastructure security
assessment and network intelligence gathering.`

var rootCmd = &cobra.Command{
	Use:     "qyvora-jabari",
	Short:   "Terminal-first CLI for infrastructure security assessment",
	Long:    appDescription,
	Version: version.String(),
	RunE:    runRoot,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default $HOME/.qyvora-jabari/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress non-error output")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "table", "output format: table, json, text, yaml")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output in JSON format (shorthand for --output json)")

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newCompletionCmd())

	rootCmd.SetVersionTemplate(fmt.Sprintf("qyvora-jabari %s\n", version.String()))
}

func initConfig() {
	v, err := config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	cfg = v

	initLogger()
	initPrinter()
}

func initLogger() {
	log = logger.New()

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

func runRoot(cmd *cobra.Command, args []string) error {
	cmd.Help()
	return nil
}

func wrapErr(msg string) string {
	return color.New(color.FgRed, color.Bold).Sprint("Error: ") + msg
}

func isTableFormat() bool {
	return !jsonOut && !cfg.GetBool("json") && strings.EqualFold(outputFmt, "table") && !cfg.IsSet("output")
}
