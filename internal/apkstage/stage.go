// Package apkstage implements the assessment stage that statically analyzes an
// offline APK artifact: it parses the package, converts it into the normalized
// application inventory (env.Apps), and records manifest/APK evidence. It is
// the device-agnostic sibling of the discovery + enumeration stages and is
// used for TargetAPK targets, which have no live transport behind them.
package apkstage

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/pkg/android/apk"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// Stage parses the APK referenced by the current target and populates the
// application inventory that the analysis stage evaluates rules against.
type Stage struct{}

// Name returns the stage name used in logs and reports.
func (s *Stage) Name() string { return "apk-analysis" }

// Run opens the APK at the target path, converts it into a models.Application,
// stores it as the sole inventory entry, and records manifest and APK evidence.
// Static app rules (backup, cleartext, debuggable, excessive permissions,
// debug signing) then fire against the converted application during the
// analysis stage.
func (s *Stage) Run(ctx context.Context, env *core.Env) error {
	if env == nil || env.Target == nil {
		return fmt.Errorf("apk stage requires a target")
	}

	path := env.Target.Address
	if path == "" {
		path = env.Target.Serial
	}
	if path == "" {
		return fmt.Errorf("apk target has no file path")
	}

	parsed, err := apk.Open(path)
	if err != nil {
		return fmt.Errorf("parsing APK %s: %w", path, err)
	}

	app := buildApplication(parsed, path)
	env.Apps = []models.Application{app}

	if env.Log != nil {
		env.Log.Info("apk analysis identified %s (%d components, %d permissions)",
			parsed.Summary(), componentCount(parsed), len(app.Permissions))
	}

	recordEvidence(ctx, env, parsed)

	if env.Events != nil {
		env.Events.Info("jabari", "apk.parsed", map[string]any{
			"package_name": app.PackageName,
			"version_name": app.VersionName,
			"path":         path,
		})
	}
	return nil
}

// buildApplication converts a parsed APK into the normalized inventory model.
func buildApplication(p *apk.APK, path string) models.Application {
	app := models.Application{
		PackageName:   p.PackageName,
		VersionName:   p.VersionName,
		Source:        "apk:" + path,
		Debuggable:    p.Debuggable,
		AllowBackup:   p.AllowBackup,
		UsesCleartext: p.UsesCleartextTraffic,
	}
	if p.VersionCode != 0 {
		app.VersionCode = strconv.FormatInt(p.VersionCode, 10)
	}
	for _, perm := range p.Permissions {
		app.Permissions = append(app.Permissions, perm.Name)
	}
	for _, c := range p.Activities {
		app.Activities = append(app.Activities, c.Name)
	}
	for _, c := range p.Services {
		app.Services = append(app.Services, c.Name)
	}
	for _, c := range p.Receivers {
		app.Receivers = append(app.Receivers, c.Name)
	}
	for _, c := range p.Providers {
		app.Providers = append(app.Providers, c.Name)
	}
	for _, cert := range p.Certificates {
		if cert.SHA256 != "" {
			app.SignerSHA256 = append(app.SignerSHA256, cert.SHA256)
		}
	}

	app.Attributes = map[string]string{
		"manifest_sha256": p.ManifestSHA256,
		"apk_sha256":      p.APKHash,
		"min_sdk":         strconv.Itoa(p.MinSDK),
		"target_sdk":      strconv.Itoa(p.TargetSDK),
		"native_code":     boolString(p.HasNativeCode()),
		"multi_dex":       boolString(p.IsMultiDex()),
	}
	if p.IsDebugSigned() {
		app.Attributes["debug_signed"] = "true"
	}
	if exported := p.ExportedComponents(); len(exported) > 0 {
		app.Attributes["exported_components"] = strings.Join(exported, ",")
	}
	return app
}

// recordEvidence persists the manifest and APK fingerprints to the evidence
// store and streams them as events, mirroring how the discovery stage records
// device information.
func recordEvidence(ctx context.Context, env *core.Env, p *apk.APK) {
	if env.Evidence == nil {
		return
	}
	payload, err := json.MarshalIndent(map[string]any{
		"package_name": p.PackageName,
		"version_name": p.VersionName,
		"version_code": p.VersionCode,
		"min_sdk":      p.MinSDK,
		"target_sdk":   p.TargetSDK,
		"debuggable":   p.Debuggable,
		"allow_backup": p.AllowBackup,
		"cleartext":    p.UsesCleartextTraffic,
		"manifest_sha": p.ManifestSHA256,
		"apk_sha":      p.APKHash,
		"permissions":  len(p.Permissions),
		"activities":   len(p.Activities),
		"services":     len(p.Services),
		"receivers":    len(p.Receivers),
		"providers":    len(p.Providers),
		"native_code":  p.HasNativeCode(),
		"multi_dex":    p.IsMultiDex(),
		"debug_signed": p.IsDebugSigned(),
	}, "", "  ")
	if err != nil {
		if env.Log != nil {
			env.Log.Warn("serializing apk evidence: %v", err)
		}
		return
	}
	ev, err := env.Evidence.Save(ctx, models.KindManifest, p.PackageName, payload)
	if err != nil {
		if env.Log != nil {
			env.Log.Warn("storing apk evidence: %v", err)
		}
		return
	}
	if env.Log != nil {
		env.Log.Debug("apk evidence saved: %s", ev.Hash)
	}
	if env.Events != nil {
		env.Events.Info("jabari", "evidence.collected", map[string]any{
			"source":     p.PackageName,
			"kind":       models.KindManifest,
			"hash":       ev.Hash,
			"finding_id": "apk",
		})
	}
}

func componentCount(p *apk.APK) int {
	return len(p.Activities) + len(p.Services) + len(p.Receivers) + len(p.Providers)
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
