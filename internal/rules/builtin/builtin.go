// Package builtin registers the initial set of JABARI rules (AND-xxx). The
// rules here cover the device and application conditions that the foundation
// can already observe; rules requiring APK extraction or runtime validation
// are added as those modules land.
package builtin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QYVORA/qyvora-jabari/internal/rules"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// Register adds every builtin rule to the registry. It returns the first
// registration error so callers know if the builtin set is inconsistent.
func Register(r *rules.Registry) error {
	for _, rule := range []rules.Rule{
		debuggableProductionRule,
		outdatedPatchRule,
		adbUnauthRule,
		rootedDeviceRule,
		backupEnabledRule,
		cleartextTrafficRule,
		debuggableAppRule,
		outdatedAndroidRule,
		testKeysRule,
		excessivePermsRule,
		debugSignedRule,
	} {
		if err := r.Register(rule); err != nil {
			return err
		}
	}
	return nil
}

// deviceRule is a helper that adapts a pure function into a Rule.
type deviceRule struct {
	id          string
	name        string
	category    string
	description string
	severity    models.Severity
	mitre       []string
	eval        func(ctx context.Context, d *models.DeviceInfo) ([]models.Finding, error)
}

func (r *deviceRule) ID() string                { return r.id }
func (r *deviceRule) Name() string              { return r.name }
func (r *deviceRule) Category() string          { return r.category }
func (r *deviceRule) Description() string       { return r.description }
func (r *deviceRule) Severity() models.Severity { return r.severity }
func (r *deviceRule) MitreRefs() []string       { return r.mitre }

func (r *deviceRule) Evaluate(ctx context.Context, ec rules.EvaluationContext) ([]models.Finding, error) {
	if ec.Device == nil {
		return nil, nil
	}
	return r.eval(ctx, ec.Device)
}

func finding(_, title, desc string, sev models.Severity, conf models.Confidence) models.Finding {
	return models.Finding{
		ID:          models.NewID("fnd"),
		Title:       title,
		Category:    "android-configuration",
		Description: desc,
		Severity:    sev,
		Confidence:  conf,
		Status:      models.StatusDetected,
		Timestamp:   time.Now().UTC(),
	}
}

// AND-001 flags a device built with ro.debuggable=1. On a shipping device
// this indicates a development build in production.
var debuggableProductionRule = &deviceRule{
	id:       "AND-001",
	name:     "Debuggable Production Device",
	category: "android-configuration",
	description: "The device reports ro.debuggable=1, which enables adb root and " +
		"debugging capabilities that should not be present on production builds.",
	severity: models.SeverityHigh,
	mitre:    []string{"T1529"},
	eval: func(_ context.Context, d *models.DeviceInfo) ([]models.Finding, error) {
		if d.DebugState != "1" {
			return nil, nil
		}
		f := finding("AND-001", "Debuggable Production Device",
			"ro.debuggable is set to 1; the device was built as a debuggable image.",
			models.SeverityHigh, models.ConfidenceConfirmed)
		f.Evidence = append(f.Evidence, models.Evidence{
			Kind:    models.KindConfiguration,
			Source:  "ro.debuggable",
			Content: "1",
		})
		f.Impact = "Debugging enables memory inspection and bypass of app sandbox."
		f.Recommendation = "Disable debugging on production builds and enforce verify apps."
		return []models.Finding{f}, nil
	},
}

// AND-002 flags devices whose security patch level is older than the
// configured threshold. The threshold defaults to six months and can be
// overridden via the assessment profile.
var outdatedPatchRule = &deviceRule{
	id:       "AND-002",
	name:     "Outdated Security Patch Level",
	category: "device-hygiene",
	description: "The Android security patch level is older than the acceptable " +
		"threshold, meaning known platform vulnerabilities may be unpatched.",
	severity: models.SeverityMedium,
	mitre:    []string{"T1210"},
	eval: func(_ context.Context, d *models.DeviceInfo) ([]models.Finding, error) {
		patch, err := time.Parse("2006-01-02", d.SecurityPatch)
		if err != nil {
			return nil, nil
		}
		threshold := time.Now().AddDate(0, -6, 0)
		if patch.After(threshold) {
			return nil, nil
		}
		f := finding("AND-002", "Outdated Security Patch Level",
			fmt.Sprintf("Security patch level %s is older than the six-month threshold.",
				d.SecurityPatch), models.SeverityMedium, models.ConfidenceConfirmed)
		f.Evidence = append(f.Evidence, models.Evidence{
			Kind:    models.KindConfiguration,
			Source:  "ro.build.version.security_patch",
			Content: d.SecurityPatch,
		})
		f.Impact = "Known platform CVEs may be unpatched and exploitable."
		f.Recommendation = "Update the device to the latest available security patch level."
		return []models.Finding{f}, nil
	},
}

