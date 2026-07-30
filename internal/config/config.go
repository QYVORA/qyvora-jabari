package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const appName = "qyvora-jabari"

type Config struct {
	Verbose bool   `mapstructure:"verbose"`
	Quiet   bool   `mapstructure:"quiet"`
	Output  string `mapstructure:"output"`
	JSON    bool   `mapstructure:"json"`
	Config  string `mapstructure:"config"`
}

func Load(cfgFile string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	configDirs := configSearchDirs(cfgFile)
	for _, dir := range configDirs {
		v.AddConfigPath(dir)
	}

	v.SetEnvPrefix("QYVORA")
	v.AutomaticEnv()

	v.SetDefault("output", "table")
	v.SetDefault("verbose", false)
	v.SetDefault("quiet", false)
	v.SetDefault("json", false)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	return v, nil
}

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
