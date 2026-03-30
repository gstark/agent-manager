package db

import (
	"os"
	"testing"
)

func TestPackCRUD(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", tmp)
	os.MkdirAll(tmp+"/packs", 0755)

	p := &Pack{
		Name:        "ruby",
		Description: "Ruby development pack",
		Skills:      []string{"tdd", "debugging"},
		Rules:       []string{"ruby-conventions"},
	}

	if err := SavePack(p); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadPack("ruby")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Description != p.Description {
		t.Errorf("description mismatch")
	}
	if len(loaded.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(loaded.Skills))
	}

	packs, _ := ListPacks()
	if len(packs) != 1 {
		t.Errorf("expected 1 pack, got %d", len(packs))
	}

	DeletePack("ruby")
	packs, _ = ListPacks()
	if len(packs) != 0 {
		t.Errorf("expected 0 packs, got %d", len(packs))
	}
}
