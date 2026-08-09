package transport

import (
	"testing"
	"time"

	"github.com/anomalyco/qyvora-jabari/pkg/models"
)

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

func TestNewForTargetUnsupported(t *testing.T) {
	if _, err := NewForTarget(&models.Target{Type: models.TargetAPK}, 0); err == nil {
		t.Error("APK targets must not build a live transport")
	}
}
