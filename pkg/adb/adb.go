// Package adb is a minimal, dependency-light wrapper around the Android
// Debug Bridge (adb) command-line tool.
//
// It deliberately shells out to the system adb binary rather than
// re-implementing the ADB protocol. That keeps the surface small and lets the
// framework work with whatever adb version is installed. The runner is
// injectable so unit tests can exercise parsing logic without a real device.
//
// All methods accept a context for cancellation and a timeout.
package adb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ErrNotFound is returned when the adb binary cannot be located.
var ErrNotFound = errors.New("adb binary not found in PATH")

// DeviceState mirrors the states printed in the "adb devices" output.
type DeviceState string

const (
	// StateDevice means the device is online and authorized.
	StateDevice DeviceState = "device"
	// StateOffline means the device is online but unresponsive.
	StateOffline DeviceState = "offline"
	// StateUnauthorized means the device is connected but the host is not
	// authorized on the device yet.
	StateUnauthorized DeviceState = "unauthorized"
	// StateUnknown covers any state we have not explicitly modeled.
	StateUnknown DeviceState = "unknown"
)

// Device is one entry from "adb devices -l".
type Device struct {
	Serial string
	State  DeviceState
	Model  string
	Device string
}

// runner abstracts execution of the adb binary. It is a package-private
// interface so tests can substitute a fake without exposing a public API.
type runner interface {
	run(ctx context.Context, args ...string) ([]byte, []byte, error)
}

// execRunner executes the real adb binary.
type execRunner struct {
	binary string
}

func (r *execRunner) run(ctx context.Context, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, r.binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// Client wraps an adb binary and an optional device serial. When a serial is
// set, every command is scoped with "-s <serial>" so the client only ever
// talks to that device.
type Client struct {
	timeout time.Duration
	runner  runner
	device  string
}

// Option configures a Client.
type Option func(*Client)

// WithTimeout sets the default timeout applied to every adb invocation.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// WithDevice pins the client to a specific device serial.
func WithDevice(serial string) Option {
	return func(c *Client) { c.device = serial }
}

// New locates adb in PATH and returns a client bound to it. It returns
// ErrNotFound when adb is not installed.
func New(opts ...Option) (*Client, error) {
	path, err := exec.LookPath("adb")
	if err != nil {
		return nil, ErrNotFound
	}
	return newWithBinary(path, opts...), nil
}

func newWithBinary(binary string, opts ...Option) *Client {
	c := &Client{
		timeout: 30 * time.Second,
		runner:  &execRunner{binary: binary},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// LookPath returns the resolved adb binary path or an empty string if adb is
// not installed.
func LookPath() string {
	path, err := exec.LookPath("adb")
	if err != nil {
		return ""
	}
	return path
}

// run executes the given adb arguments with the client timeout and optional
// device scope, then returns the combined error message for diagnostics.
func (c *Client) run(ctx context.Context, args ...string) (stdout []byte, err error) {
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	full := args
	if c.device != "" {
		scoped := make([]string, 0, len(args)+2)
		scoped = append(scoped, "-s", c.device)
		scoped = append(scoped, args...)
		full = scoped
	}

	stdout, stderr, runErr := c.runner.run(ctx, full...)
	if runErr != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("adb: %v", ctx.Err())
		}
		detail := strings.TrimSpace(string(stderr))
		if detail == "" {
			detail = runErr.Error()
		}
		return nil, fmt.Errorf("adb %s: %s", strings.Join(full, " "), detail)
	}
	return stdout, nil
}

// Version returns the "Android Debug Bridge version" line.
func (c *Client) Version(ctx context.Context) (string, error) {
	out, err := c.run(ctx, "version")
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return "", errors.New("adb version returned no output")
	}
	return lines[0], nil
}

// Devices lists connected devices with their state. It returns the empty
// slice when no devices are connected.
func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	out, err := c.run(ctx, "devices", "-l")
	if err != nil {
		return nil, err
	}
	return parseDevices(out)
}

