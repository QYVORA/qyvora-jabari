package adb

import (
	"context"
	"fmt"
	"strings"
)

// Client is a native ADB client that communicates directly with devices
// without requiring the adb binary
type Client struct {
	conn Connection
}

// Connection interface abstracts the transport (TCP or USB)
type Connection interface {
	Connect(ctx context.Context) error
	Close() error
	Shell(ctx context.Context, command string) ([]byte, error)
}

// NewTCPClient creates a new ADB client using TCP transport
func NewTCPClient(addr string) *Client {
	return &Client{
		conn: NewTCPConnection(addr),
	}
}

// Connect establishes the connection to the device
func (c *Client) Connect(ctx context.Context) error {
	return c.conn.Connect(ctx)
}

// Close closes the connection to the device
func (c *Client) Close() error {
	return c.conn.Close()
}

// Shell executes a shell command on the device
func (c *Client) Shell(ctx context.Context, command string) (string, error) {
	output, err := c.conn.Shell(ctx, command)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// GetProp reads a system property
func (c *Client) GetProp(ctx context.Context, name string) (string, error) {
	output, err := c.Shell(ctx, "getprop "+name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// GetProps reads all system properties
func (c *Client) GetProps(ctx context.Context) (map[string]string, error) {
	output, err := c.Shell(ctx, "getprop")
	if err != nil {
		return nil, err
	}

	props := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if len(line) < 4 || line[0] != '[' {
			continue
		}
		close := strings.IndexByte(line, ']')
		if close < 0 {
			continue
		}
		key := line[1:close]
		rest := strings.TrimSpace(line[close+1:])
		rest = strings.TrimPrefix(rest, ":")
		rest = strings.TrimSpace(rest)
		value := strings.TrimSuffix(strings.TrimPrefix(rest, "["), "]")
		props[key] = value
	}

	return props, nil
}

// ListPackages returns installed packages
func (c *Client) ListPackages(ctx context.Context, onlyThirdParty bool) ([]string, error) {
	cmd := "pm list packages"
	if onlyThirdParty {
		cmd += " -3"
	}

	output, err := c.Shell(ctx, cmd)
	if err != nil {
		return nil, err
	}

	var packages []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package:") {
			packages = append(packages, strings.TrimPrefix(line, "package:"))
		}
	}

	return packages, nil
}

// DeviceInfo retrieves basic device information
func (c *Client) DeviceInfo(ctx context.Context) (map[string]string, error) {
	props, err := c.GetProps(ctx)
	if err != nil {
		return nil, err
	}

	info := map[string]string{
		"manufacturer":      props["ro.product.manufacturer"],
		"model":             props["ro.product.model"],
		"brand":             props["ro.product.brand"],
		"product":           props["ro.product.name"],
		"android_version":   props["ro.build.version.release"],
		"api_level":         props["ro.build.version.sdk"],
		"security_patch":    props["ro.build.version.security_patch"],
		"build_fingerprint": props["ro.build.fingerprint"],
		"build_id":          props["ro.build.id"],
		"architecture":      props["ro.product.cpu.abi"],
		"debuggable":        props["ro.debuggable"],
		"adb_secure":        props["ro.adb.secure"],
	}

	// Get kernel version
	if kernel, err := c.Shell(ctx, "uname -r"); err == nil {
		info["kernel"] = strings.TrimSpace(kernel)
	}

	return info, nil
}

// IsAvailable checks if the native ADB implementation can be used
func IsAvailable() bool {
	// Native ADB is always available as it's built-in
	return true
}

// Version returns the native implementation version
func Version() string {
	return fmt.Sprintf("native ADB v%d", A_VERSION)
}
