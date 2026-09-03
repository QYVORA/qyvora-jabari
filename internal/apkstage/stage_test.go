package apkstage

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/internal/evidence"
	"github.com/QYVORA/qyvora-jabari/internal/rules"
	"github.com/QYVORA/qyvora-jabari/internal/rules/builtin"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

func u16(b *bytes.Buffer, v uint16) { _ = binary.Write(b, binary.LittleEndian, v) }
func u32(b *bytes.Buffer, v uint32) { _ = binary.Write(b, binary.LittleEndian, v) }

// testManifest emits a binary AndroidManifest.xml declaring a debuggable app
// with a couple of dangerous permissions and one activity. It reuses the same
// byte-exact AXML encoding the apk package tests validate.
func testManifest() []byte {
	entries := []struct {
		idx int
		val string
	}{
		{0, "manifest"},
		{1, "package"},
		{2, "com.example.malware"},
		{3, "versionName"},
		{4, "1.0.0"},
		{5, "uses-permission"},
		{6, "name"},
		{7, "android.permission.CAMERA"},
		{8, "android.permission.READ_CONTACTS"},
		{9, "android.permission.ACCESS_FINE_LOCATION"},
		{10, "android.permission.RECORD_AUDIO"},
		{11, "activity"},
		{12, "com.example.malware.MainActivity"},
		{13, "application"},
		{14, "debuggable"},
		{15, "true"},
	}

	sp := new(bytes.Buffer)
	u16(sp, 0x0001)
	u16(sp, 0x001C)
	sd := new(bytes.Buffer)
	for _, e := range entries {
		u16(sd, uint16(len(e.val)))
		for _, r := range e.val {
			u16(sd, uint16(r))
		}
		u16(sd, 0)
		for sd.Len()%4 != 0 {
			sd.WriteByte(0)
		}
	}
	offsetsEnd := uint32(28 + 4*len(entries))
	chunkSize := offsetsEnd + uint32(sd.Len())
	u32(sp, chunkSize)
	u32(sp, uint32(len(entries)))
	u32(sp, 0)
	u32(sp, 0)
	u32(sp, offsetsEnd)
	u32(sp, 0)
	off := uint32(0)
	for _, e := range entries {
		u32(sp, off)
		off += (2 + uint32(len(e.val))*2 + 2 + 3) &^ 3
	}
	sp.Write(sd.Bytes())

	el := new(bytes.Buffer)
	writeStart := func(nameIdx uint32, attrs [][3]uint32) {
		n := uint32(len(attrs))
		u16(el, 0x0102)
		u16(el, 0x0010)
		u32(el, 36+20*n)
		u32(el, 0)
		u32(el, 0xFFFFFFFF)
		u32(el, 0xFFFFFFFF)
		u32(el, nameIdx)
		u16(el, 0x0014)
		u16(el, 0x0014)
		u16(el, uint16(n))
		u16(el, 0)
		u16(el, 0)
		u16(el, 0)
		for _, a := range attrs {
			u32(el, 0xFFFFFFFF)
			u32(el, a[0])
			u32(el, a[1])
			u16(el, 0x0010)
			el.WriteByte(0)
			el.WriteByte(byte(a[2]))
			u32(el, a[1])
		}
	}

	writeStart(0, [][3]uint32{{1, 2, 0x03}})
	writeStart(5, [][3]uint32{{6, 7, 0x03}})
	writeStart(5, [][3]uint32{{6, 8, 0x03}})
	writeStart(5, [][3]uint32{{6, 9, 0x03}})
	writeStart(5, [][3]uint32{{6, 10, 0x03}})
	writeStart(13, [][3]uint32{{14, 15, 0x12}})
	writeStart(11, [][3]uint32{{6, 12, 0x03}})

	xml := new(bytes.Buffer)
	u16(xml, 0x0003)
	u16(xml, 0x0008)
	u32(xml, uint32(8+sp.Len()+el.Len()))
	xml.Write(sp.Bytes())
	xml.Write(el.Bytes())
	return xml.Bytes()
}

// writeTestAPK creates an APK file on disk containing the test manifest.
func writeTestAPK(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "app.apk")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	zw := zip.NewWriter(f)
	w, err := zw.Create("AndroidManifest.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(testManifest()); err != nil {
		t.Fatal(err)
	}
	dex, err := zw.Create("classes.dex")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = dex.Write([]byte("fake dex"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func newTestEnv(t *testing.T, path string) *core.Env {
	t.Helper()
	ev, err := evidence.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sess := models.NewSession()
	return &core.Env{
		Target: &models.Target{
			Type:    models.TargetAPK,
			Address: path,
		},
		Session:  sess,
		Evidence: ev,
	}
}

func TestStagePopulatesInventoryAndEvidence(t *testing.T) {
	path := writeTestAPK(t, t.TempDir())
	env := newTestEnv(t, path)

	if err := (&Stage{}).Run(context.Background(), env); err != nil {
		t.Fatalf("apk stage: %v", err)
	}
	if len(env.Apps) != 1 {
		t.Fatalf("env.Apps length = %d, want 1", len(env.Apps))
	}
	app := env.Apps[0]
	if app.PackageName != "com.example.malware" {
		t.Errorf("PackageName = %q", app.PackageName)
	}
	if !app.Debuggable {
		t.Error("Debuggable = false, want true from manifest")
	}
	if len(app.Permissions) != 4 {
		t.Errorf("permissions = %v, want 4", app.Permissions)
	}
	if len(app.Activities) != 1 || app.Activities[0] != "com.example.malware.MainActivity" {
		t.Errorf("activities = %v", app.Activities)
	}
	if app.Attributes["manifest_sha256"] == "" {
		t.Error("manifest_sha256 attribute not set")
	}
}

// TestEndToEndPipeline verifies the APK pipeline runs the static stage, the
// analysis stage fires the app rules against the parsed package, and a real
// finding (AND-007 debuggable) is produced.
func TestEndToEndPipeline(t *testing.T) {
	path := writeTestAPK(t, t.TempDir())
	env := newTestEnv(t, path)

	// Run the apk stage, then the analysis stage (mirrors the fixed APK
	// pipeline: apk-analysis -> analysis -> validation -> risk).
	if err := (&Stage{}).Run(context.Background(), env); err != nil {
		t.Fatalf("apk stage: %v", err)
	}

	reg := rules.NewRegistry()
	if err := builtin.Register(reg); err != nil {
		t.Fatal(err)
	}
	ec := rules.EvaluationContext{Target: env.Target, Apps: env.Apps}
	findings := reg.Evaluate(context.Background(), ec)
	sel := func(id string) *models.Finding {
		for i := range findings {
			if findings[i].RuleID == id {
				return &findings[i]
			}
		}
		return nil
	}
	if f := sel("AND-007"); f == nil {
		t.Error("AND-007 (debuggable app) did not fire for the debuggable test APK")
	}
	if f := sel("AND-010"); f == nil {
		t.Error("AND-010 (excessive permissions) did not fire for 4 dangerous permissions")
	}
}
