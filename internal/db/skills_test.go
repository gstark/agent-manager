package db

import (
	"os"
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
