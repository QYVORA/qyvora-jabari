package apk

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func u16(b *bytes.Buffer, v uint16) { _ = binary.Write(b, binary.LittleEndian, v) }
func u32(b *bytes.Buffer, v uint32) { _ = binary.Write(b, binary.LittleEndian, v) }

// buildTestManifest produces a minimal but correct AndroidManifest.xml in
// binary XML format that ParseBinaryXML/DecodeManifest can parse. The manifest
// declares a package with one permission, one activity, and the debuggable flag.
func buildTestManifest(t *testing.T) []byte {
	t.Helper()
	t.Helper()

	type str struct {
		idx int
		val string
	}
	// String pool entries
	entries := []str{
		{0, "manifest"},
		{1, "package"},
		{2, "com.example.test"},
		{3, "versionName"},
		{4, "1.0.0"},
		{5, "uses-permission"},
		{6, "name"},
		{7, "android.permission.CAMERA"},
		{8, "activity"},
		{9, "com.example.test.MainActivity"},
		{10, "application"},
		{11, "debuggable"},
		{12, "true"},
	}

	// --- Build string pool ---
	sp := new(bytes.Buffer)
	u16(sp, 0x0001) // type: string pool
	u16(sp, 0x001C) // header size

	// Calculate string data (UTF-16LE with 4-byte alignment)
	sd := new(bytes.Buffer)
	for _, e := range entries {
		u16(sd, uint16(len(e.val)))
		for _, r := range e.val {
			u16(sd, uint16(r))
		}
		u16(sd, 0) // null terminator
		for sd.Len()%4 != 0 {
			sd.WriteByte(0)
		}
	}

	offsetsEnd := uint32(28 + 4*len(entries))
	chunkSize := offsetsEnd + uint32(sd.Len())
	u32(sp, chunkSize)
	u32(sp, uint32(len(entries))) // string count
	u32(sp, 0)                    // style count
	u32(sp, 0)                    // flags = 0 (UTF-16)
	u32(sp, offsetsEnd)
	u32(sp, 0) // styles start

	// Write string offsets
	off := uint32(0)
	for _, e := range entries {
		u32(sp, off)
		off += (2 + uint32(len(e.val))*2 + 2 + 3) &^ 3
	}
	sp.Write(sd.Bytes())

	// --- Build start elements ---
	el := new(bytes.Buffer)

	// Helper to write a start element chunk
	writeStart := func(nameIdx uint32, attrs [][3]uint32) {
		n := uint32(len(attrs))
		cs := 36 + 20*n
		u16(el, 0x0102)     // type: start element
		u16(el, 0x0010)     // header size
		u32(el, cs)         // chunk size
		u32(el, 0)          // line number
		u32(el, 0xFFFFFFFF) // comment
		u32(el, 0xFFFFFFFF) // namespace
		u32(el, nameIdx)    // element name
		u16(el, 0x0014)     // attributeStart (20)
		u16(el, 0x0014)     // attributeSize (20)
		u16(el, uint16(n))  // attributeCount
		u16(el, 0)          // idIndex
		u16(el, 0)          // classIndex
		u16(el, 0)          // styleIndex
		for _, a := range attrs {
			u32(el, 0xFFFFFFFF)      // attr namespace
			u32(el, a[0])            // attr name index
			u32(el, a[1])            // rawValue (string pool index)
			u16(el, 0x0010)          // Res_value.size
			el.WriteByte(0)          // res0
			el.WriteByte(byte(a[2])) // dataType
			u32(el, a[1])            // data (same as rawValue for string types)
		}
	}

	// <manifest package="com.example.test" versionName="1.0.0">
	writeStart(0, [][3]uint32{
		{1, 2, 0x03}, // package = "com.example.test"
		{3, 4, 0x03}, // versionName = "1.0.0"
	})
	// <uses-permission name="android.permission.CAMERA">
	writeStart(5, [][3]uint32{
		{6, 7, 0x03}, // name = "android.permission.CAMERA"
	})
	// <application debuggable="true">
	writeStart(10, [][3]uint32{
		{11, 12, 0x12}, // debuggable = true (TYPE_INT_BOOLEAN, data=0xFFFFFFFF)
	})
	// <activity name="com.example.test.MainActivity">
	writeStart(8, [][3]uint32{
		{6, 9, 0x03}, // name = "com.example.test.MainActivity"
	})

	// --- Combine: XML header + string pool + elements ---
	xml := new(bytes.Buffer)
	u16(xml, 0x0003) // type: XML
	u16(xml, 0x0008) // header size
	totalSize := uint32(8) + uint32(sp.Len()) + uint32(el.Len())
	u32(xml, totalSize)
	xml.Write(sp.Bytes())
	xml.Write(el.Bytes())

	return xml.Bytes()
}

func TestDecodeManifestWithFixedParser(t *testing.T) {
	data := buildTestManifest(t)
	parser, err := ParseBinaryXML(data)
	if err != nil {
		t.Fatalf("ParseBinaryXML: %v", err)
	}
	m, err := parser.DecodeManifest()
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	if m.PackageName != "com.example.test" {
		t.Errorf("PackageName = %q, want com.example.test", m.PackageName)
	}
	if m.VersionName != "1.0.0" {
		t.Errorf("VersionName = %q, want 1.0.0", m.VersionName)
	}
	if !m.Debuggable {
		t.Error("Debuggable = false, want true")
	}
	if len(m.Permissions) != 1 || m.Permissions[0] != "android.permission.CAMERA" {
		t.Errorf("Permissions = %v, want [android.permission.CAMERA]", m.Permissions)
	}
	if len(m.Activities) != 1 || m.Activities[0] != "com.example.test.MainActivity" {
		t.Errorf("Activities = %v, want [com.example.test.MainActivity]", m.Activities)
	}
}
