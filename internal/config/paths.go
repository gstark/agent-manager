package config

import (
	"os"
	"path/filepath"
)

func ConfigDir() string {
	if d := os.Getenv("AGM_CONFIG_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "agent-manager")
}

func SkillsDir() string        { return filepath.Join(ConfigDir(), "skills") }
func RulesDir() string         { return filepath.Join(ConfigDir(), "rules") }
func PacksDir() string         { return filepath.Join(ConfigDir(), "packs") }
func GlobalConfigPath() string { return filepath.Join(ConfigDir(), "config.toml") }
