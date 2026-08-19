// Package apk provides native APK (Android Package) parsing without
// external tools like aapt or apktool.
package apk

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

// APK represents a parsed Android application package
type APK struct {
	Path string

	// Manifest data
	PackageName string
	VersionName string
	VersionCode int64
	MinSDK      int
	TargetSDK   int
	MaxSDK      int

	// Application flags
	Debuggable           bool
	AllowBackup          bool
	UsesCleartextTraffic bool

	// Components
	Permissions []Permission
	Activities  []Activity
	Services    []Service
	Receivers   []Receiver
	Providers   []Provider

	// DEX files
	DEXFiles []DEXFile

	// Native libraries
	NativeLibs []NativeLib

	// Certificates
	Certificates []Certificate

	// Resources
	HasResources bool

	// File hashes
	ManifestSHA256 string
	APKHash        string
}

// Permission represents an Android permission
type Permission struct {
	Name            string
	ProtectionLevel string
	Group           string
}

// Activity represents an Android activity
type Activity struct {
	Name          string
	Exported      bool
	IntentFilters []IntentFilter
}

// Service represents an Android service
type Service struct {
	Name          string
	Exported      bool
	IntentFilters []IntentFilter
}

// Receiver represents a broadcast receiver
type Receiver struct {
	Name          string
	Exported      bool
	IntentFilters []IntentFilter
}

// Provider represents a content provider
type Provider struct {
	Name      string
	Exported  bool
	Authority string
}

// IntentFilter represents an intent filter
type IntentFilter struct {
	Actions    []string
	Categories []string
	Data       []string
}

// DEXFile represents a DEX file in the APK
type DEXFile struct {
	Name   string
	Size   int64
	SHA256 string
}

// NativeLib represents a native library
type NativeLib struct {
	ABI  string // arm64-v8a, armeabi-v7a, x86, x86_64, etc.
	Name string
	Path string
}

// Certificate represents a signing certificate
type Certificate struct {
	Subject      string
	Issuer       string
	SerialNumber string
	NotBefore    string
	NotAfter     string
	SHA256       string
	Algorithm    string
}

// Open opens and parses an APK file
func Open(path string) (*APK, error) {
	apk := &APK{Path: path}

	// Open ZIP archive
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open APK: %w", err)
	}
	defer func() { _ = r.Close() }()

	// Parse APK contents
	if err := apk.parse(&r.Reader); err != nil {
		return nil, err
	}

	return apk, nil
}

// parse extracts information from the APK ZIP archive
func (a *APK) parse(r *zip.Reader) error {
	var manifestFile *zip.File

	for _, f := range r.File {
		name := f.Name

		switch {
		case name == "AndroidManifest.xml":
			manifestFile = f

		case strings.HasPrefix(name, "classes") && strings.HasSuffix(name, ".dex"):
			// DEX file
			hash, err := hashZipFile(f)
			if err != nil {
				return fmt.Errorf("hash %s: %w", name, err)
			}
			a.DEXFiles = append(a.DEXFiles, DEXFile{
				Name:   name,
				Size:   int64(f.UncompressedSize64),
				SHA256: hash,
			})

		case strings.HasPrefix(name, "lib/"):
			// Native library
			parts := strings.Split(name, "/")
			if len(parts) >= 3 {
				a.NativeLibs = append(a.NativeLibs, NativeLib{
					ABI:  parts[1],
					Name: parts[len(parts)-1],
					Path: name,
				})
			}

		case strings.HasPrefix(name, "META-INF/") && (strings.HasSuffix(name, ".RSA") || strings.HasSuffix(name, ".DSA") || strings.HasSuffix(name, ".EC")):
			// Certificate file
			cert, err := parseCertificate(f)
			if err != nil {
				// Log but don't fail
				continue
			}
			a.Certificates = append(a.Certificates, cert)

		case name == "resources.arsc":
			a.HasResources = true
		}
	}

	// Parse AndroidManifest.xml
	if manifestFile == nil {
		return fmt.Errorf("AndroidManifest.xml not found in APK")
	}

	if err := a.parseManifest(manifestFile); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	// Compute APK hash
	hash, err := hashFile(a.Path)
	if err != nil {
		return fmt.Errorf("hash APK: %w", err)
	}
	a.APKHash = hash

	return nil
}

// hashZipFile computes SHA256 of a file in the ZIP
func hashZipFile(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer func() { _ = rc.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, rc); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// hashFile computes SHA256 of a file
func hashFile(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = r.Close() }()

	h := sha256.New()
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		_, _ = io.Copy(h, rc)
		_ = rc.Close()
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// Summary returns a human-readable summary
func (a *APK) Summary() string {
	return fmt.Sprintf("%s %s (SDK %d-%d)",
		a.PackageName, a.VersionName, a.MinSDK, a.TargetSDK)
}

// ABIs returns the list of supported ABIs
func (a *APK) ABIs() []string {
	seen := make(map[string]bool)
	var abis []string
	for _, lib := range a.NativeLibs {
		if !seen[lib.ABI] {
			seen[lib.ABI] = true
			abis = append(abis, lib.ABI)
		}
	}
	return abis
}

// IsMultiDex returns true if the APK contains multiple DEX files
func (a *APK) IsMultiDex() bool {
	return len(a.DEXFiles) > 1
}

// HasNativeCode returns true if the APK contains native libraries
func (a *APK) HasNativeCode() bool {
	return len(a.NativeLibs) > 0
}

// ExportedComponents returns all exported components
func (a *APK) ExportedComponents() []string {
	var exported []string
	for _, act := range a.Activities {
		if act.Exported {
			exported = append(exported, "activity:"+act.Name)
		}
	}
	for _, svc := range a.Services {
		if svc.Exported {
			exported = append(exported, "service:"+svc.Name)
		}
	}
	for _, rcv := range a.Receivers {
		if rcv.Exported {
			exported = append(exported, "receiver:"+rcv.Name)
		}
	}
	for _, prv := range a.Providers {
		if prv.Exported {
			exported = append(exported, "provider:"+prv.Name)
		}
	}
	return exported
}

// DangerousPermissions returns permissions classified as dangerous
func (a *APK) DangerousPermissions() []string {
	dangerous := []string{
		"android.permission.CAMERA",
		"android.permission.READ_CONTACTS",
		"android.permission.WRITE_CONTACTS",
		"android.permission.ACCESS_FINE_LOCATION",
		"android.permission.ACCESS_COARSE_LOCATION",
		"android.permission.RECORD_AUDIO",
		"android.permission.READ_SMS",
		"android.permission.SEND_SMS",
		"android.permission.READ_EXTERNAL_STORAGE",
		"android.permission.WRITE_EXTERNAL_STORAGE",
		"android.permission.READ_PHONE_STATE",
		"android.permission.CALL_PHONE",
	}

	dangerousMap := make(map[string]bool)
	for _, d := range dangerous {
		dangerousMap[d] = true
	}

	var result []string
	for _, p := range a.Permissions {
		if dangerousMap[p.Name] {
			result = append(result, p.Name)
		}
	}
	return result
}
