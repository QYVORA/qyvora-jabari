package apk

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

// parseManifest extracts information from AndroidManifest.xml
func (a *APK) parseManifest(f *zip.File) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	// Compute manifest hash
	h := sha256.New()
	h.Write(data)
	a.ManifestSHA256 = hex.EncodeToString(h.Sum(nil))

	// Parse binary XML
	parser, err := ParseBinaryXML(data)
	if err != nil {
		return fmt.Errorf("parse binary XML: %w", err)
	}

	manifestInfo, err := parser.DecodeManifest()
	if err != nil {
		return fmt.Errorf("decode manifest: %w", err)
	}

	// Map to APK struct
	a.PackageName = manifestInfo.PackageName
	a.VersionName = manifestInfo.VersionName
	a.VersionCode = manifestInfo.VersionCode
	a.MinSDK = manifestInfo.MinSDK
	a.TargetSDK = manifestInfo.TargetSDK
	a.Debuggable = manifestInfo.Debuggable
	a.AllowBackup = manifestInfo.AllowBackup
	a.UsesCleartextTraffic = manifestInfo.UsesCleartextTraffic

	// Convert permission strings to Permission structs
	for _, perm := range manifestInfo.Permissions {
		a.Permissions = append(a.Permissions, Permission{
			Name: perm,
		})
	}

	// Convert component names to component structs
	for _, name := range manifestInfo.Activities {
		a.Activities = append(a.Activities, Activity{
			Name:     name,
			Exported: false, // Full detection requires intent-filter analysis
		})
	}

	for _, name := range manifestInfo.Services {
		a.Services = append(a.Services, Service{
			Name:     name,
			Exported: false,
		})
	}

	for _, name := range manifestInfo.Receivers {
		a.Receivers = append(a.Receivers, Receiver{
			Name:     name,
			Exported: false,
		})
	}

	for _, name := range manifestInfo.Providers {
		a.Providers = append(a.Providers, Provider{
			Name:     name,
			Exported: false,
		})
	}

	return nil
}

// parseManifestDeep performs deeper manifest analysis
// This would be extended to handle exported detection, intent filters, etc.
func (a *APK) parseManifestDeep(data []byte) error {
	// Placeholder for full manifest parsing
	// Would extract:
	// - Exported component detection
	// - Intent filters
	// - Permission protection levels
	// - Network security config
	// - Deep links
	return nil
}

func sha256Hash(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

// inferExported attempts to determine if a component is exported
// based on common patterns
func inferExported(name string, hasIntentFilter bool) bool {
	// Components with intent filters are exported by default (pre-API 31)
	if hasIntentFilter {
		return true
	}

	// Common exported activity patterns
	exportedPatterns := []string{
		"MainActivity",
		"LoginActivity",
		"DeepLink",
		"ShareActivity",
	}

	for _, pattern := range exportedPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}

	return false
}

// ParsePermission extracts permission details
func ParsePermission(name string) Permission {
	parts := strings.Split(name, ".")
	group := ""
	if len(parts) > 2 {
		group = strings.Join(parts[:len(parts)-1], ".")
	}

	return Permission{
		Name:            name,
		Group:           group,
		ProtectionLevel: getProtectionLevel(name),
	}
}

// getProtectionLevel classifies a permission
func getProtectionLevel(name string) string {
	dangerous := map[string]bool{
		"android.permission.READ_CALENDAR":          true,
		"android.permission.WRITE_CALENDAR":         true,
		"android.permission.CAMERA":                 true,
		"android.permission.READ_CONTACTS":          true,
		"android.permission.WRITE_CONTACTS":         true,
		"android.permission.GET_ACCOUNTS":           true,
		"android.permission.ACCESS_FINE_LOCATION":   true,
		"android.permission.ACCESS_COARSE_LOCATION": true,
		"android.permission.RECORD_AUDIO":           true,
		"android.permission.READ_PHONE_STATE":       true,
		"android.permission.CALL_PHONE":             true,
		"android.permission.READ_CALL_LOG":          true,
		"android.permission.WRITE_CALL_LOG":         true,
		"android.permission.ADD_VOICEMAIL":          true,
		"android.permission.USE_SIP":                true,
		"android.permission.PROCESS_OUTGOING_CALLS": true,
		"android.permission.BODY_SENSORS":           true,
		"android.permission.SEND_SMS":               true,
		"android.permission.RECEIVE_SMS":            true,
		"android.permission.READ_SMS":               true,
		"android.permission.RECEIVE_WAP_PUSH":       true,
		"android.permission.RECEIVE_MMS":            true,
		"android.permission.READ_EXTERNAL_STORAGE":  true,
		"android.permission.WRITE_EXTERNAL_STORAGE": true,
	}

	if dangerous[name] {
		return "dangerous"
	}

	if strings.HasPrefix(name, "android.permission.") {
		return "normal"
	}

	return "unknown"
}
