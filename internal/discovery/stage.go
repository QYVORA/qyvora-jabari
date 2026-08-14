// Package discovery implements the first assessment stage. It answers "what
// authorized Android target are we dealing with" by collecting the device
// metadata every later stage depends on.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/internal/transport"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// Stage discovers device metadata for the current target.
type Stage struct{}

// Name returns the stage name used in logs and reports.
func (s *Stage) Name() string { return "discovery" }

// Run collects device information through the transport and records it on
// the target and in the evidence store. Without a connected transport the
// stage fails loudly rather than producing an empty assessment.
func (s *Stage) Run(ctx context.Context, env *core.Env) error {
	if env.Transport == nil {
		return transport.ErrNotConnected
	}

	info, err := env.Transport.Info(ctx)
	if err != nil {
		return fmt.Errorf("device discovery: %w", err)
	}

	env.Target.Device = info
	if env.Log != nil {
		env.Log.Info("target %s identified: %s", env.Target.DisplayName(), info.Summary())
	}

	if env.Evidence != nil {
		if raw, err := json.MarshalIndent(info, "", "  "); err == nil {
			ev, saveErr := env.Evidence.Save(ctx, models.KindDevice, "device-info", raw)
			if saveErr != nil {
				if env.Log != nil {
					env.Log.Warn("storing device evidence: %v", saveErr)
				}
				return nil
			}
			if env.Log != nil {
				env.Log.Debug("device evidence saved: %s", ev.Hash)
			}
		}
	}
	return nil
}
