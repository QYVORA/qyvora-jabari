package transport

import (
	"fmt"
	"net"
	"time"

	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// NewForTarget builds the transport that matches a target's type. USB targets
// get a USBTransport; network targets get a NetworkTransport. APK targets have
// no live transport and return ErrUnsupported because they are analyzed
// statically rather than connected to.
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
	default:
		return nil, fmt.Errorf("%w: target type %q", ErrUnsupported, t.Type)
	}
}
