// Package transport abstracts how the assessment engine reaches an Android
// target. The engine works against the Transport interface and never needs
// to know whether a device is connected over USB or reached by IP.
package transport

import (
	"context"
	"errors"

	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// Transport is the boundary between the assessment pipeline and the physical
// device. Implementations must honor context cancellation and timeouts on
// every call.
type Transport interface {
	// Connect establishes the connection to the target. It is idempotent:
	// calling Connect on an already-connected transport is a no-op.
	Connect(ctx context.Context) error
	// Disconnect tears down the connection. It must be safe to call more
	// than once.
	Disconnect() error
	// Info collects the normalized device metadata for the target.
	Info(ctx context.Context) (*models.DeviceInfo, error)
	// Execute runs a logical request against the target and returns a
	// normalized response.
	Execute(ctx context.Context, req models.Request) (models.Response, error)
	// String returns a stable human-readable label for logging.
	String() string
}

// Sentinel errors shared by all transport implementations. Callers should
// use errors.Is to test for these rather than comparing strings.
var (
	// ErrNotConnected is returned when an operation requires a live
	// connection that is not present.
	ErrNotConnected = errors.New("transport not connected")
	// ErrDeviceDisconnected is returned when the underlying device is lost
	// mid-assessment.
	ErrDeviceDisconnected = errors.New("device disconnected")
	// ErrUnauthorized is returned when the transport cannot authenticate to
	// the device (for example ADB authorization is pending).
	ErrUnauthorized = errors.New("transport unauthorized")
	// ErrUnsupported is returned when the transport cannot service a
	// request for this target type.
	ErrUnsupported = errors.New("operation not supported by this transport")
)
