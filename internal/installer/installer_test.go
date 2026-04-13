package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/frontmatter"
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
			"helper.sh":            []byte("#!/bin/bash\necho test"),
			"references/guide.md":  []byte("# Guide\nUse red-green-refactor."),
			"scripts/bootstrap.sh": []byte("#!/bin/bash\necho bootstrap"),
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
	canonicalNested := filepath.Join(projectDir, ".agm", "skills", "tdd", "references", "guide.md")
	if _, err := os.Stat(canonicalNested); err != nil {
		t.Error("canonical nested skill extra file not created")
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
	if _, err := os.Stat(filepath.Join(claudeSkillDir, "references", "guide.md")); err != nil {
		t.Error("claude nested skill file not readable through symlink")
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
	if _, err := os.Stat(filepath.Join(codexSkillDir, "references", "guide.md")); err != nil {
		t.Error("codex nested skill file not readable through symlink")
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

	guide, err := os.ReadFile(filepath.Join(projectDir, ".agm", "skills", "tdd", "references", "guide.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(guide) != "# Guide\nUse red-green-refactor." {
		t.Errorf("guide content = %q", string(guide))
	}
}

func TestInstallCanonicalSkillQuotesDescription(t *testing.T) {
	configDir := t.TempDir()
	projectDir := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", configDir)
	config.EnsureDirs()

	// Description with YAML-special characters: colons, quotes, hash
	db.SaveSkill(&db.Skill{
		Name:        "tricky",
		Description: `Deploy: "production" #1 server`,
		Source:      "local",
		Body:        "Handle with care.",
	})

	cfg := &config.ProjectConfig{
		Skills: []string{"tricky"},
	}

	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(projectDir, ".agm", "skills", "tricky", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}

	// Re-parse to verify the frontmatter round-trips correctly
	var parsed db.Skill
	_, err = frontmatter.Parse(bytes.NewReader(content), &parsed)
	if err != nil {
		t.Fatalf("failed to re-parse installed SKILL.md: %v\ncontent:\n%s", err, content)
	}
	if parsed.Description != `Deploy: "production" #1 server` {
		t.Errorf("description round-trip failed: got %q", parsed.Description)
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

func TestInstallRemovesStaleNestedCanonicalSkillFiles(t *testing.T) {
	configDir, projectDir := setupTestEnv(t)

	cfg := &config.ProjectConfig{
		Skills: []string{"tdd"},
	}

	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	db.SaveSkill(&db.Skill{
		Name:        "tdd",
		Description: "TDD workflow",
		Source:      "local",
		Body:        "# TDD\nWrite tests first.",
		Files: map[string][]byte{
			"helper.sh": []byte("#!/bin/bash\necho updated"),
		},
	})
	t.Setenv("AGM_CONFIG_DIR", configDir)

	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".agm", "skills", "tdd", "references", "guide.md")); !os.IsNotExist(err) {
		t.Error("stale nested canonical skill file should be removed on reinstall")
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".agm", "skills", "tdd", "scripts")); !os.IsNotExist(err) {
		t.Error("empty nested canonical skill directory should be removed on reinstall")
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

// mockAdapter records whether it was called and what artifacts it received.
type mockAdapter struct {
	name      string
	called    bool
	artifacts *ArtifactSet
}

func (m *mockAdapter) Name() string { return m.name }
func (m *mockAdapter) Install(projectDir string, a *ArtifactSet) ([]ItemResult, error) {
	m.called = true
	m.artifacts = a
	return []ItemResult{{Kind: "mock", Name: m.name, Status: StatusInstalled}}, nil
}

func TestAdapterDispatchCombined(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	cfg := &config.ProjectConfig{
		Skills:   []string{"tdd"},
		Rules:    []string{"concise"},
		Policies: []string{"no-network"},
	}

	// Default adapters + a custom third-tool adapter
	mock := &mockAdapter{name: "third-tool"}
	adapters := append(DefaultAdapters, mock)

	results, err := InstallWith(projectDir, cfg, adapters)
	if err != nil {
		t.Fatal(err)
	}

	// Claude output should exist
	if _, err := os.Stat(filepath.Join(projectDir, "CLAUDE.md")); err != nil {
		t.Error("CLAUDE.md not created by Claude adapter")
	}

	// Codex output should exist
	if _, err := os.Stat(filepath.Join(projectDir, "AGENTS.md")); err != nil {
		t.Error("AGENTS.md not created by Codex adapter")
	}

	// Third-tool adapter should have been called with artifacts
	if !mock.called {
		t.Fatal("third-tool adapter was not called")
	}
	if len(mock.artifacts.Skills) != 1 {
		t.Errorf("third-tool got %d skills, want 1", len(mock.artifacts.Skills))
	}

	// Results should include output from all three adapters
	kinds := map[string]bool{}
	for _, r := range results {
		kinds[r.Kind] = true
	}
	if !kinds["mock"] {
		t.Error("results missing third-tool adapter output")
	}
	if !kinds["rule"] {
		t.Error("results missing claude adapter output")
	}
	if !kinds["skill"] {
		t.Error("results missing canonical skill output")
	}
}

func TestInstallCleansStaleClaudeRules(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	// Install with two rules
	cfg := &config.ProjectConfig{
		Rules: []string{"concise", "ruby-style"},
	}
	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Both rules should exist
	for _, name := range []string{"concise", "ruby-style"} {
		if _, err := os.Stat(filepath.Join(projectDir, ".claude", "rules", name+".md")); err != nil {
			t.Fatalf("rule %s not created", name)
		}
	}

	// Re-install with only one rule — stale rule should be removed
	cfg2 := &config.ProjectConfig{
		Rules: []string{"concise"},
	}
	if _, err := Install(projectDir, cfg2); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "rules", "concise.md")); err != nil {
		t.Error("kept rule should still exist")
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "rules", "ruby-style.md")); !os.IsNotExist(err) {
		t.Error("stale claude rule should be removed")
	}
}

func TestInstallCleansStaleClaudeSkills(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	// Add a second skill
	db.SaveSkill(&db.Skill{
		Name:        "review",
		Description: "Code review",
		Source:      "local",
		Body:        "# Review\nReview code.",
	})

	cfg := &config.ProjectConfig{
		Skills: []string{"tdd", "review"},
	}
	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Both skills should exist
	for _, name := range []string{"tdd", "review"} {
		if _, err := os.Stat(filepath.Join(projectDir, ".claude", "skills", name)); err != nil {
			t.Fatalf("skill %s not created", name)
		}
		if _, err := os.Stat(filepath.Join(projectDir, ".agm", "skills", name)); err != nil {
			t.Fatalf("canonical skill %s not created", name)
		}
	}

	// Re-install with only tdd — review should be cleaned up everywhere
	cfg2 := &config.ProjectConfig{
		Skills: []string{"tdd"},
	}
	if _, err := Install(projectDir, cfg2); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "skills", "tdd")); err != nil {
		t.Error("kept skill should still exist in .claude/skills/")
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "skills", "review")); !os.IsNotExist(err) {
		t.Error("stale skill should be removed from .claude/skills/")
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".agm", "skills", "review")); !os.IsNotExist(err) {
		t.Error("stale skill should be removed from .agm/skills/")
	}
}

