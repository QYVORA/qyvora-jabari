package apk

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAPKStructBasic tests basic APK struct methods
func TestAPKStructBasic(t *testing.T) {
	apk := &APK{
		Path:        "/test/app.apk",
		PackageName: "com.example.app",
		VersionName: "1.0.0",
		VersionCode: 1,
		MinSDK:      21,
		TargetSDK:   30,
	}

	t.Run("Summary", func(t *testing.T) {
		summary := apk.Summary()
		expected := "com.example.app 1.0.0 (SDK 21-30)"
		if summary != expected {
			t.Errorf("Summary() = %q, want %q", summary, expected)
		}
	})

	t.Run("IsMultiDex_False", func(t *testing.T) {
		apk.DEXFiles = []DEXFile{
			{Name: "classes.dex", Size: 1024},
		}
		if apk.IsMultiDex() {
			t.Error("IsMultiDex() = true, want false for single DEX")
		}
	})

	t.Run("IsMultiDex_True", func(t *testing.T) {
		apk.DEXFiles = []DEXFile{
			{Name: "classes.dex", Size: 1024},
			{Name: "classes2.dex", Size: 2048},
		}
		if !apk.IsMultiDex() {
			t.Error("IsMultiDex() = false, want true for multiple DEX")
		}
	})

	t.Run("HasNativeCode_False", func(t *testing.T) {
		apk.NativeLibs = []NativeLib{}
		if apk.HasNativeCode() {
			t.Error("HasNativeCode() = true, want false")
		}
	})

	t.Run("HasNativeCode_True", func(t *testing.T) {
		apk.NativeLibs = []NativeLib{
			{ABI: "arm64-v8a", Name: "libnative.so"},
		}
		if !apk.HasNativeCode() {
			t.Error("HasNativeCode() = false, want true")
		}
	})
}

// TestABIs tests ABI extraction
func TestABIs(t *testing.T) {
	tests := []struct {
		name string
		libs []NativeLib
		want []string
	}{
		{
			name: "no_libraries",
			libs: []NativeLib{},
			want: []string{},
		},
		{
			name: "single_abi",
			libs: []NativeLib{
				{ABI: "arm64-v8a", Name: "lib1.so"},
				{ABI: "arm64-v8a", Name: "lib2.so"},
			},
			want: []string{"arm64-v8a"},
		},
		{
			name: "multiple_abis",
			libs: []NativeLib{
				{ABI: "arm64-v8a", Name: "lib1.so"},
				{ABI: "armeabi-v7a", Name: "lib1.so"},
				{ABI: "x86_64", Name: "lib1.so"},
				{ABI: "arm64-v8a", Name: "lib2.so"},
			},
			want: []string{"arm64-v8a", "armeabi-v7a", "x86_64"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apk := &APK{NativeLibs: tt.libs}
			got := apk.ABIs()

			if len(got) != len(tt.want) {
				t.Errorf("ABIs() returned %d ABIs, want %d", len(got), len(tt.want))
				return
			}

			// Check all expected ABIs are present
			abiMap := make(map[string]bool)
			for _, abi := range got {
				abiMap[abi] = true
			}

			for _, want := range tt.want {
				if !abiMap[want] {
					t.Errorf("ABIs() missing %q", want)
				}
			}
		})
	}
}

// TestExportedComponents tests exported component detection
func TestExportedComponents(t *testing.T) {
	apk := &APK{
		Activities: []Activity{
			{Name: "com.example.MainActivity", Exported: true},
			{Name: "com.example.Settings", Exported: false},
		},
		Services: []Service{
			{Name: "com.example.BackgroundService", Exported: true},
		},
		Receivers: []Receiver{
			{Name: "com.example.BootReceiver", Exported: true},
		},
		Providers: []Provider{
			{Name: "com.example.DataProvider", Exported: false},
		},
	}

	exported := apk.ExportedComponents()

	expected := []string{
		"activity:com.example.MainActivity",
		"service:com.example.BackgroundService",
		"receiver:com.example.BootReceiver",
	}

	if len(exported) != len(expected) {
		t.Errorf("ExportedComponents() returned %d components, want %d", len(exported), len(expected))
	}

	exportedMap := make(map[string]bool)
	for _, e := range exported {
		exportedMap[e] = true
	}

	for _, exp := range expected {
		if !exportedMap[exp] {
			t.Errorf("ExportedComponents() missing %q", exp)
		}
	}
}

