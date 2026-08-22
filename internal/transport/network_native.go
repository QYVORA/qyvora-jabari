package transport

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	nativeadb "github.com/QYVORA/qyvora-jabari/pkg/android/adb"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// NativeNetworkTransport uses the native ADB protocol implementation
// instead of shelling out to the adb binary
type NativeNetworkTransport struct {
	addr      string
	adbPort   int
	client    *nativeadb.Client
	connected bool
}

// NewNativeNetworkTransport creates a transport using native ADB protocol
func NewNativeNetworkTransport(addr string, adbPort int) (*NativeNetworkTransport, error) {
	if adbPort == 0 {
		adbPort = 5555
	}
	endpoint := net.JoinHostPort(strings.TrimSpace(addr), strconv.Itoa(adbPort))
	client := nativeadb.NewTCPClient(endpoint)

	return &NativeNetworkTransport{
		addr:    strings.TrimSpace(addr),
		adbPort: adbPort,
		client:  client,
	}, nil
}

// Connect establishes the native ADB connection
func (t *NativeNetworkTransport) Connect(ctx context.Context) error {
	// Test TCP connectivity first, honoring caller cancellation.
	endpoint := net.JoinHostPort(t.addr, strconv.Itoa(t.adbPort))
	probeCtx, probeCancel := context.WithTimeout(ctx, 5*time.Second)
	conn, err := new(net.Dialer).DialContext(probeCtx, "tcp", endpoint)
	probeCancel()
	if err != nil {
		return fmt.Errorf("target %s unreachable: %w", t.addr, err)
	}
	_ = conn.Close()

	// Perform native ADB handshake
	if err := t.client.Connect(ctx); err != nil {
		return fmt.Errorf("ADB handshake failed: %w", err)
	}

	t.connected = true
	return nil
}

// Disconnect closes the native ADB connection
func (t *NativeNetworkTransport) Disconnect() error {
	t.connected = false
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}

// Info retrieves device information using native ADB
func (t *NativeNetworkTransport) Info(ctx context.Context) (*models.DeviceInfo, error) {
	if !t.connected {
		return nil, ErrNotConnected
	}

	info := &models.DeviceInfo{
		SystemProperties: map[string]string{},
	}

	// Get all properties at once
	props, err := t.client.GetProps(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading system properties: %w", err)
	}

	info.SystemProperties = props

	// Map to typed fields
	info.Manufacturer = props["ro.product.manufacturer"]
	info.Model = props["ro.product.model"]
	info.Brand = props["ro.product.brand"]
	info.Product = props["ro.product.name"]
	info.AndroidVersion = props["ro.build.version.release"]
	info.APILevel = props["ro.build.version.sdk"]
	info.SecurityPatch = props["ro.build.version.security_patch"]
	info.BuildFingerprint = props["ro.build.fingerprint"]
	info.BuildID = props["ro.build.id"]
	info.Architecture = props["ro.product.cpu.abi"]
	info.RoAdbSecure = props["ro.adb.secure"]
	info.DebugState = props["ro.debuggable"]

	// Get kernel version
	if kernel, err := t.client.Shell(ctx, "uname -r"); err == nil {
		info.Kernel = strings.TrimSpace(kernel)
	}

	// Check for root
	info.Rooted = t.isRooted(ctx)

	return info, nil
}

// Execute runs a command using native ADB
func (t *NativeNetworkTransport) Execute(ctx context.Context, req models.Request) (models.Response, error) {
	if !t.connected {
		return models.Response{}, ErrNotConnected
	}

	if req.Command != "shell" {
		return models.Response{}, fmt.Errorf("%w: command %q", ErrUnsupported, req.Command)
	}

	script := strings.Join(req.Args, " ")
	start := time.Now()
	out, err := t.client.Shell(ctx, script)
	dur := time.Since(start)

	resp := models.Response{
		Stdout:   []byte(out),
		Duration: dur,
		ExitCode: 0,
	}

	if err != nil {
		if ctx.Err() != nil {
			return models.Response{}, ctx.Err()
		}
		resp.ExitCode = 1
		resp.Stderr = []byte(err.Error())
	}

	return resp, nil
}

// String returns a stable label for logs
func (t *NativeNetworkTransport) String() string {
	return "native-network:" + t.addr
}

// isRooted performs a best-effort root check
func (t *NativeNetworkTransport) isRooted(ctx context.Context) bool {
	for _, probe := range []string{"command -v su", "which su"} {
		if out, err := t.client.Shell(ctx, probe); err == nil && strings.TrimSpace(out) != "" {
			return true
		}
	}
	return false
}
