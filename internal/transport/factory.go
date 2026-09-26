package transport

import (
	"fmt"
	"net"
	"time"

	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// NewForTarget builds the transport that matches a target's type. USB targets
// get a USBTransport; network targets get a NetworkTransport. APK targets are
// analyzed statically rather than connected to, so they get a no-op
// StaticTransport (never a live connection).
func NewForTarget(t *models.Target, timeout time.Duration) (Transport, error) {
	if t == nil {
		return nil, fmt.Errorf("cannot build transport for nil target")
	}
	switch t.Type {
	case models.TargetUSB:
		return NewUSBTransport(t.Serial, timeout)
	case models.TargetNetwork:
		port := 0
		if t.Address != "" {
			if _, p, err := net.SplitHostPort(t.Address); err == nil {
				var parsed int
				if _, err := fmt.Sscanf(p, "%d", &parsed); err == nil {
					port = parsed
				}
			}
		}
		return NewNetworkTransport(t.Address, port, timeout)
	case models.TargetAPK:
		// APK targets have no live device; they are analyzed statically, so
		// bind a no-op static transport rather than erroring.
		return &StaticTransport{}, nil
	default:
		return nil, fmt.Errorf("%w: target type %q", ErrUnsupported, t.Type)
	}
}

// NewForTargetWithConfig selects the transport for a target honoring the
// transport.native configuration: network targets may use the native ADB
// protocol implementation (no adb binary dependency). USB and APK targets
// keep their existing transports — the native USB path is not implemented
// and APK targets are static-only.
func NewForTargetWithConfig(t *models.Target, timeout time.Duration, useNative bool) (Transport, error) {
	if useNative && t != nil && t.Type == models.TargetNetwork {
		return NewNativeForTarget(t)
	}
	return NewForTarget(t, timeout)
}

// NewNativeForTarget builds a native transport (no adb binary dependency)
// Currently only supports network targets. USB support requires libusb.
func NewNativeForTarget(t *models.Target) (Transport, error) {
	if t == nil {
		return nil, fmt.Errorf("cannot build transport for nil target")
	}
	switch t.Type {
	case models.TargetNetwork:
		port := 5555
		if t.Address != "" {
			if _, p, err := net.SplitHostPort(t.Address); err == nil {
				var parsed int
				if _, err := fmt.Sscanf(p, "%d", &parsed); err == nil {
					port = parsed
				}
			}
		}
		return NewNativeNetworkTransport(t.Address, port)
	case models.TargetUSB:
		return nil, fmt.Errorf("native USB transport not yet implemented, use legacy transport")
	default:
		return nil, fmt.Errorf("%w: target type %q", ErrUnsupported, t.Type)
	}
}
