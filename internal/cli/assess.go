package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	errs "github.com/QYVORA/qyvora-jabari/internal/errors"
	"github.com/QYVORA/qyvora-jabari/internal/orchestration"
	"github.com/QYVORA/qyvora-jabari/internal/poc"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// assessFlags are shared by the assess command.
var assessFlags struct {
	profile     string
	poc         bool
	pocHighRisk bool
}

// newAssessCmd builds the "jabari assess" command, the full-pipeline entry
// point.
func newAssessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assess",
		Short: "Run a full assessment pipeline",
		Long: `Run the complete assessment pipeline (discovery, enumeration,
analysis, validation, risk, reporting) against an authorized target.

The optional poc stage is offensive: it runs proof-of-concept modules
against live findings and only runs when --poc is passed on an
authorized target. High-risk PoCs (module: android.exported_activity)
additionally require --poc-high-risk.

Examples:
  jabari assess usb
  jabari assess usb SERIAL
  jabari assess ip 192.168.1.50 --profile deep
  jabari assess usb --poc --authorized`,
	}
	cmd.PersistentFlags().StringVar(&assessFlags.profile, "profile", "", "assessment profile (quick, standard, deep, application, device, network, compliance, research); default from config")
	cmd.AddCommand(newAssessUSBCmd())
	cmd.AddCommand(newAssessIPCmd())
	return cmd
}

// resolveProfile validates the flag/profile and returns an orchestration
// profile.
func resolveProfile() (orchestration.Profile, error) {
	name := assessFlags.profile
	if name == "" {
		name = cfg.GetString("profile")
	}
	if name == "" {
		name = string(orchestration.ProfileStandard)
	}
	if !orchestration.IsValid(name) {
		return "", errs.NewExitError(2, "unknown profile "+name)
	}
	return orchestration.Profile(name), nil
}

// newAssessUSBCmd implements "jabari assess usb [serial]".
func newAssessUSBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "usb [serial]",
		Short: "Assess a connected authorized device",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serial, err := resolveUSBSerial(cmd.Context(), args)
			if err != nil {
				return err
			}
			t := &models.Target{
				ID:        models.NewID("tgt"),
				Name:      "USB device " + serial,
				Type:      models.TargetUSB,
				Serial:    serial,
				CreatedAt: nowUTC(),
			}
			return assess(cmd, t)
		},
	}
	cmd.Flags().BoolVarP(&authorizationFlags.authorized, "authorized", "y", false,
		"confirm authorization non-interactively")
	cmd.Flags().BoolVar(&assessFlags.poc, "poc", false,
		"run proof-of-concept modules against live findings (requires an authorized target)")
	cmd.Flags().BoolVar(&assessFlags.pocHighRisk, "poc-high-risk", false,
		"allow PoC modules that change device state (e.g. android.exported_activity)")
	return cmd
}

// newAssessIPCmd implements "jabari assess ip <address>".
func newAssessIPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ip <address>",
		Short: "Assess one specific authorized Android device",
		Long: `Assess the single authorized device at the supplied IP address.
Only that address is assessed; no surrounding hosts are contacted.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := strings.TrimSpace(args[0])
			if addr == "" || !strings.Contains(addr, ".") {
				return errs.NewExitError(2, "invalid target address: "+args[0])
			}
			t := &models.Target{
				ID:        models.NewID("tgt"),
				Name:      "network device " + addr,
				Type:      models.TargetNetwork,
				Address:   addr,
				CreatedAt: nowUTC(),
			}
			return assess(cmd, t)
		},
	}
	cmd.Flags().BoolVarP(&authorizationFlags.authorized, "authorized", "y", false,
		"confirm authorization non-interactively")
	cmd.Flags().BoolVar(&assessFlags.poc, "poc", false,
		"run proof-of-concept modules against live findings (requires an authorized target)")
	cmd.Flags().BoolVar(&assessFlags.pocHighRisk, "poc-high-risk", false,
		"allow PoC modules that change device state (e.g. android.exported_activity)")
	return cmd
}

// assess runs the full pipeline against a freshly built target.
func assess(cmd *cobra.Command, t *models.Target) error {
	authorized, err := authorize(cmd, t)
	if err != nil {
		return err
	}
	if err := targets.Set(authorized); err != nil {
		return errs.NewExitError(2, "setting target: "+err.Error())
	}

	profile, err := resolveProfile()
	if err != nil {
		return err
	}

	// --dry-run validates the target, the authorization decision and the
	// profile, then prints the plan without touching the device.
	if dryRun {
		pipe := orchestration.ForProfile(profile)
		if assessFlags.poc {
			pipe.Add(&poc.Stage{AllowHighRisk: assessFlags.pocHighRisk})
		}
		fmt.Fprintf(cmd.OutOrStdout(), "dry run: no commands executed\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  target:   %s (%s)\n", t.DisplayName(), t.Type)
		fmt.Fprintf(cmd.OutOrStdout(), "  profile:  %s\n", profile)
		fmt.Fprintf(cmd.OutOrStdout(), "  authorized: %t\n", authorized != nil)
		fmt.Fprintf(cmd.OutOrStdout(), "  stages:\n")
		for _, st := range pipe.Stages() {
			fmt.Fprintf(cmd.OutOrStdout(), "    - %s\n", st.Name())
		}
		return nil
	}

	log.Info("starting assessment of %s (profile %s)", t.DisplayName(), profile)

	session, err := runPipeline(cmd.Context(), profile)
	if err != nil {
		return err
	}
	return renderSession(cmd.Context(), session)
}
