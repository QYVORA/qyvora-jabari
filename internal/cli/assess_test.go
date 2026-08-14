package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/QYVORA/qyvora-jabari/internal/logger"
	"github.com/QYVORA/qyvora-jabari/internal/target"
	"github.com/spf13/viper"
)

// newAssessEnv resets the package-level globals the assess command depends on
// so a test can run the command in-process without adb or a device.
func newAssessEnv(t *testing.T) {
	t.Helper()
	cfg = viper.New()
	cfg.SetDefault("profile", "standard")
	cfg.SetDefault("report.dir", "reports")
	targets = target.NewManager()
	log = logger.New()
	dryRun = false
	authorizationFlags.authorized = false
}

func TestAssessDryRunPlansWithoutExecuting(t *testing.T) {
	newAssessEnv(t)
	dryRun = true

	cmd := newAssessIPCmd()
	// Registering the --authorized flag resets the global, so set it after.
	authorizationFlags.authorized = true
	cmd.SetArgs([]string{"192.168.1.50"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("assess ip --dry-run: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"dry run", "network device", "standard", "stages:", "- discovery", "- analysis"} {
		if !strings.Contains(out, want) {
			t.Errorf("dry-run plan missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "reports/session-") {
		t.Errorf("dry-run must not persist a session:\n%s", out)
	}
}

func TestAssessDryRunStillRequiresAuthorization(t *testing.T) {
	newAssessEnv(t)
	dryRun = true
	authorizationFlags.authorized = false
	t.Setenv("QYVORA_AUTHORIZED", "")

	cmd := newAssessIPCmd()
	cmd.SetArgs([]string{"192.168.1.50"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("dry-run without authorization must fail")
	}
}
