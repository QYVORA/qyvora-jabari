// Package config loads and exposes assessment configuration through viper.
// Configuration comes from a YAML file, environment variables, and defaults,
// in that order of precedence.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/anomalyco/qyvora-jabari/internal/orchestration"
)

const appName = "qyvora-jabari"

// Profile is a type alias for orchestration.Profile so config callers do not
// need to import orchestration.
type Profile = orchestration.Profile

// Default profile names, aliased from orchestration for convenience.
const (
	ProfileQuick       = orchestration.ProfileQuick
	ProfileStandard    = orchestration.ProfileStandard
	ProfileDeep        = orchestration.ProfileDeep
	ProfileApplication = orchestration.ProfileApplication
	ProfileDevice      = orchestration.ProfileDevice
	ProfileNetwork     = orchestration.ProfileNetwork
	ProfileCompliance  = orchestration.ProfileCompliance
	ProfileResearch    = orchestration.ProfileResearch
)

// Load builds a viper configuration from a config file (when given), the
// QYVORA_* environment namespace, and defaults. A missing config file is not
// an error; a malformed one is.
func Load(cfgFile string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	for _, dir := range configSearchDirs(cfgFile) {
		v.AddConfigPath(dir)
	}

	v.SetEnvPrefix("QYVORA")
	v.AutomaticEnv()

	v.SetDefault("profile", "standard")
	v.SetDefault("output", "table")
	v.SetDefault("verbose", false)
	v.SetDefault("quiet", false)
	v.SetDefault("json", false)
	v.SetDefault("report.dir", "reports")
	v.SetDefault("report.format", "terminal")
	v.SetDefault("audit.enabled", true)
	v.SetDefault("timeout.seconds", 30)
	v.SetDefault("enumeration.detail_limit", 100)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}
	return v, nil
}

// ProfileOf returns the configured profile name, validated against the known
// profile set. An invalid configured profile is an error so a typo cannot
// silently change what an assessment does.
func ProfileOf(v *viper.Viper) (Profile, error) {
	p := v.GetString("profile")
	if !orchestration.IsValid(p) {
		return "", fmt.Errorf("unknown profile %q (valid: %s)", p, profileList())
	}
	return Profile(p), nil
}

func profileList() string {
	out := ""
	for i, p := range orchestration.Profiles {
		if i > 0 {
			out += ", "
		}
		out += string(p)
	}
	return out
}

// configSearchDirs returns the candidate config directories in precedence
// order: an explicit file's directory, the working directory, the per-user
// app dirs, and /etc.
func configSearchDirs(cfgFile string) []string {
	dirs := []string{"."}

	if cfgFile != "" {
		if info, err := os.Stat(cfgFile); err == nil && info.IsDir() {
			dirs = append(dirs, cfgFile)
		} else {
			dirs = append(dirs, filepath.Dir(cfgFile))
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "."+appName))
		dirs = append(dirs, filepath.Join(home, ".config", appName))
	}

	dirs = append(dirs, filepath.Join("/etc", appName))
	return dirs
}
