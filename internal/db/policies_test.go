package db

import (
	"os"
	"testing"
)

func TestPolicyCRUD(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", tmp)
	os.MkdirAll(tmp+"/policies", 0755)

	p := &Policy{
		Name:        "no-network",
		Description: "Deny network access",
		Body:        "Do not make network requests.",
	}

	if err := SavePolicy(p); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadPolicy("no-network")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Description != p.Description {
		t.Errorf("description mismatch: got %q", loaded.Description)
	}
	if loaded.Body != p.Body {
		t.Errorf("body mismatch: got %q", loaded.Body)
	}

	policies, _ := ListPolicies()
	if len(policies) != 1 {
		t.Errorf("expected 1 policy, got %d", len(policies))
	}

	DeletePolicy("no-network")
	policies, _ = ListPolicies()
	if len(policies) != 0 {
		t.Errorf("expected 0 policies after delete, got %d", len(policies))
	}
}