// parseDevices converts the "adb devices -l" listing into Device structs.
// The listing looks like:
//
//	List of devices attached
//	0123456789ABCDEF       device product:foo model:Bar device:baz
//
// Lines without a serial are ignored. A "no permissions" warning on the
// first line is tolerated.
func parseDevices(out []byte) ([]Device, error) {
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var devices []Device
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of devices") || strings.HasPrefix(line, "* ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		d := Device{
			Serial: fields[0],
			State:  parseState(fields[1]),
		}
		for _, f := range fields[2:] {
			switch {
			case strings.HasPrefix(f, "model:"):
				d.Model = strings.TrimPrefix(f, "model:")
			case strings.HasPrefix(f, "device:"):
				d.Device = strings.TrimPrefix(f, "device:")
			}
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func parseState(s string) DeviceState {
	switch DeviceState(s) {
	case StateDevice, StateOffline, StateUnauthorized:
		return DeviceState(s)
	default:
		return StateUnknown
	}
}

// Shell runs a shell command on the target device and returns its stdout.
func (c *Client) Shell(ctx context.Context, command string) (string, error) {
	out, err := c.run(ctx, "shell", command)
	return string(out), err
}

// ShellCmd returns an *exec.Cmd configured to run a shell session on the
// target device, scoped to the client's device when one is set. The caller
// attaches stdin/stdout/stderr and Runs it, so interactive sessions (an
// interactive shell, or commands that read input) work normally. No timeout
// is imposed: interactive sessions must not be killed mid-conversation.
// It returns an error when the client is not backed by the exec runner.
func (c *Client) ShellCmd(ctx context.Context, command string) (*exec.Cmd, error) {
	r, ok := c.runner.(*execRunner)
	if !ok {
		return nil, errors.New("adb: interactive shell requires the exec runner")
	}
	args := make([]string, 0, 3)
	if c.device != "" {
		args = append(args, "-s", c.device)
	}
	args = append(args, "shell")
	if command != "" {
		args = append(args, command)
	}
	return exec.CommandContext(ctx, r.binary, args...), nil
}

// GetProp returns the value of an Android system property, or an empty
// string if the property is unset.
func (c *Client) GetProp(ctx context.Context, name string) (string, error) {
	out, err := c.run(ctx, "shell", "getprop", name)
	return strings.TrimSpace(string(out)), err
}

// GetProps returns the full getprop output as "key=value" lines.
func (c *Client) GetProps(ctx context.Context) ([]byte, error) {
	return c.run(ctx, "shell", "getprop")
}

// ListPackages returns the package names installed on the device as
// "pm list packages" reports them (optionally filtered by -3 for user apps).
func (c *Client) ListPackages(ctx context.Context, onlyThirdParty bool) ([]string, error) {
	args := []string{"shell", "pm", "list", "packages"}
	if onlyThirdParty {
		args = append(args, "-3")
	}
	out, err := c.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	var pkgs []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package:") {
			pkgs = append(pkgs, strings.TrimPrefix(line, "package:"))
		}
	}
	return pkgs, nil
}

// Connect attaches adb to a TCP endpoint of the form "host:port" (for
// example "192.168.1.50:5555"). It returns nil when the device is connected
// or already connected, and an error otherwise. The endpoint string is never
// shell-interpreted because it is passed to adb as a single argument.
func (c *Client) Connect(ctx context.Context, endpoint string) error {
	if c.device != "" {
		return fmt.Errorf("adb: Connect must be called on an unscoped client")
	}
	out, err := c.run(ctx, "connect", endpoint)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(string(out))
	if strings.Contains(text, "connected to "+endpoint) || strings.Contains(text, "already connected to "+endpoint) {
		return nil
	}
	return fmt.Errorf("adb connect %s: %s", endpoint, text)
}

// Disconnect detaches adb from a TCP endpoint.
func (c *Client) Disconnect(ctx context.Context, endpoint string) error {
	_, err := c.run(ctx, "disconnect", endpoint)
	return err
}
