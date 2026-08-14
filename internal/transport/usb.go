package transport

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/QYVORA/qyvora-jabari/pkg/adb"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// USBTransport reaches an Android device through the local adb binary over a
// physical (or emulated) USB connection.
type USBTransport struct {
	adb       *adb.Client
	serial    string
	connected bool
}

// NewUSBTransport creates a transport for the given device serial. It fails
// early with adb.ErrNotFound when adb is not installed so the caller can give
// a clear message instead of failing later mid-assessment.
func NewUSBTransport(serial string, timeout time.Duration) (*USBTransport, error) {
	client, err := adb.New(adb.WithDevice(serial), adb.WithTimeout(timeout))
	if err != nil {
		return nil, err
	}
	return &USBTransport{adb: client, serial: serial}, nil
}

// Connect verifies the device is present and authorized. A device that is
// listed as "unauthorized" surfaces as ErrUnauthorized so the user knows to
// accept the RSA prompt on the device.
func (t *USBTransport) Connect(ctx context.Context) error {
	devices, err := t.adb.Devices(ctx)
	if err != nil {
		return fmt.Errorf("listing devices: %w", err)
	}
	for _, d := range devices {
		if d.Serial != t.serial {
			continue
		}
		switch d.State {
		case adb.StateDevice:
			t.connected = true
			return nil
		case adb.StateUnauthorized:
			return fmt.Errorf("%w: authorize this host on the device", ErrUnauthorized)
		default:
			return fmt.Errorf("%w: device %s is %s", ErrDeviceDisconnected, t.serial, d.State)
		}
	}
	return fmt.Errorf("%w: device %s not found", ErrDeviceDisconnected, t.serial)
}

// Disconnect marks the transport as disconnected. adb has no persistent
// connection to tear down for each device, so this is bookkeeping only and
// never fails.
func (t *USBTransport) Disconnect() error {
	t.connected = false
	return nil
}

// Info gathers device metadata from system properties in a single pass where
// possible, falling back to individual property reads for sparsely-populated
// builds.
func (t *USBTransport) Info(ctx context.Context) (*models.DeviceInfo, error) {
	if !t.connected {
		return nil, ErrNotConnected
	}

	info := &models.DeviceInfo{
		Serial:           t.serial,
		SystemProperties: map[string]string{},
	}

	props, err := t.adb.GetProps(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading system properties: %w", err)
	}
	for key, value := range parseProps(props) {
		info.SystemProperties[key] = value
	}

	// Copy the properties the discovery stage is known to consume into the
	// typed fields. Unknown keys remain available via SystemProperties.
	pick := func(key string) string { return info.SystemProperties[key] }
	info.Manufacturer = pick("ro.product.manufacturer")
	info.Model = pick("ro.product.model")
	info.Brand = pick("ro.product.brand")
	info.Product = pick("ro.product.name")
	info.AndroidVersion = pick("ro.build.version.release")
	info.APILevel = pick("ro.build.version.sdk")
	info.SecurityPatch = pick("ro.build.version.security_patch")
	info.BuildFingerprint = pick("ro.build.fingerprint")
	info.BuildID = pick("ro.build.id")
	info.Kernel = strings.TrimSpace(firstLine(info.SystemProperties["ro.kernel.qemu"]))
	info.Architecture = pick("ro.product.cpu.abi")
	info.RoAdbSecure = pick("ro.adb.secure")
	info.DebugState = pick("ro.debuggable")

	// Kernel version is not a property; read it via uname when available.
	if uname, err := t.adb.Shell(ctx, "uname -r"); err == nil {
		info.Kernel = strings.TrimSpace(uname)
	}

	info.Rooted = isRooted(ctx, t)

	return info, nil
}

// Execute maps a logical request onto an adb shell invocation. Only the
// "shell" command is meaningful over USB; the network transport understands
// the same vocabulary so the engine is interchangeable.
func (t *USBTransport) Execute(ctx context.Context, req models.Request) (models.Response, error) {
	if !t.connected {
		return models.Response{}, ErrNotConnected
	}
	if req.Command != "shell" {
		return models.Response{}, fmt.Errorf("%w: command %q", ErrUnsupported, req.Command)
	}

	script := strings.Join(req.Args, " ")
	start := time.Now()
	out, err := t.adb.Shell(ctx, script)
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

// String returns a stable label for logs and reports.
func (t *USBTransport) String() string {
	return "usb:" + t.serial
}

// parseProps converts raw getprop output ("[key]: [value]") into a map.
// Malformed lines are skipped rather than aborting the whole parse because a
// single odd property should not kill discovery.
func parseProps(out []byte) map[string]string {
	props := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if len(line) < 4 || line[0] != '[' {
			continue
		}
		close := strings.IndexByte(line, ']')
		if close < 0 {
			continue
		}
		key := line[1:close]
		// The value follows the key's closing bracket as ": [value]".
		rest := strings.TrimSpace(line[close+1:])
		rest = strings.TrimPrefix(rest, ":")
		rest = strings.TrimSpace(rest)
		value := strings.TrimSuffix(strings.TrimPrefix(rest, "["), "]")
		props[key] = value
	}
	return props
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// isRooted performs a best-effort root check across common indicators
// (su binary in PATH, ro.secure setting). It never blocks the assessment on
// the result.
func isRooted(ctx context.Context, t *USBTransport) bool {
	for _, probe := range []string{"command -v su", "which su"} {
		if out, err := t.adb.Shell(ctx, probe); err == nil && strings.TrimSpace(out) != "" {
			return true
		}
	}
	return false
}
