package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillCRUD(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", tmp)
	os.MkdirAll(tmp+"/skills", 0755)

	s := &Skill{
		Name:        "tdd",
		Description: "Test-driven development",
		Source:      "local",
		Body:        "# TDD\n\nWrite tests first.",
	}

	// Create
	if err := SaveSkill(s); err != nil {
		t.Fatal(err)
	}

	// Verify directory structure
	if _, err := os.Stat(filepath.Join(tmp, "skills", "tdd", "SKILL.md")); err != nil {
		t.Fatal("expected SKILL.md in skill directory")
	}

	// Read
	loaded, err := LoadSkill("tdd")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Description != s.Description {
		t.Errorf("expected %q, got %q", s.Description, loaded.Description)
	}
	if loaded.Body != s.Body {
		t.Errorf("body mismatch: got %q", loaded.Body)
	}

	// List
	skills, err := ListSkills()
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(skills))
	}

	// Delete
	if err := DeleteSkill("tdd"); err != nil {
		t.Fatal(err)
	}
	skills, _ = ListSkills()
	if len(skills) != 0 {
		t.Errorf("expected 0 skills after delete, got %d", len(skills))
	}
}

func TestSkillWithExtraFiles(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", tmp)
	os.MkdirAll(tmp+"/skills", 0755)

	s := &Skill{
		Name:        "multi",
		Description: "Multi-file skill",
		Source:      "skills.sh/owner/repo@multi",
		Body:        "# Multi\n\nA skill with extra files.",
		Files: map[string][]byte{
			"helper.sh":   []byte("#!/bin/bash\necho hello"),
			"template.md": []byte("# Template"),
		},
	}

	if err := SaveSkill(s); err != nil {
		t.Fatal(err)
	}

	// Verify extra files on disk
	for name, want := range s.Files {
		got, err := os.ReadFile(filepath.Join(tmp, "skills", "multi", name))
		if err != nil {
			t.Fatalf("extra file %q not found: %v", name, err)
		}
		if string(got) != string(want) {
			t.Errorf("file %q: got %q, want %q", name, got, want)
		}
	}

	// Load and verify Files are populated
	loaded, err := LoadSkill("multi")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Files) != 2 {
		t.Errorf("expected 2 extra files, got %d", len(loaded.Files))
	}
	if string(loaded.Files["helper.sh"]) != "#!/bin/bash\necho hello" {
		t.Errorf("helper.sh mismatch: %q", loaded.Files["helper.sh"])
	}

	// Update: remove one file, add another
	s.Files = map[string][]byte{
		"helper.sh": []byte("#!/bin/bash\necho updated"),
		"new.txt":   []byte("new file"),
	}
	if err := SaveSkill(s); err != nil {
		t.Fatal(err)
	}

	// template.md should be removed
	if _, err := os.Stat(filepath.Join(tmp, "skills", "multi", "template.md")); !os.IsNotExist(err) {
		t.Error("stale file template.md should have been removed")
	}

	loaded, err = LoadSkill("multi")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Files) != 2 {
		t.Errorf("expected 2 extra files after update, got %d", len(loaded.Files))
	}

	// Delete removes everything
	if err := DeleteSkill("multi"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "skills", "multi")); !os.IsNotExist(err) {
		t.Error("skill directory should be removed after delete")
	}
}

func TestContentHashRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", tmp)
	os.MkdirAll(tmp+"/skills", 0755)

	s := &Skill{
		Name:        "hashed",
		Description: "A skill with a content hash",
		Source:      "skills.sh/owner/repo@hashed",
		Body:        "# Hashed skill",
		ContentHash: "abc123def456",
	}

	if err := SaveSkill(s); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadSkill("hashed")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ContentHash != "abc123def456" {
		t.Errorf("ContentHash: got %q, want %q", loaded.ContentHash, "abc123def456")
	}
}

func TestContentHashEmptyForLegacySkills(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", tmp)
	skillDir := filepath.Join(tmp, "skills", "legacy")
	os.MkdirAll(skillDir, 0755)

	// Write a SKILL.md without content_hash in frontmatter
	content := "---\nname: legacy\ndescription: Old skill\nsource: skills.sh/owner/repo@legacy\n---\n\n# Legacy"
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644)

	loaded, err := LoadSkill("legacy")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ContentHash != "" {
		t.Errorf("expected empty ContentHash for legacy skill, got %q", loaded.ContentHash)
	}
}

func TestComputeContentHash(t *testing.T) {
	body := "# My Skill"
	files := map[string][]byte{
		"helper.sh":   []byte("#!/bin/bash\necho hello"),
		"template.md": []byte("# Template"),
	}

	hash1 := ComputeContentHash(body, files)
	if hash1 == "" {
		t.Fatal("expected non-empty hash")
	}

	// Same content → same hash (deterministic)
	hash2 := ComputeContentHash(body, files)
	if hash1 != hash2 {
		t.Errorf("hash not deterministic: %q != %q", hash1, hash2)
	}

	// Different body → different hash
	hash3 := ComputeContentHash("# Changed", files)
	if hash3 == hash1 {
		t.Error("expected different hash when body changes")
	}

	// Different file content → different hash
	files2 := map[string][]byte{
		"helper.sh":   []byte("#!/bin/bash\necho changed"),
		"template.md": []byte("# Template"),
	}
	hash4 := ComputeContentHash(body, files2)
	if hash4 == hash1 {
		t.Error("expected different hash when file content changes")
	}

	// No extra files → still works
	hash5 := ComputeContentHash(body, nil)
	if hash5 == "" {
		t.Fatal("expected non-empty hash with no extra files")
	}
	if hash5 == hash1 {
		t.Error("expected different hash when files are removed")
	}
}
