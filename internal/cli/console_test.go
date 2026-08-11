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
	for _, want := range []string{"assess usb [serial]", "target ip <addr>", "report [session-id]", "quit"} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing %q", want)
		}
	}
}
