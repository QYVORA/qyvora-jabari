package transport

import (
	"context"

	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// StaticTransport is a no-op Transport bound to an offline target (an APK
// artifact). Every device-bound operation (Connect, Info, Execute) is
// unsupported because there is no live device behind the target; the APK is
// analyzed statically instead.
//
// It exists so the assessment environment can always build a non-nil Transport
// for any authorized target type, keeping the pipeline contract uniform.
type StaticTransport struct{}

// Connect is a no-op: an offline artifact never needs a connection.
func (s *StaticTransport) Connect(context.Context) error { return nil }

// Disconnect is a no-op.
func (s *StaticTransport) Disconnect() error { return nil }

// Info is unsupported for a static target.
func (s *StaticTransport) Info(context.Context) (*models.DeviceInfo, error) {
	return nil, ErrUnsupported
}

// Execute is unsupported for a static target.
func (s *StaticTransport) Execute(_ context.Context, _ models.Request) (models.Response, error) {
	return models.Response{}, ErrUnsupported
}

// String returns a stable label for logging.
func (s *StaticTransport) String() string { return "static:offline" }
