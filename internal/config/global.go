package config

import (
	"errors"
	"os"

	"github.com/BurntSushi/toml"
)

type GlobalConfig struct {
	GitHub   GitHubConfig   `toml:"github"`
	Defaults DefaultsConfig `toml:"defaults"`
}

type GitHubConfig struct {
	Token string `toml:"token"`
}

type DefaultsConfig struct {
	Editor string `toml:"editor"`
}

func LoadGlobalConfig() (*GlobalConfig, error) {
	cfg := &GlobalConfig{}
	path := GlobalConfigPath()

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func EnsureDirs() error {
	for _, dir := range []string{ConfigDir(), SkillsDir(), RulesDir(), PacksDir()} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
