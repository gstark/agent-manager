package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const ProjectConfigFile = ".agent-manager.toml"

type LocalRule struct {
	Name        string   `toml:"name"`
	Description string   `toml:"description"`
	Paths       []string `toml:"paths"`
	Content     string   `toml:"content"`
}

type ProjectConfig struct {
	Skills     []string    `toml:"skills"`
	Rules      []string    `toml:"rules"`
	Packs      []string    `toml:"packs"`
	LocalRules []LocalRule `toml:"local_rules"`
}

func ProjectConfigPath(dir string) string {
	return filepath.Join(dir, ProjectConfigFile)
}

func LoadProjectConfig(dir string) (*ProjectConfig, error) {
	cfg := &ProjectConfig{}
	if _, err := toml.DecodeFile(ProjectConfigPath(dir), cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func SaveProjectConfig(dir string, cfg *ProjectConfig) error {
	f, err := os.Create(ProjectConfigPath(dir))
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
