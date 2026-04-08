package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
)

func setupTestEnv(t *testing.T) (configDir, projectDir string) {
	t.Helper()
	configDir = t.TempDir()
	projectDir = t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", configDir)
	config.EnsureDirs()

	// Create a skill with extra files
	db.SaveSkill(&db.Skill{
		Name:        "tdd",
		Description: "TDD workflow",
		Source:      "local",
		Body:        "# TDD\nWrite tests first.",
		Files: map[string][]byte{
			"helper.sh": []byte("#!/bin/bash\necho test"),
		},
	})

	// Create a rule (prompt instruction)
	db.SaveRule(&db.Rule{
		Name:        "concise",
		Description: "Be concise",
		Body:        "Be extremely concise.",
	})

	// Create a rule with paths
	db.SaveRule(&db.Rule{
		Name:        "ruby-style",
		Description: "Ruby style",
		Paths:       []string{"**/*.rb"},
		Body:        "Use snake_case.",
	})

	// Create a policy (Codex execution rule)
	db.SavePolicy(&db.Policy{
		Name:        "no-network",
		Description: "Deny network access",
		Body:        "Do not make network requests.",
	})

	// Create a pack
	db.SavePack(&db.Pack{
		Name:   "ruby",
		Skills: []string{"tdd"},
		Rules:  []string{"ruby-style"},
	})

	return configDir, projectDir
}

func TestInstall(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	cfg := &config.ProjectConfig{
		Skills:   []string{"tdd"},
		Rules:    []string{"concise"},
		Policies: []string{"no-network"},
		Packs:    []string{"ruby"},
	}

	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	// AGENTS.md should exist
	if _, err := os.Stat(filepath.Join(projectDir, "AGENTS.md")); err != nil {
		t.Error("AGENTS.md not created")
	}

	// CLAUDE.md should be a regular file (not symlink) containing @AGENTS.md
	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	info, err := os.Lstat(claudePath)
	if err != nil {
		t.Fatal("CLAUDE.md not created")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("CLAUDE.md should not be a symlink")
	}
	claudeContent, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(claudeContent), "@AGENTS.md") {
		t.Error("CLAUDE.md should contain @AGENTS.md import")
	}

	// Claude rules should exist
	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "rules", "ruby-style.md")); err != nil {
		t.Error("claude rule not created")
	}

	// Canonical skill should exist under .agm/skills/
	canonicalSkill := filepath.Join(projectDir, ".agm", "skills", "tdd", "SKILL.md")
	if _, err := os.Stat(canonicalSkill); err != nil {
		t.Error("canonical skill not created")
	}

	// Canonical extra files should exist
	canonicalHelper := filepath.Join(projectDir, ".agm", "skills", "tdd", "helper.sh")
	if _, err := os.Stat(canonicalHelper); err != nil {
		t.Error("canonical skill extra file not created")
	}

	// Claude skills should be symlinks to canonical
	claudeSkillDir := filepath.Join(projectDir, ".claude", "skills", "tdd")
	target, err := os.Readlink(claudeSkillDir)
	if err != nil {
		t.Error("claude skill dir is not a symlink")
	} else {
		expected := filepath.Join("..", "..", ".agm", "skills", "tdd")
		if target != expected {
			t.Errorf("claude skill symlink points to %q, want %q", target, expected)
		}
	}

	// Claude skill content should be readable through symlink
	if _, err := os.Stat(filepath.Join(claudeSkillDir, "SKILL.md")); err != nil {
		t.Error("claude skill SKILL.md not readable through symlink")
	}
	if _, err := os.Stat(filepath.Join(claudeSkillDir, "helper.sh")); err != nil {
		t.Error("claude skill helper.sh not readable through symlink")
	}

	// Codex skills should be symlinks to canonical
	codexSkillDir := filepath.Join(projectDir, ".agents", "skills", "tdd")
	codexTarget, codexErr := os.Readlink(codexSkillDir)
	if codexErr != nil {
		t.Error("codex skill dir is not a symlink")
	} else {
		expected := filepath.Join("..", "..", ".agm", "skills", "tdd")
		if codexTarget != expected {
			t.Errorf("codex skill symlink points to %q, want %q", codexTarget, expected)
		}
	}

	// Codex skill content should be readable through symlink
	if _, err := os.Stat(filepath.Join(codexSkillDir, "SKILL.md")); err != nil {
		t.Error("codex skill SKILL.md not readable through symlink")
	}
	if _, err := os.Stat(filepath.Join(codexSkillDir, "helper.sh")); err != nil {
		t.Error("codex skill helper.sh not readable through symlink")
	}
}