// AND-003 flags ro.adb.secure=0, which means ADB on the device requires no
// authorization and any connected host can run commands.
var adbUnauthRule = &deviceRule{
	id:       "AND-003",
	name:     "ADB Unauthenticated Access",
	category: "device-exposure",
	description: "ro.adb.secure is set to 0, so ADB does not require host " +
		"authorization. Any host able to reach the ADB port can execute commands.",
	severity: models.SeverityCritical,
	mitre:    []string{"T1471"},
	eval: func(_ context.Context, d *models.DeviceInfo) ([]models.Finding, error) {
		if d.RoAdbSecure == "1" {
			return nil, nil
		}
		if d.RoAdbSecure == "" {
			// Older devices may not expose the property at all; do not fire
			// on missing data.
			return nil, nil
		}
		f := finding("AND-003", "ADB Unauthenticated Access",
			"ADB is enabled without device-side authorization.",
			models.SeverityCritical, models.ConfidenceConfirmed)
		f.Evidence = append(f.Evidence, models.Evidence{
			Kind:    models.KindConfiguration,
			Source:  "ro.adb.secure",
			Content: "0",
		})
		f.Impact = "Unauthenticated ADB allows arbitrary command execution on the device."
		f.Recommendation = "Set ro.adb.secure=1 and disable ADB on production devices."
		return []models.Finding{f}, nil
	},
}

// AND-004 reports a rooted device as a security-relevant condition.
var rootedDeviceRule = &deviceRule{
	id:       "AND-004",
	name:     "Rooted Device",
	category: "device-hygiene",
	description: "A su binary is present, indicating the device has been rooted. " +
		"Root access weakens the platform trust boundary.",
	severity: models.SeverityHigh,
	mitre:    []string{"T1471"},
	eval: func(_ context.Context, d *models.DeviceInfo) ([]models.Finding, error) {
		if !d.Rooted {
			return nil, nil
		}
		f := finding("AND-004", "Rooted Device",
			"Evidence of a su binary was found on the device.",
			models.SeverityHigh, models.ConfidenceMedium)
		f.Evidence = append(f.Evidence, models.Evidence{
			Kind:    models.KindConfiguration,
			Source:  "root_access",
			Content: "su binary present",
		})
		f.Impact = "Root access breaks the Android security sandbox entirely."
		f.Recommendation = "Assess whether root is required; consider attestation-based hardening."
		return []models.Finding{f}, nil
	},
}

// appRule adapts a function over a single app into a Rule that evaluates the
// whole app inventory.
type appRule struct {
	id          string
	name        string
	category    string
	description string
	severity    models.Severity
	mitre       []string
	eval        func(ctx context.Context, app models.Application) ([]models.Finding, error)
}

func (r *appRule) ID() string                { return r.id }
func (r *appRule) Name() string              { return r.name }
func (r *appRule) Category() string          { return r.category }
func (r *appRule) Description() string       { return r.description }
func (r *appRule) Severity() models.Severity { return r.severity }
func (r *appRule) MitreRefs() []string       { return r.mitre }

func (r *appRule) Evaluate(ctx context.Context, ec rules.EvaluationContext) ([]models.Finding, error) {
	var findings []models.Finding
	for _, app := range ec.Apps {
		found, err := r.eval(ctx, app)
		if err != nil {
			return nil, err
		}
		for i := range found {
			found[i].Attributes = map[string]string{"package": app.PackageName}
			findings = append(findings, found[i])
		}
	}
	return findings, nil
}

