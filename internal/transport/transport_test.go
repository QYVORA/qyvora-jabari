package transport

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// stubADB places an executable `adb` stub first in PATH so construction-time
// LookPath succeeds. The factory tests only exercise transport wiring (which
// address and port a target produces), never adb itself, so they must not
// require a real Android SDK on the machine or CI runner.
func stubADB(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	stub := filepath.Join(dir, "adb")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write adb stub: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestParseProps(t *testing.T) {
	in := []byte(`[ro.product.manufacturer]: [Example]
[ro.build.version.release]: [13]
[ro.adb.secure]: [1]
[ro.build.version.security_patch]: [2025-11-01]
`)
	props := parseProps(in)

	want := map[string]string{
		"ro.product.manufacturer":         "Example",
		"ro.build.version.release":        "13",
		"ro.adb.secure":                   "1",
		"ro.build.version.security_patch": "2025-11-01",
	}
	for k, v := range want {
		if props[k] != v {
			t.Errorf("props[%q] = %q, want %q", k, props[k], v)
		}
	}
}

func TestParsePropsSkipsMalformedLines(t *testing.T) {
	in := []byte("garbage\n[broken\n[good]: [value]\n")
	props := parseProps(in)
	if props["good"] != "value" {
		t.Errorf("props[good] = %q, want value", props["good"])
	}
	if len(props) != 1 {
		t.Errorf("parseProps = %v, want only the well-formed line", props)
	}
}

func TestNewForTargetNetwork(t *testing.T) {
	// The factory must build a network transport with the requested address
	// and default ADB port.
	stubADB(t)
	tr, err := NewForTarget(&models.Target{
		Type:    models.TargetNetwork,
		Address: "203.0.113.7",
	}, 5*time.Second)
	if err != nil {
		t.Fatalf("NewForTarget: %v", err)
	}
	nt, ok := tr.(*NetworkTransport)
	if !ok {
		t.Fatalf("expected *NetworkTransport, got %T", tr)
	}
	if nt.addr != "203.0.113.7" || nt.adbPort != 5555 {
		t.Errorf("network transport = %s:%d", nt.addr, nt.adbPort)
	}
}

func TestNewForTargetUnknownType(t *testing.T) {
	if _, err := NewForTarget(&models.Target{Type: models.TargetType("bogus")}, 0); err == nil {
		t.Error("unknown target types must not build a transport")
	}
}

func TestNewForTargetAPKReturnsStaticTransport(t *testing.T) {
	tr, err := NewForTarget(&models.Target{Type: models.TargetAPK, Address: "/tmp/app.apk"}, 0)
	if err != nil {
		t.Fatalf("NewForTarget(APK): %v", err)
	}
	if _, ok := tr.(*StaticTransport); !ok {
		t.Fatalf("expected *StaticTransport, got %T", tr)
	}
}

func TestStaticTransportIsNonOperational(t *testing.T) {
	tr := &StaticTransport{}
	ctx := context.TODO()
	if err := tr.Connect(ctx); err != nil {
		t.Errorf("Connect returned error: %v", err)
	}
	if err := tr.Disconnect(); err != nil {
		t.Errorf("Disconnect returned error: %v", err)
	}
	if _, err := tr.Info(ctx); err == nil {
		t.Error("StaticTransport.Info must be unsupported")
	}
	if _, err := tr.Execute(ctx, models.Request{}); err == nil {
		t.Error("StaticTransport.Execute must be unsupported")
	}
}