func TestInstallCanonicalSkillContent(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	cfg := &config.ProjectConfig{
		Skills: []string{"tdd"},
	}

	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Canonical SKILL.md should contain frontmatter and body
	content, err := os.ReadFile(filepath.Join(projectDir, ".agm", "skills", "tdd", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	if !strings.Contains(s, "name: tdd") {
		t.Error("canonical SKILL.md missing name")
	}
	if !strings.Contains(s, "Write tests first") {
		t.Error("canonical SKILL.md missing body")
	}

	// Helper file content should match
	helper, err := os.ReadFile(filepath.Join(projectDir, ".agm", "skills", "tdd", "helper.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if string(helper) != "#!/bin/bash\necho test" {
		t.Errorf("helper content = %q", string(helper))
	}
}

func TestInstallSymlinkCopyFallback(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	cfg := &config.ProjectConfig{
		Skills: []string{"tdd"},
	}

	// Install normally first
	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Replace a symlink with a regular directory to simulate broken symlink env
	claudeSkillDir := filepath.Join(projectDir, ".claude", "skills", "tdd")
	os.Remove(claudeSkillDir) // remove the symlink
	os.MkdirAll(claudeSkillDir, 0755)
	os.WriteFile(filepath.Join(claudeSkillDir, "stale.txt"), []byte("stale"), 0644)

	// Re-install should fix it (back to symlink since symlinks work here)
	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Should be a symlink again
	if _, err := os.Readlink(claudeSkillDir); err != nil {
		t.Error("claude skill dir should be a symlink after re-install")
	}
}

func TestInstallIdempotent(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	cfg := &config.ProjectConfig{
		Skills:   []string{"tdd"},
		Rules:    []string{"concise"},
		Policies: []string{"no-network"},
	}

	results1, err := Install(projectDir, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// First install: everything should be StatusInstalled
	for _, r := range results1 {
		if r.Status != StatusInstalled {
			t.Errorf("first install: %s %q got status %d, want StatusInstalled", r.Kind, r.Name, r.Status)
		}
	}

	results2, err := Install(projectDir, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Second install: everything should be StatusUpToDate
	for _, r := range results2 {
		if r.Status != StatusUpToDate {
			t.Errorf("second install: %s %q got status %d, want StatusUpToDate", r.Kind, r.Name, r.Status)
		}
	}

	// Codex rules should exist for policies
	codexRulePath := filepath.Join(projectDir, ".codex", "rules", "no-network.md")
	if _, err := os.Stat(codexRulePath); err != nil {
		t.Error(".codex/rules/no-network.md not created")
	}

	// Policies should NOT appear in AGENTS.md (they're execution rules, not instructions)
	agentsContent, _ := os.ReadFile(filepath.Join(projectDir, "AGENTS.md"))
	if strings.Contains(string(agentsContent), "no-network") {
		t.Error("policy should not appear in AGENTS.md")
	}
	if strings.Contains(string(agentsContent), "network requests") {
		t.Error("policy content should not leak into AGENTS.md")
	}

	// Prompt instructions SHOULD appear in AGENTS.md
	if !strings.Contains(string(agentsContent), "Be extremely concise") {
		t.Error("prompt instruction should appear in AGENTS.md")
	}
}

func TestInstallWithLocalRules(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	cfg := &config.ProjectConfig{
		LocalRules: []config.LocalRule{
			{
				Name:        "use-rspec",
				Description: "Use RSpec",
				Paths:       []string{"**/*.rb"},
				Content:     "Always use RSpec.",
			},
		},
	}

	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "rules", "use-rspec.md")); err != nil {
		t.Error("local rule not created in claude rules")
	}
}

func TestInstallBackwardCompatibility(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	// Existing config with only rules (no policies) should still work
	cfg := &config.ProjectConfig{
		Skills: []string{"tdd"},
		Rules:  []string{"concise"},
		Packs:  []string{"ruby"},
	}

	results, err := Install(projectDir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Error("expected install results")
	}

	// AGENTS.md and CLAUDE.md should still be created
	if _, err := os.Stat(filepath.Join(projectDir, "AGENTS.md")); err != nil {
		t.Error("AGENTS.md not created")
	}
	if _, err := os.Stat(filepath.Join(projectDir, "CLAUDE.md")); err != nil {
		t.Error("CLAUDE.md not created")
	}

	// No .codex/rules/ should exist when no policies configured
	if _, err := os.Stat(filepath.Join(projectDir, ".codex", "rules")); err == nil {
		t.Error(".codex/rules/ should not exist when no policies configured")
	}
}

func TestInstallClaudeRuleFrontmatter(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	cfg := &config.ProjectConfig{
		Rules: []string{"ruby-style"},
	}

	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(projectDir, ".claude", "rules", "ruby-style.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)
	if !strings.Contains(s, "paths:") {
		t.Error("scoped rule should include paths frontmatter")
	}
	if !strings.Contains(s, "**/*.rb") {
		t.Error("scoped rule should include path pattern")
	}
}

func TestInstallPackWithPolicies(t *testing.T) {
	configDir := t.TempDir()
	projectDir := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", configDir)
	config.EnsureDirs()

	db.SavePolicy(&db.Policy{
		Name:        "sandbox",
		Description: "Run in sandbox",
		Body:        "Always use sandboxed execution.",
	})

	db.SavePack(&db.Pack{
		Name:     "secure",
		Policies: []string{"sandbox"},
	})

	cfg := &config.ProjectConfig{
		Packs: []string{"secure"},
	}

	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".codex", "rules", "sandbox.md")); err != nil {
		t.Error(".codex/rules/sandbox.md not created from pack")
	}
}
