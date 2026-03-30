package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGlobalConfig_Default(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", tmp)

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Defaults.Editor != "" {
		t.Errorf("expected empty editor, got %q", cfg.Defaults.Editor)
	}
}

func TestLoadGlobalConfig_FromFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", tmp)

	content := `[defaults]
editor = "nvim"
`
	os.WriteFile(filepath.Join(tmp, "config.toml"), []byte(content), 0644)

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Defaults.Editor != "nvim" {
		t.Errorf("expected nvim, got %q", cfg.Defaults.Editor)
	}
}

func TestEnsureDirs(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", tmp)

	if err := EnsureDirs(); err != nil {
		t.Fatal(err)
	}

	for _, sub := range []string{"skills", "rules", "packs"} {
		p := filepath.Join(tmp, sub)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected dir %s to exist", sub)
		}
	}
}
