package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
)

func TestAddRunsInstallToWriteNewFiles(t *testing.T) {
	configDir := t.TempDir()
	projectDir := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", configDir)
	config.EnsureDirs()

	db.SaveRule(&db.Rule{
		Name:        "concise",
		Description: "Be concise",
		Body:        "Be extremely concise.",
	})

	// Create a project config with no rules
	cfg := &config.ProjectConfig{}
	config.SaveProjectConfig(projectDir, cfg)

	// Simulate `agm add rule concise`
	origDir, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(origDir)

	addCmd.SetArgs([]string{"rule", "concise"})
	if err := addCmd.RunE(addCmd, []string{"rule", "concise"}); err != nil {
		t.Fatalf("add command failed: %v", err)
	}

	// After add, the rule file should be written immediately
	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "rules", "concise.md")); err != nil {
		t.Error("added rule file should be installed immediately after add")
	}
}
