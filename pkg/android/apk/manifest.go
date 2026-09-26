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
	defer func() { _ = rc.Close() }()

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

	// Convert component names to component structs. A component's exported flag
	// is taken from the compiled manifest's android:exported attribute when the
	// parser captured one; undecorated components fall back to the documented
	// name/intent-filter heuristic rather than silently reporting nothing.
	for _, name := range manifestInfo.Activities {
		a.Activities = append(a.Activities, Activity{
			Name:     name,
			Exported: exportedState("activity:"+name, name, manifestInfo.Exported),
		})
	}

	for _, name := range manifestInfo.Services {
		a.Services = append(a.Services, Service{
			Name:     name,
			Exported: exportedState("service:"+name, name, manifestInfo.Exported),
		})
	}

	for _, name := range manifestInfo.Receivers {
		a.Receivers = append(a.Receivers, Receiver{
			Name:     name,
			Exported: exportedState("receiver:"+name, name, manifestInfo.Exported),
		})
	}

	for _, name := range manifestInfo.Providers {
		a.Providers = append(a.Providers, Provider{
			Name:     name,
			Exported: exportedState("provider:"+name, name, manifestInfo.Exported),
		})
	}

	return nil
}

// exportedState resolves a component's exported flag: manifest-declared
// attribute wins; otherwise inferExported applies the name-pattern heuristic
// (commit: intent-filter cross-referencing is not yet available, so
// hasIntentFilter is always false here).
func exportedState(key, name string, declared map[string]bool) bool {
	if v, ok := declared[key]; ok {
		return v
	}
	return inferExported(name, false)
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
