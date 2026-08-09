package adb

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// fakeRunner records the arguments passed to adb and returns canned output.
// It lets parsing and scoping logic be tested without a real device.
type fakeRunner struct {
	output  string
	stderr  string
	callErr error
	calls   [][]string
}

func (f *fakeRunner) run(_ context.Context, args ...string) ([]byte, []byte, error) {
	f.calls = append(f.calls, args)
	if f.callErr != nil {
		return nil, []byte(f.stderr), f.callErr
	}
	return []byte(f.output), []byte(f.stderr), nil
}

func testClient(f *fakeRunner) *Client {
	return &Client{timeout: 0, runner: f}
}

func TestParseDevices(t *testing.T) {
	in := []byte(`List of devices attached
0123456789ABCDEF       device product:jabari model:Pixel_8 device:panther
abc123               offline
zzz99                unauthorized
`)
	devices, err := parseDevices(in)
	if err != nil {
		t.Fatalf("parseDevices: %v", err)
	}
	want := []Device{
		{Serial: "0123456789ABCDEF", State: StateDevice, Model: "Pixel_8", Device: "panther"},
		{Serial: "abc123", State: StateOffline},
		{Serial: "zzz99", State: StateUnauthorized},
	}
	if !reflect.DeepEqual(devices, want) {
		t.Errorf("parseDevices = %+v, want %+v", devices, want)
	}
}

func TestParseDevicesEmpty(t *testing.T) {
	devices, err := parseDevices([]byte("List of devices attached\n\n"))
	if err != nil {
		t.Fatalf("parseDevices: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("expected no devices, got %+v", devices)
	}
}

func TestDevices(t *testing.T) {
	f := &fakeRunner{output: "List of devices attached\nabc123 device\n"}
	c := testClient(f)

	devices, err := c.Devices(context.Background())
	if err != nil {
		t.Fatalf("Devices: %v", err)
	}
	if len(devices) != 1 || devices[0].Serial != "abc123" {
		t.Fatalf("Devices = %+v, want one device abc123", devices)
	}
	// The scope argument must be "devices -l".
	if !reflect.DeepEqual(f.calls[0], []string{"devices", "-l"}) {
		t.Errorf("calls[0] = %v, want [devices -l]", f.calls[0])
	}
}

func TestDeviceScoping(t *testing.T) {
	f := &fakeRunner{output: "4"}
	c := testClient(f)
	c.device = "SERIAL42"

	if _, err := c.Shell(context.Background(), "getprop ro.build.version.sdk"); err != nil {
		t.Fatalf("Shell: %v", err)
	}
	// Shell passes the command as a single argument; adb joins args.
	want := []string{"-s", "SERIAL42", "shell", "getprop ro.build.version.sdk"}
	if !reflect.DeepEqual(f.calls[0], want) {
		t.Errorf("calls[0] = %v, want %v", f.calls[0], want)
	}
}

func TestListPackages(t *testing.T) {
	f := &fakeRunner{output: "package:com.android.settings\npackage:com.example.app\n"}
	c := testClient(f)

	pkgs, err := c.ListPackages(context.Background(), false)
	if err != nil {
		t.Fatalf("ListPackages: %v", err)
	}
	want := []string{"com.android.settings", "com.example.app"}
	if !reflect.DeepEqual(pkgs, want) {
		t.Errorf("ListPackages = %v, want %v", pkgs, want)
	}
}

func TestGetProp(t *testing.T) {
	f := &fakeRunner{output: "android.test\n"}
	c := testClient(f)

	v, err := c.GetProp(context.Background(), "ro.product.brand")
	if err != nil {
		t.Fatalf("GetProp: %v", err)
	}
	if v != "android.test" {
		t.Errorf("GetProp = %q, want android.test", v)
	}
}

func TestCommandError(t *testing.T) {
	f := &fakeRunner{stderr: "device 'abc' not found", callErr: errors.New("exit status 1")}
	c := testClient(f)
	c.device = "abc"

	_, err := c.Shell(context.Background(), "echo hi")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "adb -s abc shell echo hi: device 'abc' not found" {
		t.Errorf("error = %q", got)
	}
}
