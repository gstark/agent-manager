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

	// Create a rule
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
		Skills: []string{"tdd"},
		Rules:  []string{"concise"},
		Packs:  []string{"ruby"},
	}

	if _, err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	// AGENTS.md should exist
	if _, err := os.Stat(filepath.Join(projectDir, "AGENTS.md")); err != nil {
		t.Error("AGENTS.md not created")
	}

	// CLAUDE.md should be a symlink to AGENTS.md
	target, err := os.Readlink(filepath.Join(projectDir, "CLAUDE.md"))
	if err != nil {
		t.Error("CLAUDE.md is not a symlink")
	} else if target != "AGENTS.md" {
		t.Errorf("CLAUDE.md points to %q, expected AGENTS.md", target)
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
	target, err = os.Readlink(claudeSkillDir)
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
		Skills: []string{"tdd"},
		Rules:  []string{"concise"},
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
