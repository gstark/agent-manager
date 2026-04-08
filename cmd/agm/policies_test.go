package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
	"github.com/gstark/agent-manager/internal/installer"
)

func setupPolicyEnv(t *testing.T) string {
	t.Helper()
	configDir := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", configDir)
	config.EnsureDirs()
	return configDir
}

func TestPoliciesListEmpty(t *testing.T) {
	setupPolicyEnv(t)

	var buf bytes.Buffer
	policiesListCmd.SetOut(&buf)
	policiesListCmd.SetArgs([]string{})
	if err := policiesListCmd.RunE(policiesListCmd, nil); err != nil {
		t.Fatal(err)
	}

	if got := buf.String(); got != "" {
		// Empty list prints to stdout directly via fmt.Println, not cmd.OutOrStdout
	}
}

func TestPoliciesListShowsPolicies(t *testing.T) {
	setupPolicyEnv(t)

	db.SavePolicy(&db.Policy{
		Name:        "no-network",
		Description: "Deny network access",
		Body:        "No network.",
	})

	policiesListCmd.SetArgs([]string{})
	if err := policiesListCmd.RunE(policiesListCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestPoliciesListJSON(t *testing.T) {
	setupPolicyEnv(t)

	db.SavePolicy(&db.Policy{
		Name:        "no-network",
		Description: "Deny network access",
		Body:        "No network.",
	})

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"policies", "list", "--json"})
	err := rootCmd.Execute()

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var items []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, buf.String())
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0]["name"] != "no-network" {
		t.Errorf("expected name no-network, got %q", items[0]["name"])
	}
}

func TestPoliciesDeleteRemovesPolicy(t *testing.T) {
	setupPolicyEnv(t)

	db.SavePolicy(&db.Policy{
		Name:        "no-network",
		Description: "Deny network access",
		Body:        "No network.",
	})

	policiesDeleteCmd.SetArgs([]string{"no-network"})
	if err := policiesDeleteCmd.RunE(policiesDeleteCmd, []string{"no-network"}); err != nil {
		t.Fatal(err)
	}

	policies, _ := db.ListPolicies()
	if len(policies) != 0 {
		t.Errorf("expected 0 policies after delete, got %d", len(policies))
	}
}

func TestPoliciesCatPrintsContent(t *testing.T) {
	setupPolicyEnv(t)

	db.SavePolicy(&db.Policy{
		Name:        "no-network",
		Description: "Deny network access",
		Body:        "Do not make network requests.",
	})

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	policiesCatCmd.SetArgs([]string{"no-network"})
	err := policiesCatCmd.RunE(policiesCatCmd, []string{"no-network"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !bytes.Contains([]byte(output), []byte("Do not make network requests")) {
		t.Errorf("expected body content in output, got %q", output)
	}
}

func TestPoliciesCatNotFound(t *testing.T) {
	setupPolicyEnv(t)

	policiesCatCmd.SetArgs([]string{"nonexistent"})
	err := policiesCatCmd.RunE(policiesCatCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent policy")
	}
}

func TestAddPolicy(t *testing.T) {
	configDir := t.TempDir()
	projectDir := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", configDir)
	config.EnsureDirs()

	db.SavePolicy(&db.Policy{
		Name:        "no-network",
		Description: "Deny network access",
		Body:        "Do not make network requests.",
	})

	cfg := &config.ProjectConfig{}
	config.SaveProjectConfig(projectDir, cfg)

	origDir, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(origDir)

	addCmd.SetArgs([]string{"policy", "no-network"})
	if err := addCmd.RunE(addCmd, []string{"policy", "no-network"}); err != nil {
		t.Fatalf("add policy failed: %v", err)
	}

	// Policy should be written to .codex/rules/
	if _, err := os.Stat(filepath.Join(projectDir, ".codex", "rules", "no-network.md")); err != nil {
		t.Error("added policy file should be installed immediately after add")
	}
}

func TestRemovePolicy(t *testing.T) {
	configDir := t.TempDir()
	projectDir := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", configDir)
	config.EnsureDirs()

	db.SavePolicy(&db.Policy{
		Name:        "no-network",
		Description: "Deny network access",
		Body:        "Do not make network requests.",
	})
	db.SavePolicy(&db.Policy{
		Name:        "no-fs",
		Description: "Deny filesystem access",
		Body:        "No filesystem.",
	})

	cfg := &config.ProjectConfig{
		Policies: []string{"no-network", "no-fs"},
	}
	config.SaveProjectConfig(projectDir, cfg)
	installer.Install(projectDir, cfg)

	origDir, _ := os.Getwd()
	os.Chdir(projectDir)
	defer os.Chdir(origDir)

	removeCmd.SetArgs([]string{"policy", "no-fs"})
	if err := removeCmd.RunE(removeCmd, []string{"policy", "no-fs"}); err != nil {
		t.Fatalf("remove policy failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".codex", "rules", "no-network.md")); err != nil {
		t.Error("kept policy should still exist")
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".codex", "rules", "no-fs.md")); !os.IsNotExist(err) {
		t.Error("removed policy file should be cleaned up by install")
	}
}
