package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/QYVORA/qyvora-jabari/internal/errors"
	"github.com/QYVORA/qyvora-jabari/pkg/adb"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// newTargetCmd builds the "jabari target" command group.
func newTargetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "target",
		Short: "Manage assessment targets",
		Long: `Select and manage the Android targets under assessment.

A target is either a device connected over USB, a specific authorized device
reached by IP, or an offline APK. Targets must be authorized before they can
be assessed.`,
	}
	cmd.PersistentFlags().BoolVarP(&authorizationFlags.authorized, "authorized", "y", false,
		"confirm authorization for the target non-interactively")
	cmd.AddCommand(newTargetUSBCmd())
	cmd.AddCommand(newTargetIPCmd())
	cmd.AddCommand(newTargetShowCmd())
	cmd.AddCommand(newTargetListCmd())
	return cmd
}

// newTargetUSBCmd implements "jabari target usb [serial]".
func newTargetUSBCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "usb [serial]",
		Short: "Select a connected Android device",
		Long: `Select a device connected over ADB as the current target. When exactly
one device is connected the serial may be omitted.`,
		Args: cobra.MaximumNArgs(1),
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
			authorized, err := authorize(cmd, t)
			if err != nil {
				return err
			}
			if err := targets.Set(authorized); err != nil {
				return errors.NewExitError(2, "setting target: "+err.Error())
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Target set: %s (usb:%s)\n", t.Name, serial)
			return nil
		},
	}
}

// resolveUSBSerial picks the device serial for a USB target.
func resolveUSBSerial(ctx context.Context, args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	client, err := adb.New()
	if err != nil {
		return "", errors.NewExitError(2, err.Error()+"; install the Android platform-tools (adb) to use USB targets")
	}
	devices, err := client.Devices(ctx)
	if err != nil {
		return "", errors.NewExitError(2, "listing devices: "+err.Error())
	}
	var ready []adb.Device
	for _, d := range devices {
		if d.State == adb.StateDevice {
			ready = append(ready, d)
		}
	}
	switch len(ready) {
	case 0:
		return "", errors.NewExitError(2, "no authorized Android device found over ADB; connect one and accept the authorization prompt")
	case 1:
		return ready[0].Serial, nil
	default:
		var names []string
		for _, d := range ready {
			names = append(names, d.Serial)
		}
		return "", errors.NewExitError(2,
			"multiple devices connected ("+strings.Join(names, ", ")+"); specify the serial: 'jabari target usb <serial>'")
	}
}

// newTargetIPCmd implements "jabari target ip <address>".
func newTargetIPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ip <address>",
		Short: "Select a specific authorized Android device by IP",
		Long: `Select a specific Android device by its known IP address as the current
target. Only the supplied address is assessed; jabari never expands to the
surrounding subnet.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := strings.TrimSpace(args[0])
			if addr == "" || !strings.Contains(addr, ".") {
				return errors.NewExitError(2, "invalid target address: "+args[0])
			}
			t := &models.Target{
				ID:        models.NewID("tgt"),
				Name:      "network device " + addr,
				Type:      models.TargetNetwork,
				Address:   addr,
				CreatedAt: nowUTC(),
			}
			authorized, err := authorize(cmd, t)
			if err != nil {
				return err
			}
			if err := targets.Set(authorized); err != nil {
				return errors.NewExitError(2, "setting target: "+err.Error())
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Target set: %s (network:%s)\n", t.Name, addr)
			return nil
		},
	}
}

// newTargetShowCmd implements "jabari target show".
func newTargetShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the current target",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			t := targets.Current()
			if t == nil {
				return errors.NewExitError(2, "no target selected")
			}
			status := "not authorized"
			if t.Authorized() {
				status = "authorized"
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Target: %s\n", t.DisplayName())
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  ID:     %s\n", t.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Type:   %s\n", t.Type)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Status: %s\n", status)
			if t.Device != nil {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Device: %s\n", t.Device.Summary())
			}
			return nil
		},
	}
}

// newTargetListCmd implements "jabari target list".
func newTargetListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List known targets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			list := targets.List()
			if len(list) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No targets.")
				return nil
			}
			for _, t := range list {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s  %-10s  %-30s  %s\n",
					t.ID, t.Type, t.DisplayName(), authStatus(t))
			}
			return nil
		},
	}
}

func authStatus(t *models.Target) string {
	if t.Authorized() {
		return "authorized"
	}
	return "not authorized"
}
