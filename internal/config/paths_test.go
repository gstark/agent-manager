package config

import "testing"

func TestConfigDir(t *testing.T) {
	dir := ConfigDir()
	if dir == "" {
		t.Fatal("ConfigDir returned empty string")
	}
}

func TestSubdirs(t *testing.T) {
	for _, fn := range []func() string{SkillsDir, RulesDir, PacksDir} {
		d := fn()
		if d == "" {
			t.Fatal("subdir returned empty string")
		}
	}
}
