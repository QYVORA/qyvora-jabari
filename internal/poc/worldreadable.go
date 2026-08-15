package poc

import (
	"context"
	"strings"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// worldReadableData proves that an application's private data directory is
// readable by the adb shell user. On a correctly scoped device, /data/data
// is protected by SELinux and the shell uid gets an empty listing or a
// permission error; a readable listing demonstrates the data is exposed.
// The probe is strictly read-only.
type worldReadableData struct{}

const (
	worldReadableModuleID   = "android.world_readable_data"
	worldReadableModuleName = "World-Readable App Data Proof"
)

func init() {
	Register(&worldReadableData{})
}

func (m *worldReadableData) Meta() ModuleMeta {
	return ModuleMeta{
		ID:          worldReadableModuleID,
		Name:        worldReadableModuleName,
		Category:    "application",
		Risk:        RiskLow,
		Description: "Proves the shell user can list an app's /data/data directory, indicating world-readable private data.",
		Preconditions: "An application finding that names a package whose data directory " +
			"can be probed on the live device.",
	}
}

func (m *worldReadableData) Eligible(_ context.Context, env *core.Env, f *models.Finding) bool {
	if env.Transport == nil {
		return false
	}
	if f == nil || f.Attributes["package"] == "" {
		return false
	}
	// Only application findings name a package to probe.
	return f.Category == "application-security"
}

func (m *worldReadableData) Run(ctx context.Context, env *core.Env, f *models.Finding) (*Result, error) {
	pkg := f.Attributes["package"]
	path := "/data/data/" + pkg
	resp, err := execShell(ctx, env, "ls", "-l", path)
	if err != nil {
		return notProven(worldReadableModuleID, f.ID, string(RiskLow),
			"could not probe "+path+" ("+err.Error()+")"), nil
	}
	out := strings.TrimSpace(string(resp.Stdout))
	if resp.OK() && out != "" {
		lines := strings.Split(out, "\n")
		if len(lines) > 3 {
			lines = lines[:3]
		}
		return proven(worldReadableModuleID, f.ID, string(RiskLow),
			"shell user can read "+path,
			[]string{"ls -l " + path + " -> " + strings.Join(lines, " | ")}), nil
	}
	return notProven(worldReadableModuleID, f.ID, string(RiskLow),
		"shell user cannot list "+path+" (data directory is protected)"), nil
}
