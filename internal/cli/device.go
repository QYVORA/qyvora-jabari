package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	errs "github.com/anomalyco/qyvora-jabari/internal/errors"
	"github.com/anomalyco/qyvora-jabari/pkg/adb"
	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// defaultADBPort is the standard ADB over TCP port used when a network target
// has no explicit port.
const defaultADBPort = 5555

// cmdDevice implements the "device" console command.
func (c *jabariConsole) cmdDevice(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: device shell [command]")
	}
	switch strings.ToLower(args[0]) {
	case "shell":
		return c.cmdDeviceShell(args[1:])
	default:
		return fmt.Errorf("unknown device subcommand %q (try 'device shell')", args[0])
	}
}

// cmdDeviceShell opens a shell on the current target. Bare "device shell"
// drops into an interactive device shell (the Metasploit shell equivalent);
// "device shell <command>" runs one command on the device with the operator's
// stdin attached so interactive sub-commands still work.
func (c *jabariConsole) cmdDeviceShell(args []string) error {
	t := targets.Current()
	if t == nil {
		return errs.NewExitError(2, "no target selected; use 'target usb' or 'target ip <addr>' first")
	}
	if !t.Authorized() {
		return errs.NewExitError(2, "current target is not authorized: "+t.DisplayName())
	}

	scope, err := c.deviceScope(t)
	if err != nil {
		return err
	}

	client, err := adb.New(adb.WithDevice(scope), adb.WithTimeout(timeout))
	if err != nil {
		return errs.NewExitError(2, err.Error()+"; install the Android platform-tools (adb) to use device shell")
	}

	// A network target must be handed to adb with `adb connect` before the
	// scoped client can talk to it.
	if t.Type == models.TargetNetwork {
		unscoped, err := adb.New()
		if err != nil {
			return errs.NewExitError(2, err.Error()+"; install the Android platform-tools (adb) to use device shell")
		}
		ctx, cancel := context.WithTimeout(c.ctx, 15*time.Second)
		defer cancel()
		if err := unscoped.Connect(ctx, scope); err != nil {
			return errs.NewExitError(2, err.Error())
		}
	}

	if len(args) == 0 {
		c.ui.Status(">", "starting interactive shell on %s; type 'exit' to return", t.DisplayName())
	}

	cmd, err := client.ShellCmd(c.ctx, strings.Join(args, " "))
	if err != nil {
		return err
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = c.out
	cmd.Stderr = c.out
	return cmd.Run()
}

// deviceScope resolves the adb device scope for the current target: the USB
// serial for a USB target, or the host:port ADB endpoint for a network
// target.
func (c *jabariConsole) deviceScope(t *models.Target) (string, error) {
	switch t.Type {
	case models.TargetUSB:
		if t.Serial == "" {
			return "", errs.NewExitError(2, "target has no device serial")
		}
		return t.Serial, nil
	case models.TargetNetwork:
		addr := strings.TrimSpace(t.Address)
		if addr == "" {
			return "", errs.NewExitError(2, "target has no network address")
		}
		host := addr
		port := defaultADBPort
		if h, p, err := net.SplitHostPort(addr); err == nil {
			host = h
			if parsed, err := strconv.Atoi(p); err == nil {
				port = parsed
			}
		}
		return net.JoinHostPort(host, strconv.Itoa(port)), nil
	default:
		return "", errs.NewExitError(2, "device shell requires a USB or network target (APK targets are analyzed statically)")
	}
}
