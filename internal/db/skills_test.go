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