// TestDangerousPermissions tests dangerous permission classification
func TestDangerousPermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions []Permission
		want        []string
	}{
		{
			name:        "no_permissions",
			permissions: []Permission{},
			want:        []string{},
		},
		{
			name: "normal_permissions_only",
			permissions: []Permission{
				{Name: "android.permission.INTERNET"},
				{Name: "android.permission.ACCESS_NETWORK_STATE"},
			},
			want: []string{},
		},
		{
			name: "dangerous_permissions",
			permissions: []Permission{
				{Name: "android.permission.CAMERA"},
				{Name: "android.permission.INTERNET"},
				{Name: "android.permission.READ_CONTACTS"},
				{Name: "android.permission.ACCESS_FINE_LOCATION"},
			},
			want: []string{
				"android.permission.CAMERA",
				"android.permission.READ_CONTACTS",
				"android.permission.ACCESS_FINE_LOCATION",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apk := &APK{Permissions: tt.permissions}
			got := apk.DangerousPermissions()

			if len(got) != len(tt.want) {
				t.Errorf("DangerousPermissions() returned %d permissions, want %d", len(got), len(tt.want))
			}

			gotMap := make(map[string]bool)
			for _, p := range got {
				gotMap[p] = true
			}

			for _, want := range tt.want {
				if !gotMap[want] {
					t.Errorf("DangerousPermissions() missing %q", want)
				}
			}
		})
	}
}

// TestParsePermission tests permission parsing
func TestParsePermission(t *testing.T) {
	tests := []struct {
		name      string
		permName  string
		wantGroup string
		wantLevel string
	}{
		{
			name:      "dangerous_permission",
			permName:  "android.permission.CAMERA",
			wantGroup: "android.permission",
			wantLevel: "dangerous",
		},
		{
			name:      "normal_permission",
			permName:  "android.permission.INTERNET",
			wantGroup: "android.permission",
			wantLevel: "normal",
		},
		{
			name:      "custom_permission",
			permName:  "com.example.MY_PERMISSION",
			wantGroup: "com.example",
			wantLevel: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm := ParsePermission(tt.permName)

			if perm.Name != tt.permName {
				t.Errorf("ParsePermission().Name = %q, want %q", perm.Name, tt.permName)
			}

			if perm.Group != tt.wantGroup {
				t.Errorf("ParsePermission().Group = %q, want %q", perm.Group, tt.wantGroup)
			}

			if perm.ProtectionLevel != tt.wantLevel {
				t.Errorf("ParsePermission().ProtectionLevel = %q, want %q", perm.ProtectionLevel, tt.wantLevel)
			}
		})
	}
}

// TestCertificateMethods tests certificate-related methods
func TestCertificateMethods(t *testing.T) {
	t.Run("CertificateFingerprint_Empty", func(t *testing.T) {
		apk := &APK{Certificates: []Certificate{}}
		if fp := apk.CertificateFingerprint(); fp != "" {
			t.Errorf("CertificateFingerprint() = %q, want empty string", fp)
		}
	})

	t.Run("CertificateFingerprint_Present", func(t *testing.T) {
		apk := &APK{
			Certificates: []Certificate{
				{SHA256: "abc123"},
			},
		}
		if fp := apk.CertificateFingerprint(); fp != "abc123" {
			t.Errorf("CertificateFingerprint() = %q, want %q", fp, "abc123")
		}
	})

	t.Run("IsDebugSigned_False", func(t *testing.T) {
		apk := &APK{
			Certificates: []Certificate{
				{Subject: "CN=MyCompany, O=MyOrg"},
			},
		}
		if apk.IsDebugSigned() {
			t.Error("IsDebugSigned() = true, want false")
		}
	})

	t.Run("IsDebugSigned_True", func(t *testing.T) {
		apk := &APK{
			Certificates: []Certificate{
				{Subject: "CN=Android Debug, O=Android"},
			},
		}
		if !apk.IsDebugSigned() {
			t.Error("IsDebugSigned() = false, want true")
		}
	})

	t.Run("SignatureScheme_Empty", func(t *testing.T) {
		apk := &APK{Certificates: []Certificate{}}
		schemes := apk.SignatureScheme()
		if len(schemes) != 0 {
			t.Errorf("SignatureScheme() = %v, want empty", schemes)
		}
	})

	t.Run("SignatureScheme_V1", func(t *testing.T) {
		apk := &APK{
			Certificates: []Certificate{
				{Subject: "CN=Test"},
			},
		}
		schemes := apk.SignatureScheme()
		if len(schemes) != 1 || schemes[0] != "v1" {
			t.Errorf("SignatureScheme() = %v, want [v1]", schemes)
		}
	})
}

