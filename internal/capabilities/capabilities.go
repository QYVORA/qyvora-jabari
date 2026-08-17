// Package capabilities provides a capability detection system to distinguish
// native Jabari features from optional external tool integrations.
package capabilities

import (
	"os/exec"

	nativeadb "github.com/QYVORA/qyvora-jabari/pkg/android/adb"
)

// Category represents a capability category
type Category string

const (
	CategoryAndroidEngine Category = "Android Engine"
	CategoryAPKEngine     Category = "APK Engine"
	CategoryAnalysis      Category = "Analysis Engine"
	CategoryIntegrations  Category = "Optional Integrations"
)

// Status represents capability availability
type Status string

const (
	StatusNative      Status = "native"
	StatusIntegration Status = "integration"
	StatusMissing     Status = "missing"
)

// Capability represents a single capability
type Capability struct {
	Name        string
	Category    Category
	Status      Status
	Description string
	Binary      string // For integrations
}

// Detect returns all capabilities and their status
func Detect() []Capability {
	caps := []Capability{
		// Native Android Engine
		{
			Name:        "Device discovery",
			Category:    CategoryAndroidEngine,
			Status:      StatusNative,
			Description: "USB/TCP device discovery",
		},
		{
			Name:        "TCP transport",
			Category:    CategoryAndroidEngine,
			Status:      StatusNative,
			Description: "Native ADB protocol over TCP",
		},
		{
			Name:        "USB transport",
			Category:    CategoryAndroidEngine,
			Status:      detectADB(),
			Description: "ADB protocol over USB (requires adb binary for now)",
			Binary:      "adb",
		},
		{
			Name:        "Shell channel",
			Category:    CategoryAndroidEngine,
			Status:      StatusNative,
			Description: "Execute shell commands",
		},
		{
			Name:        "Device properties",
			Category:    CategoryAndroidEngine,
			Status:      StatusNative,
			Description: "Read system properties",
		},
		{
			Name:        "Package enumeration",
			Category:    CategoryAndroidEngine,
			Status:      StatusNative,
			Description: "List installed packages",
		},

		// Native APK Engine
		{
			Name:        "APK container parsing",
			Category:    CategoryAPKEngine,
			Status:      StatusNative,
			Description: "ZIP-based APK structure",
		},
		{
			Name:        "Manifest parsing",
			Category:    CategoryAPKEngine,
			Status:      StatusNative,
			Description: "Binary XML manifest decoding",
		},
		{
			Name:        "Certificate inspection",
			Category:    CategoryAPKEngine,
			Status:      StatusNative,
			Description: "Extract signing certificates",
		},
		{
			Name:        "DEX discovery",
			Category:    CategoryAPKEngine,
			Status:      StatusNative,
			Description: "Identify DEX files",
		},
		{
			Name:        "Component analysis",
			Category:    CategoryAPKEngine,
			Status:      StatusNative,
			Description: "Activities, services, receivers, providers",
		},
		{
			Name:        "Permission analysis",
			Category:    CategoryAPKEngine,
			Status:      StatusNative,
			Description: "Classify permissions",
		},

		// Native Analysis Engine
		{
			Name:        "Security rules",
			Category:    CategoryAnalysis,
			Status:      StatusNative,
			Description: "Rule engine with AND-001 to AND-007",
		},
		{
			Name:        "Evidence collection",
			Category:    CategoryAnalysis,
			Status:      StatusNative,
			Description: "SHA-256 hashed evidence",
		},
		{
			Name:        "Risk scoring",
			Category:    CategoryAnalysis,
			Status:      StatusNative,
			Description: "Severity × confidence scoring",
		},
		{
			Name:        "JSON reporting",
			Category:    CategoryAnalysis,
			Status:      StatusNative,
			Description: "Machine-readable reports",
		},
		{
			Name:        "HTML reporting",
			Category:    CategoryAnalysis,
			Status:      StatusNative,
			Description: "Human-readable reports",
		},

		// Optional Integrations
		{
			Name:        "Frida",
			Category:    CategoryIntegrations,
			Status:      detectTool("frida"),
			Description: "Runtime instrumentation",
			Binary:      "frida",
		},
		{
			Name:        "JADX",
			Category:    CategoryIntegrations,
			Status:      detectTool("jadx"),
			Description: "DEX to Java decompiler",
			Binary:      "jadx",
		},
		{
			Name:        "Apktool",
			Category:    CategoryIntegrations,
			Status:      detectTool("apktool"),
			Description: "APK disassembly/reassembly",
			Binary:      "apktool",
		},
		{
			Name:        "Drozer",
			Category:    CategoryIntegrations,
			Status:      detectTool("drozer"),
			Description: "Attack surface testing",
			Binary:      "drozer",
		},
	}

	return caps
}

// detectADB checks if adb binary is available
func detectADB() Status {
	if _, err := exec.LookPath("adb"); err == nil {
		return StatusIntegration
	}
	return StatusMissing
}

// detectTool checks if a tool is available
func detectTool(name string) Status {
	if _, err := exec.LookPath(name); err == nil {
		return StatusIntegration
	}
	return StatusMissing
}

// Summary returns a summary of native vs integration capabilities
func Summary() (native, integration, missing int) {
	for _, cap := range Detect() {
		switch cap.Status {
		case StatusNative:
			native++
		case StatusIntegration:
			integration++
		case StatusMissing:
			missing++
		}
	}
	return
}

// IsNativeADBAvailable returns true if native ADB implementation is available
func IsNativeADBAvailable() bool {
	return nativeadb.IsAvailable()
}

// ByCategory groups capabilities by category
func ByCategory() map[Category][]Capability {
	result := make(map[Category][]Capability)
	for _, cap := range Detect() {
		result[cap.Category] = append(result[cap.Category], cap)
	}
	return result
}