// AND-005 flags apps that allow backup of their data.
var backupEnabledRule = &appRule{
	id:       "AND-005",
	name:     "Application Backup Enabled",
	category: "application-security",
	description: "The application allows data backup, which can expose private app " +
		"data through adb backup.",
	severity: models.SeverityMedium,
	mitre:    []string{"T1409"},
	eval: func(_ context.Context, app models.Application) ([]models.Finding, error) {
		if !app.AllowBackup {
			return nil, nil
		}
		f := finding("AND-005", "Application Backup Enabled",
			"The application declares allowBackup=true.",
			models.SeverityMedium, models.ConfidenceMedium)
		f.Evidence = append(f.Evidence, models.Evidence{
			Kind:    models.KindConfiguration,
			Source:  "android:allowBackup",
			Content: "true",
		})
		f.Impact = "Backup extraction may reveal app data, credentials, or tokens."
		f.Recommendation = "Set android:allowBackup=false for apps storing sensitive data."
		return []models.Finding{f}, nil
	},
}

// AND-006 flags apps that allow cleartext traffic.
var cleartextTrafficRule = &appRule{
	id:       "AND-006",
	name:     "Cleartext Traffic Allowed",
	category: "application-security",
	description: "The application permits cleartext network traffic, exposing data " +
		"in transit.",
	severity: models.SeverityMedium,
	mitre:    []string{"T1573"},
	eval: func(_ context.Context, app models.Application) ([]models.Finding, error) {
		if !app.UsesCleartext {
			return nil, nil
		}
		f := finding("AND-006", "Cleartext Traffic Allowed",
			"The application uses or permits cleartext traffic.",
			models.SeverityMedium, models.ConfidenceMedium)
		f.Evidence = append(f.Evidence, models.Evidence{
			Kind:    models.KindConfiguration,
			Source:  "usesCleartextTraffic",
			Content: "true",
		})
		f.Impact = "Data in transit can be intercepted on shared networks."
		f.Recommendation = "Enforce HTTPS and set usesCleartextTraffic=false."
		return []models.Finding{f}, nil
	},
}

// AND-007 flags debuggable apps.
var debuggableAppRule = &appRule{
	id:       "AND-007",
	name:     "Debuggable Application",
	category: "application-security",
	description: "The application is built with android:debuggable=true, exposing " +
		"debugging interfaces.",
	severity: models.SeverityHigh,
	mitre:    []string{"T1529"},
	eval: func(_ context.Context, app models.Application) ([]models.Finding, error) {
		if !app.Debuggable {
			return nil, nil
		}
		f := finding("AND-007", "Debuggable Application",
			"The application is built with android:debuggable=true, exposing debugging interfaces.",
			models.SeverityHigh, models.ConfidenceConfirmed)
		f.Evidence = append(f.Evidence, models.Evidence{
			Kind:    models.KindConfiguration,
			Source:  "android:debuggable",
			Content: "true",
		})
		f.Impact = "Debug builds expose debugging interfaces and memory to other apps."
		f.Recommendation = "Remove android:debuggable=true from release builds."
		return []models.Finding{f}, nil
	},
}

// parseAPILevel converts an API level string (e.g. "34") to an int.
func parseAPILevel(s string) (int, bool) {
	level, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, false
	}
	return level, true
}

