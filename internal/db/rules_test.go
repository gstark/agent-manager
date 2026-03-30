package db

import (
	"os"
	"testing"
)

func TestRuleCRUD(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", tmp)
	os.MkdirAll(tmp+"/rules", 0755)

	r := &Rule{
		Name:        "ruby-conventions",
		Description: "Ruby style rules",
		Paths:       []string{"**/*.rb"},
		Body:        "Use snake_case for methods.",
	}

	if err := SaveRule(r); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadRule("ruby-conventions")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Paths) != 1 || loaded.Paths[0] != "**/*.rb" {
		t.Errorf("paths mismatch: %v", loaded.Paths)
	}
	if loaded.Body != r.Body {
		t.Errorf("body mismatch: got %q", loaded.Body)
	}

	rules, _ := ListRules()
	if len(rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(rules))
	}

	DeleteRule("ruby-conventions")
	rules, _ = ListRules()
	if len(rules) != 0 {
		t.Errorf("expected 0 rules after delete, got %d", len(rules))
	}
}