func TestInstallCleansStaleCodexPolicies(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	// Add a second policy
	db.SavePolicy(&db.Policy{
		Name:        "no-fs",
		Description: "Deny filesystem access",
		Body:        "Do not write to disk.",
	})

	cfg := &config.ProjectConfig{
		Policies: []string{"no-network", "no-fs"},
	}
	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"no-network", "no-fs"} {
		if _, err := os.Stat(filepath.Join(projectDir, ".codex", "rules", name+".md")); err != nil {
			t.Fatalf("policy %s not created", name)
		}
	}

	// Re-install without no-fs
	cfg2 := &config.ProjectConfig{
		Policies: []string{"no-network"},
	}
	if _, err := Install(projectDir, cfg2); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".codex", "rules", "no-network.md")); err != nil {
		t.Error("kept policy should still exist")
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".codex", "rules", "no-fs.md")); !os.IsNotExist(err) {
		t.Error("stale codex policy should be removed")
	}
}

func TestInstallCleansStaleCodexSkills(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	db.SaveSkill(&db.Skill{
		Name:        "review",
		Description: "Code review",
		Source:      "local",
		Body:        "# Review\nReview code.",
	})

	cfg := &config.ProjectConfig{
		Skills: []string{"tdd", "review"},
	}
	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".agents", "skills", "review")); err != nil {
		t.Fatal("codex skill not created")
	}

	cfg2 := &config.ProjectConfig{
		Skills: []string{"tdd"},
	}
	if _, err := Install(projectDir, cfg2); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".agents", "skills", "tdd")); err != nil {
		t.Error("kept skill should still exist in .agents/skills/")
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".agents", "skills", "review")); !os.IsNotExist(err) {
		t.Error("stale skill should be removed from .agents/skills/")
	}
}

func TestInstallPreservesNonManagedFiles(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	cfg := &config.ProjectConfig{
		Rules: []string{"concise"},
	}
	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Place a non-.md file in .claude/rules/ — should NOT be removed
	foreignFile := filepath.Join(projectDir, ".claude", "rules", "user-custom.txt")
	os.WriteFile(foreignFile, []byte("custom"), 0644)

	// Re-install with empty rules
	cfg2 := &config.ProjectConfig{}
	if _, err := Install(projectDir, cfg2); err != nil {
		t.Fatal(err)
	}

	// concise.md should be gone
	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "rules", "concise.md")); !os.IsNotExist(err) {
		t.Error("stale rule should be removed")
	}
	// Non-.md file should be preserved
	if _, err := os.Stat(foreignFile); err != nil {
		t.Error("non-managed file should be preserved")
	}
}

func TestAdapterDispatch(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	cfg := &config.ProjectConfig{
		Skills:   []string{"tdd"},
		Rules:    []string{"concise"},
		Policies: []string{"no-network"},
	}

	mock := &mockAdapter{name: "test-tool"}
	results, err := InstallWith(projectDir, cfg, []ToolAdapter{mock})
	if err != nil {
		t.Fatal(err)
	}

	if !mock.called {
		t.Fatal("adapter was not called")
	}

	// ArtifactSet should contain resolved skills, rules, policies
	if len(mock.artifacts.Skills) != 1 || mock.artifacts.Skills[0].Name != "tdd" {
		t.Errorf("expected 1 skill 'tdd', got %v", mock.artifacts.Skills)
	}
	if len(mock.artifacts.Rules) != 1 || mock.artifacts.Rules[0].Name != "concise" {
		t.Errorf("expected 1 rule 'concise', got %v", mock.artifacts.Rules)
	}
	if len(mock.artifacts.Policies) != 1 || mock.artifacts.Policies[0].Name != "no-network" {
		t.Errorf("expected 1 policy 'no-network', got %v", mock.artifacts.Policies)
	}

	// Should include results from the adapter
	found := false
	for _, r := range results {
		if r.Kind == "mock" && r.Name == "test-tool" {
			found = true
		}
	}
	if !found {
		t.Error("adapter results not included in Install output")
	}
}
