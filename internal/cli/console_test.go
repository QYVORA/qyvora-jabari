package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/anomalyco/qyvora-jabari/internal/target"
	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

// setupConsole builds a console with fresh globals and a buffer writer, so
// tests exercise the command dispatch without a terminal.
func setupConsole(t *testing.T) *jabariConsole {
	t.Helper()
	cfg = viper.New()
	targets = target.NewManager()
	authorizationFlags.authorized = false
	var buf bytes.Buffer
	return &jabariConsole{ctx: context.Background(), out: &buf, ui: newConsoleUI(&buf)}
}

func TestConsoleExecDispatch(t *testing.T) {
	c := setupConsole(t)
	cases := []struct {
		line    string
		wantErr string
	}{
		{"help", ""},
		{"?", ""},
		{"banner", ""},
		{"version", ""},
		{"config", ""},
		{"history", ""},
		{"target list", ""},
		{"get profile", ""},
		{"set profile deep", ""},
		{"unknowncmd", "unknown command"},
		{"target bogus", "unknown target subcommand"},
	}
	for _, tc := range cases {
		_, err := c.exec(tc.line)
		if tc.wantErr == "" {
			if err != nil {
				t.Errorf("%q: unexpected error %v", tc.line, err)
			}
		} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("%q: want error containing %q, got %v", tc.line, tc.wantErr, err)
		}
	}
}

func TestConsoleQuit(t *testing.T) {
	c := setupConsole(t)
	for _, line := range []string{"quit", "exit", "bye"} {
		quit, err := c.exec(line)
		if err != nil {
			t.Errorf("%q: %v", line, err)
		}
		if !quit {
			t.Errorf("%q: expected quit", line)
		}
	}
}

func TestConsoleSetGet(t *testing.T) {
	c := setupConsole(t)
	if err := c.cmdSet([]string{"profile", "deep"}); err != nil {
		t.Fatalf("set profile: %v", err)
	}
	if got := cfg.GetString("profile"); got != "deep" {
		t.Errorf("profile = %q, want deep", got)
	}
	if err := c.cmdSet([]string{"timeout", "60s"}); err != nil {
		t.Fatalf("set timeout: %v", err)
	}
	if timeout != 60*time.Second {
		t.Errorf("timeout = %v, want 60s", timeout)
	}
	if err := c.cmdSet([]string{"report.dir", "/tmp/sessions"}); err != nil {
		t.Fatalf("set report.dir: %v", err)
	}
	if err := c.cmdGet([]string{"profile"}); err != nil {
		t.Fatalf("get profile: %v", err)
	}

	for _, tc := range [][2]string{
		{"profile", "bogus"},
		{"timeout", "nope"},
		{"nope", "x"},
	} {
		if err := c.cmdSet([]string{tc[0], tc[1]}); err == nil {
			t.Errorf("set %s %s: expected error", tc[0], tc[1])
		}
	}
}

func TestConsoleAssessTargetParsing(t *testing.T) {
	c := setupConsole(t)

	profile := ""
	tgt, err := c.assessTarget([]string{"ip", "1.2.3.4"}, &profile)
	if err != nil {
		t.Fatalf("assess ip: %v", err)
	}
	if tgt == nil || tgt.Type != models.TargetNetwork {
		t.Errorf("expected network target, got %v", tgt)
	}

	if _, err := c.assessTarget([]string{"ip", "notanip"}, &profile); err == nil {
		t.Error("assess ip notanip: expected error")
	}
	if _, err := c.assessTarget([]string{"bogus"}, &profile); err == nil {
		t.Error("assess bogus: expected error")
	}

	// A bare --profile flag with no target must set the profile and then
	// fail on the missing current target.
	profile = ""
	if _, err := c.assessTarget([]string{"--profile", "deep"}, &profile); err == nil {
		t.Error("assess --profile deep: expected missing-target error")
	}
	if profile != "deep" {
		t.Errorf("profile = %q, want deep", profile)
	}
}

func TestConsoleHelpOutput(t *testing.T) {
	c := setupConsole(t)
	if _, err := c.exec("help"); err != nil {
		t.Fatalf("help: %v", err)
	}
	out := c.out.(*bytes.Buffer).String()
	for _, want := range []string{"assess usb [serial]", "target ip <addr>", "report [session-id]", "quit", "!<command>"} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing %q", want)
		}
	}
}