// parseAndroidVersion converts a version string like "10" or "9.0" to a float.
func parseAndroidVersion(s string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// AND-008 flags devices running an outdated Android version (below Android 11
// / API 30), which are past their security update window.
var outdatedAndroidRule = &deviceRule{
	id:       "AND-008",
	name:     "Outdated Android Version",
	category: "device-hygiene",
	description: "The device runs an Android version that has reached end of " +
		"life, so the platform no longer receives security updates.",
	severity: models.SeverityMedium,
	mitre:    []string{"T1210"},
	eval: func(_ context.Context, d *models.DeviceInfo) ([]models.Finding, error) {
		outdated := false
		detail := ""
		if level, ok := parseAPILevel(d.APILevel); ok {
			if level < 30 {
				outdated = true
				detail = fmt.Sprintf("API level %d (below the Android 11 minimum)", level)
			}
		} else if v, ok := parseAndroidVersion(d.AndroidVersion); ok {
			if v < 11 {
				outdated = true
				detail = fmt.Sprintf("Android %s (below version 11)", d.AndroidVersion)
			}
		}
		if !outdated {
			return nil, nil
		}
		f := finding("AND-008", "Outdated Android Version",
			detail+". Android versions below 11 are end-of-life and receive no security updates.",
			models.SeverityMedium, models.ConfidenceConfirmed)
		f.Evidence = append(f.Evidence, models.Evidence{
			Kind:    models.KindConfiguration,
			Source:  "android_version",
			Content: d.AndroidVersion,
		})
		f.Impact = "End-of-life platforms receive no security patches."
		f.Recommendation = "Update the device to a supported Android version receiving security updates."
		return []models.Finding{f}, nil
	},
}

// AND-009 flags builds tagged test-keys (ro.build.tags=test-keys), indicating
// an image signed with platform test keys rather than production keys.
var testKeysRule = &deviceRule{
	id:       "AND-009",
	name:     "Test-Keys Build",
	category: "android-configuration",
	description: "The build is tagged test-keys, meaning it was signed with " +
		"platform test keys rather than production signing keys.",
	severity: models.SeverityHigh,
	mitre:    []string{"T1471"},
	eval: func(_ context.Context, d *models.DeviceInfo) ([]models.Finding, error) {
		tags := d.SystemProperties["ro.build.tags"]
		if tags == "" && strings.Contains(d.BuildFingerprint, "test-keys") {
			tags = "test-keys"
		}
		if tags != "test-keys" {
			return nil, nil
		}
		f := finding("AND-009", "Test-Keys Build",
			"ro.build.tags is test-keys; the image was built with test platform keys.",
			models.SeverityHigh, models.ConfidenceConfirmed)
		f.Evidence = append(f.Evidence, models.Evidence{
			Kind:    models.KindConfiguration,
			Source:  "ro.build.tags",
			Content: tags,
		})
		f.Impact = "Test-signed builds can be modified without detection."
		f.Recommendation = "Reject test-key signed builds in production and enforce production signing."
		return []models.Finding{f}, nil
	},
}

// AND-010 flags apps requesting many dangerous permissions, which widens the
// impact surface if the app or any of its SDKs are compromised.
var excessivePermsRule = &appRule{
	id:       "AND-010",
	name:     "Excessive Dangerous Permissions",
	category: "application-security",
	description: "The application requests a large number of dangerous " +
		"permissions, expanding the impact of a compromise.",
	severity: models.SeverityMedium,
	mitre:    []string{"T1406"},
	eval: func(_ context.Context, app models.Application) ([]models.Finding, error) {
		var dangerous []string
		for _, perm := range app.Permissions {
			if models.PermissionRisk(perm) == "dangerous" {
				dangerous = append(dangerous, perm)
			}
		}
		if len(dangerous) < 4 {
			return nil, nil
		}
		f := finding("AND-010", "Excessive Dangerous Permissions",
			fmt.Sprintf("Requests %d dangerous permissions: %s", len(dangerous), strings.Join(dangerous, ", ")),
			models.SeverityMedium, models.ConfidenceMedium)
		f.Evidence = append(f.Evidence, models.Evidence{
			Kind:    models.KindConfiguration,
			Source:  "permissions",
			Content: strings.Join(dangerous, ", "),
		})
		f.Impact = "Wide permission surface increases damage from app compromise."
		f.Recommendation = "Request only the permissions strictly required by the app's functionality."
		return []models.Finding{f}, nil
	},
}

// AND-012 flags an APK signed with a debug certificate (e.g. CN=Android
// Debug). Debug-signed packages are not trusted for distribution and are a
// common marker of pre-release or tampered builds.
var debugSignedRule = &appRule{
	id:       "AND-012",
	name:     "Debug-Signed APK",
	category: "application-security",
	description: "The application package is signed with a debug certificate " +
		"(e.g. CN=Android Debug), which is not acceptable for distribution.",
	severity: models.SeverityHigh,
	mitre:    []string{"T1402"},
	eval: func(_ context.Context, app models.Application) ([]models.Finding, error) {
		if app.Attributes["debug_signed"] != "true" {
			return nil, nil
		}
		f := finding("AND-012", "Debug-Signed APK",
			"The package is signed with a debug certificate; debug-signed builds are not trusted for distribution.",
			models.SeverityHigh, models.ConfidenceConfirmed)
		f.Evidence = append(f.Evidence, models.Evidence{
			Kind:    models.KindConfiguration,
			Source:  "debug_signed",
			Content: "true",
		})
		f.Impact = "Debug signatures can be forged and are a sign of an untrusted or tampered build."
		f.Recommendation = "Sign release builds with a production key and reject debug-signed packages."
		return []models.Finding{f}, nil
	},
}
