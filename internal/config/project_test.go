package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfig(t *testing.T) {
	tmp := t.TempDir()
	content := `
skills = ["tdd", "debugging"]
rules = ["concise-output"]
packs = ["ruby"]

[[local_rules]]
name = "use-rspec"
description = "Use RSpec"
paths = ["**/*.rb"]
content = "Always use RSpec"
`
	path := filepath.Join(tmp, ".agent-manager.toml")
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := LoadProjectConfig(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(cfg.Skills))
	}
	if len(cfg.Packs) != 1 {
		t.Errorf("expected 1 pack, got %d", len(cfg.Packs))
	}
	if len(cfg.LocalRules) != 1 {
		t.Errorf("expected 1 local rule, got %d", len(cfg.LocalRules))
	}
	if cfg.LocalRules[0].Name != "use-rspec" {
		t.Errorf("expected use-rspec, got %q", cfg.LocalRules[0].Name)
	}
}

func TestLoadProjectConfig_NotFound(t *testing.T) {
	_, err := LoadProjectConfig(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestSaveProjectConfig(t *testing.T) {
	tmp := t.TempDir()
	cfg := &ProjectConfig{
		Skills: []string{"tdd"},
		Rules:  []string{"concise"},
		Packs:  []string{"ruby"},
	}
	if err := SaveProjectConfig(tmp, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadProjectConfig(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Skills) != 1 || loaded.Skills[0] != "tdd" {
		t.Errorf("round-trip failed: %v", loaded.Skills)
	}
}
