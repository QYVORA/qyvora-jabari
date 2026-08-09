package transport

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/anomalyco/qyvora-jabari/pkg/adb"
	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// NetworkTransport reaches a specific authorized Android device by its known
// IP address. It never expands to scan the surrounding subnet: the supplied
// address is the target, nothing else.
type NetworkTransport struct {
	addr      string
	adbPort   int
	adb       *adb.Client
	connected bool
}

// NewNetworkTransport creates a transport for addr. When adbPort is zero it
// defaults to 5555 (the standard ADB over TCP port).
func NewNetworkTransport(addr string, adbPort int, timeout time.Duration) (*NetworkTransport, error) {
	if adbPort == 0 {
		adbPort = 5555
	}
	host := net.JoinHostPort(strings.TrimSpace(addr), strconv.Itoa(adbPort))
	client, err := adb.New(adb.WithDevice(host), adb.WithTimeout(timeout))
	if err != nil {
		return nil, err
	}
	return &NetworkTransport{addr: strings.TrimSpace(addr), adbPort: adbPort, adb: client}, nil
}

// adbEndpoint is the host:port the adb binary should talk to.
func (t *NetworkTransport) adbEndpoint() string {
	return net.JoinHostPort(t.addr, strconv.Itoa(t.adbPort))
}

// Connect tries to reach the target over ADB TCP. Failure to connect is
// surfaced so the caller can distinguish "unreachable target" from
// "authorization problem".
func (t *NetworkTransport) Connect(ctx context.Context) error {
	// Attempt a raw TCP dial first so an unreachable host is reported
	// clearly instead of a confusing adb error.
	conn, err := net.DialTimeout("tcp", t.adbEndpoint(), 5*time.Second)
	if err != nil {
		return fmt.Errorf("target %s unreachable: %w", t.addr, err)
	}
	conn.Close()

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// adb connect must run before the endpoint exists as a known device, so
	// use an unscoped client for the handshake, then the scoped client for
	// all subsequent commands.
	plain, err := adb.New(adb.WithTimeout(15 * time.Second))
	if err != nil {
		return err
	}
	if err := plain.Connect(ctx, t.adbEndpoint()); err != nil {
		return err
	}
	t.connected = true
	return nil
}

// Info returns the device metadata gathered over the connected TCP session.
// It delegates to the same property-collection logic used by the USB
// transport so both transports produce an identical DeviceInfo shape.
func (t *NetworkTransport) Info(ctx context.Context) (*models.DeviceInfo, error) {
	if !t.connected {
		return nil, ErrNotConnected
	}
	usb, err := NewUSBTransport(t.adbEndpoint(), 30*time.Second)
	if err != nil {
		return nil, err
	}
	// The endpoint is already connected; skip the device listing handshake
	// and collect properties directly.
	usb.connected = true
	return usb.Info(ctx)
}

// Execute runs a shell command on the target over the TCP session.
func (t *NetworkTransport) Execute(ctx context.Context, req models.Request) (models.Response, error) {
	if !t.connected {
		return models.Response{}, ErrNotConnected
	}
	if req.Command != "shell" {
		return models.Response{}, fmt.Errorf("%w: command %q", ErrUnsupported, req.Command)
	}
	start := time.Now()
	out, err := t.adb.Shell(ctx, strings.Join(req.Args, " "))
	dur := time.Since(start)
	if err != nil {
		if ctx.Err() != nil {
			return models.Response{}, ctx.Err()
		}
		return models.Response{ExitCode: 1, Stderr: []byte(err.Error()), Duration: dur}, nil
	}
	return models.Response{Stdout: []byte(out), Duration: dur}, nil
}

func (t *NetworkTransport) Disconnect() error {
	t.connected = false
	return nil
}

func (t *NetworkTransport) String() string {
	return "network:" + t.addr
}
