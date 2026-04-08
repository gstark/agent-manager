package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
	"github.com/gstark/agent-manager/internal/installer"
)

func TestRemoveRunsInstallToCleanStaleFiles(t *testing.T) {
	configDir := t.TempDir()
	projectDir := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", configDir)
	config.EnsureDirs()

	db.SaveRule(&db.Rule{
		Name:        "concise",
		Description: "Be concise",
		Body:        "Be extremely concise.",
	})
	db.SaveRule(&db.Rule{
		Name:        "verbose",
		Description: "Be verbose",
		Body:        "Be very verbose.",
	})

	// Install both rules
	cfg := &config.ProjectConfig{
		Rules: []string{"concise", "verbose"},
	}
	config.SaveProjectConfig(projectDir, cfg)
	installer.Install(projectDir, cfg)

	// Verify both rule files exist
	for _, name := range []string{"concise", "verbose"} {
		if _, err := os.Stat(filepath.Join(projectDir, ".claude", "rules", name+".md")); err != nil {
			t.Fatalf("rule %s not installed: %v", name, err)
		}
	}

	// Simulate `agm remove rule verbose` by running the command
	origDir, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(origDir)

	removeCmd.SetArgs([]string{"rule", "verbose"})
	if err := removeCmd.RunE(removeCmd, []string{"rule", "verbose"}); err != nil {
		t.Fatalf("remove command failed: %v", err)
	}

	// After remove, the stale rule file should be cleaned up
	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "rules", "concise.md")); err != nil {
		t.Error("kept rule should still exist")
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "rules", "verbose.md")); !os.IsNotExist(err) {
		t.Error("removed rule file should be cleaned up by install")
	}
}