func TestConsoleCdPwd(t *testing.T) {
	c := setupConsole(t)

	if _, err := c.exec("pwd"); err != nil {
		t.Fatalf("pwd: %v", err)
	}

	dir := t.TempDir()
	if _, err := c.exec("cd " + dir); err != nil {
		t.Fatalf("cd %s: %v", dir, err)
	}
	if c.cwd != dir {
		t.Errorf("cwd = %q, want %q", c.cwd, dir)
	}

	if _, err := c.exec("cd ."); err != nil {
		t.Fatalf("cd .: %v", err)
	}
	if c.cwd != dir {
		t.Errorf("cwd after cd . = %q, want %q", c.cwd, dir)
	}

	if _, err := c.exec("cd /nonexistent-dir-xyz"); err == nil {
		t.Error("cd /nonexistent-dir-xyz: expected error")
	}
	if c.cwd != dir {
		t.Errorf("cwd after failed cd = %q, want unchanged %q", c.cwd, dir)
	}
}

func TestConsoleShellBangUnknown(t *testing.T) {
	c := setupConsole(t)
	_, err := c.exec("!definitely_not_a_command_xyz")
	if err == nil || !strings.Contains(err.Error(), "command not found") {
		t.Errorf("expected command not found, got %v", err)
	}
}

func TestConsoleShellKind(t *testing.T) {
	c := setupConsole(t)
	cases := []struct {
		line string
		want string
	}{
		{"!ls -l", "shell"},
		{"!pwd", "shell"},
		{"shell", "interactive"},
		{"shell uname -a", "shell"},
		{"cd /tmp", "shell"},
		{"cd", "shell"},
		{"pwd", "shell"},
		{"device shell", "interactive"},
		{"device shell pm list packages", "shell"},
		{"help", ""},
		{"target usb", ""},
		{"assess", ""},
	}
	for _, tc := range cases {
		if got := c.shellKind(tc.line); got != tc.want {
			t.Errorf("shellKind(%q) = %q, want %q", tc.line, got, tc.want)
		}
	}
}

func TestConsoleDeviceDispatch(t *testing.T) {
	c := setupConsole(t)

	_, err := c.exec("device shell ls")
	if err == nil || !strings.Contains(err.Error(), "no target selected") {
		t.Errorf("device shell with no target: want missing-target error, got %v", err)
	}

	if _, err := c.exec("device bogus"); err == nil || !strings.Contains(err.Error(), "unknown device subcommand") {
		t.Errorf("device bogus: want unknown subcommand error, got %v", err)
	}
}

func TestConsoleDeviceScope(t *testing.T) {
	c := setupConsole(t)

	usb, err := c.deviceScope(&models.Target{
		ID:     models.NewID("tgt"),
		Name:   "USB device ABC",
		Type:   models.TargetUSB,
		Serial: "ABC123",
	})
	if err != nil {
		t.Fatalf("deviceScope usb: %v", err)
	}
	if usb != "ABC123" {
		t.Errorf("usb scope = %q, want ABC123", usb)
	}

	netScope, err := c.deviceScope(&models.Target{
		ID:      models.NewID("tgt"),
		Name:    "network device",
		Type:    models.TargetNetwork,
		Address: "10.0.0.5",
	})
	if err != nil {
		t.Fatalf("deviceScope network: %v", err)
	}
	if netScope != "10.0.0.5:5555" {
		t.Errorf("network scope = %q, want 10.0.0.5:5555", netScope)
	}

	explicit, err := c.deviceScope(&models.Target{
		ID:      models.NewID("tgt"),
		Name:    "network device",
		Type:    models.TargetNetwork,
		Address: "10.0.0.5:6000",
	})
	if err != nil {
		t.Fatalf("deviceScope explicit port: %v", err)
	}
	if explicit != "10.0.0.5:6000" {
		t.Errorf("explicit scope = %q, want 10.0.0.5:6000", explicit)
	}

	if _, err := c.deviceScope(&models.Target{
		ID:   models.NewID("tgt"),
		Name: "apk",
		Type: models.TargetAPK,
	}); err == nil {
		t.Error("deviceScope apk: expected error")
	}
}