// TestOpenAPK_NotFound tests opening non-existent APK
func TestOpenAPK_NotFound(t *testing.T) {
	_, err := Open("/nonexistent/file.apk")
	if err == nil {
		t.Error("Open() on non-existent file should return error")
	}
	if !strings.Contains(err.Error(), "open APK") {
		t.Errorf("Open() error = %v, want error containing 'open APK'", err)
	}
}

// TestOpenAPK_NotZip tests opening non-ZIP file
func TestOpenAPK_NotZip(t *testing.T) {
	// Create temporary non-ZIP file
	tmpDir := t.TempDir()
	notZip := filepath.Join(tmpDir, "not-an-apk.apk")

	if err := os.WriteFile(notZip, []byte("not a zip file"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := Open(notZip)
	if err == nil {
		t.Error("Open() on non-ZIP file should return error")
	}
}

// TestOpenAPK_MissingManifest tests APK without manifest
func TestOpenAPK_MissingManifest(t *testing.T) {
	// Create a ZIP without AndroidManifest.xml
	tmpDir := t.TempDir()
	apkPath := filepath.Join(tmpDir, "no-manifest.apk")

	// Create ZIP file
	f, err := os.Create(apkPath)
	if err != nil {
		t.Fatalf("Failed to create test APK: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	// Add a dummy file (not manifest)
	w, err := zw.Create("classes.dex")
	if err != nil {
		t.Fatalf("Failed to create ZIP entry: %v", err)
	}
	w.Write([]byte("fake dex"))

	if err := zw.Close(); err != nil {
		t.Fatalf("Failed to close ZIP: %v", err)
	}

	_, err = Open(apkPath)
	if err == nil {
		t.Error("Open() on APK without manifest should return error")
	}
	if !strings.Contains(err.Error(), "AndroidManifest.xml not found") {
		t.Errorf("Open() error = %v, want error containing 'AndroidManifest.xml not found'", err)
	}
}

// TestParseDEXFiles tests DEX file parsing
func TestParseDEXFiles(t *testing.T) {
	tmpDir := t.TempDir()
	apkPath := filepath.Join(tmpDir, "test.apk")

	// Create test APK with DEX files
	f, err := os.Create(apkPath)
	if err != nil {
		t.Fatalf("Failed to create test APK: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	// Add classes.dex
	w, _ := zw.Create("classes.dex")
	dexContent := []byte("fake dex content 1")
	w.Write(dexContent)

	// Add classes2.dex
	w, _ = zw.Create("classes2.dex")
	dex2Content := []byte("fake dex content 2")
	w.Write(dex2Content)

	// Add classes3.dex
	w, _ = zw.Create("classes3.dex")
	dex3Content := []byte("fake dex content 3")
	w.Write(dex3Content)

	// Add minimal manifest
	w, _ = zw.Create("AndroidManifest.xml")
	w.Write(createMinimalManifest(t))

	if err := zw.Close(); err != nil {
		t.Fatalf("Failed to close ZIP: %v", err)
	}

	apk, err := Open(apkPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	if len(apk.DEXFiles) != 3 {
		t.Errorf("Expected 3 DEX files, got %d", len(apk.DEXFiles))
	}

	if !apk.IsMultiDex() {
		t.Error("IsMultiDex() should return true for 3 DEX files")
	}

	// Verify DEX file names
	dexNames := make(map[string]bool)
	for _, dex := range apk.DEXFiles {
		dexNames[dex.Name] = true
		if dex.Size == 0 {
			t.Errorf("DEX file %s has size 0", dex.Name)
		}
		if dex.SHA256 == "" {
			t.Errorf("DEX file %s has no SHA256", dex.Name)
		}
	}

	for _, expected := range []string{"classes.dex", "classes2.dex", "classes3.dex"} {
		if !dexNames[expected] {
			t.Errorf("Missing DEX file: %s", expected)
		}
	}
}

// TestParseNativeLibs tests native library parsing
func TestParseNativeLibs(t *testing.T) {
	tmpDir := t.TempDir()
	apkPath := filepath.Join(tmpDir, "test.apk")

	f, err := os.Create(apkPath)
	if err != nil {
		t.Fatalf("Failed to create test APK: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	// Add native libraries
	libs := []string{
		"lib/arm64-v8a/libnative.so",
		"lib/arm64-v8a/libcrypto.so",
		"lib/armeabi-v7a/libnative.so",
		"lib/x86_64/libnative.so",
	}

	for _, lib := range libs {
		w, _ := zw.Create(lib)
		w.Write([]byte("fake native library"))
	}

	// Add minimal manifest
	w, _ := zw.Create("AndroidManifest.xml")
	w.Write(createMinimalManifest(t))

	if err := zw.Close(); err != nil {
		t.Fatalf("Failed to close ZIP: %v", err)
	}

	apk, err := Open(apkPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	if len(apk.NativeLibs) != 4 {
		t.Errorf("Expected 4 native libraries, got %d", len(apk.NativeLibs))
	}

	if !apk.HasNativeCode() {
		t.Error("HasNativeCode() should return true")
	}

	// Check ABIs
	abis := apk.ABIs()
	expectedABIs := map[string]bool{
		"arm64-v8a":   true,
		"armeabi-v7a": true,
		"x86_64":      true,
	}

	if len(abis) != 3 {
		t.Errorf("Expected 3 ABIs, got %d", len(abis))
	}

	for _, abi := range abis {
		if !expectedABIs[abi] {
			t.Errorf("Unexpected ABI: %s", abi)
		}
	}
}

// TestParseResourcesArsc tests resources.arsc detection
func TestParseResourcesArsc(t *testing.T) {
	tmpDir := t.TempDir()
	apkPath := filepath.Join(tmpDir, "test.apk")

	f, err := os.Create(apkPath)
	if err != nil {
		t.Fatalf("Failed to create test APK: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	// Add resources.arsc
	w, _ := zw.Create("resources.arsc")
	w.Write([]byte("fake resources"))

	// Add minimal manifest
	w, _ = zw.Create("AndroidManifest.xml")
	w.Write(createMinimalManifest(t))

	if err := zw.Close(); err != nil {
		t.Fatalf("Failed to close ZIP: %v", err)
	}

	apk, err := Open(apkPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	if !apk.HasResources {
		t.Error("HasResources should be true when resources.arsc present")
	}
}

// TestParseAPKHash tests APK hash computation
func TestParseAPKHash(t *testing.T) {
	tmpDir := t.TempDir()
	apkPath := filepath.Join(tmpDir, "test.apk")

	f, err := os.Create(apkPath)
	if err != nil {
		t.Fatalf("Failed to create test APK: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)

	// Add minimal manifest
	w, _ := zw.Create("AndroidManifest.xml")
	w.Write(createMinimalManifest(t))

	if err := zw.Close(); err != nil {
		t.Fatalf("Failed to close ZIP: %v", err)
	}

	apk, err := Open(apkPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	if apk.APKHash == "" {
		t.Error("APKHash should not be empty")
	}

	// Hash should be hex string
	if len(apk.APKHash) != 64 {
		t.Errorf("APKHash should be 64 characters (SHA-256), got %d", len(apk.APKHash))
	}

	// Verify it's valid hex
	for _, c := range apk.APKHash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("APKHash contains non-hex character: %c", c)
		}
	}
}

// createMinimalManifest creates a minimal binary XML manifest for testing
func createMinimalManifest(t *testing.T) []byte {
	t.Helper()

	// This is a simplified binary XML manifest structure
	// In a real test, we'd need to create a proper AXML structure
	// For now, we'll create a minimal structure that won't crash the parser

	buf := new(bytes.Buffer)

	// AXML header
	binary.Write(buf, binary.LittleEndian, uint16(0x0003)) // Type: XML
	binary.Write(buf, binary.LittleEndian, uint16(0x0008)) // Header size
	binary.Write(buf, binary.LittleEndian, uint32(0))      // File size (placeholder)

	// String pool chunk
	binary.Write(buf, binary.LittleEndian, uint16(0x0001)) // Type: String Pool
	binary.Write(buf, binary.LittleEndian, uint16(0x001C)) // Header size
	binary.Write(buf, binary.LittleEndian, uint32(100))    // Chunk size
	binary.Write(buf, binary.LittleEndian, uint32(1))      // String count
	binary.Write(buf, binary.LittleEndian, uint32(0))      // Style count
	binary.Write(buf, binary.LittleEndian, uint32(0))      // Flags
	binary.Write(buf, binary.LittleEndian, uint32(28))     // Strings start
	binary.Write(buf, binary.LittleEndian, uint32(0))      // Styles start

	// String offset
	binary.Write(buf, binary.LittleEndian, uint32(0))

	// String data: "manifest"
	binary.Write(buf, binary.LittleEndian, uint16(8))           // Length
	buf.WriteString("m\x00a\x00n\x00i\x00f\x00e\x00s\x00t\x00") // UTF-16LE
	binary.Write(buf, binary.LittleEndian, uint16(0))           // Null terminator

	// Pad to make chunk size match
	for buf.Len() < 100 {
		buf.WriteByte(0)
	}

	// Resource ID chunk
	binary.Write(buf, binary.LittleEndian, uint16(0x0180)) // Type: Resource IDs
	binary.Write(buf, binary.LittleEndian, uint16(0x0008)) // Header size
	binary.Write(buf, binary.LittleEndian, uint32(16))     // Chunk size
	binary.Write(buf, binary.LittleEndian, uint32(0x01010000))
	binary.Write(buf, binary.LittleEndian, uint32(0x01010000))

	// Start namespace chunk
	binary.Write(buf, binary.LittleEndian, uint16(0x0100))     // Type: Start Namespace
	binary.Write(buf, binary.LittleEndian, uint16(0x0010))     // Header size
	binary.Write(buf, binary.LittleEndian, uint32(24))         // Chunk size
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Line number
	binary.Write(buf, binary.LittleEndian, uint32(0xFFFFFFFF)) // Comment
	binary.Write(buf, binary.LittleEndian, uint32(0xFFFFFFFF)) // Prefix
	binary.Write(buf, binary.LittleEndian, uint32(0))          // URI

	// Start tag chunk for manifest
	binary.Write(buf, binary.LittleEndian, uint16(0x0102))     // Type: Start Tag
	binary.Write(buf, binary.LittleEndian, uint16(0x0010))     // Header size
	binary.Write(buf, binary.LittleEndian, uint32(36))         // Chunk size
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Line number
	binary.Write(buf, binary.LittleEndian, uint32(0xFFFFFFFF)) // Comment
	binary.Write(buf, binary.LittleEndian, uint32(0xFFFFFFFF)) // Namespace
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Name (string index 0 = "manifest")
	binary.Write(buf, binary.LittleEndian, uint16(0x0014))     // Attribute start
	binary.Write(buf, binary.LittleEndian, uint16(0x0014))     // Attribute size
	binary.Write(buf, binary.LittleEndian, uint16(0))          // Attribute count
	binary.Write(buf, binary.LittleEndian, uint16(0))          // ID index
	binary.Write(buf, binary.LittleEndian, uint16(0))          // Class index
	binary.Write(buf, binary.LittleEndian, uint16(0))          // Style index

	// End tag chunk
	binary.Write(buf, binary.LittleEndian, uint16(0x0103))     // Type: End Tag
	binary.Write(buf, binary.LittleEndian, uint16(0x0010))     // Header size
	binary.Write(buf, binary.LittleEndian, uint32(24))         // Chunk size
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Line number
	binary.Write(buf, binary.LittleEndian, uint32(0xFFFFFFFF)) // Comment
	binary.Write(buf, binary.LittleEndian, uint32(0xFFFFFFFF)) // Namespace
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Name

	// End namespace chunk
	binary.Write(buf, binary.LittleEndian, uint16(0x0101))     // Type: End Namespace
	binary.Write(buf, binary.LittleEndian, uint16(0x0010))     // Header size
	binary.Write(buf, binary.LittleEndian, uint32(24))         // Chunk size
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Line number
	binary.Write(buf, binary.LittleEndian, uint32(0xFFFFFFFF)) // Comment
	binary.Write(buf, binary.LittleEndian, uint32(0xFFFFFFFF)) // Prefix
	binary.Write(buf, binary.LittleEndian, uint32(0))          // URI

	return buf.Bytes()
}

// TestInferExported tests exported component inference
func TestInferExported(t *testing.T) {
	tests := []struct {
		name            string
		componentName   string
		hasIntentFilter bool
		want            bool
	}{
		{
			name:            "with_intent_filter",
			componentName:   "com.example.SomeActivity",
			hasIntentFilter: true,
			want:            true,
		},
		{
			name:            "MainActivity_no_filter",
			componentName:   "com.example.MainActivity",
			hasIntentFilter: false,
			want:            true,
		},
		{
			name:            "regular_activity_no_filter",
			componentName:   "com.example.SomeActivity",
			hasIntentFilter: false,
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferExported(tt.componentName, tt.hasIntentFilter)
			if got != tt.want {
				t.Errorf("inferExported(%q, %v) = %v, want %v",
					tt.componentName, tt.hasIntentFilter, got, tt.want)
			}
		})
	}
}

// TestContainsIgnoreCase tests case-insensitive string matching
func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"Android Debug", "debug", true},
		{"Android Debug", "DEBUG", true},
		{"Android Debug", "android", true},
		{"MyApp", "app", true},
		{"MyApp", "test", false},
		{"", "test", false},
		{"test", "", false},
	}

	for _, tt := range tests {
		got := containsIgnoreCase(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v",
				tt.s, tt.substr, got, tt.want)
		}
	}
}
