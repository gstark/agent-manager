package importer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseSkillRef(t *testing.T) {
	tests := []struct {
		input       string
		owner, repo string
		skill       string
		wantErr     bool
	}{
		{"mattpocock/skills@tdd", "mattpocock", "skills", "tdd", false},
		{"owner/repo@my-skill", "owner", "repo", "my-skill", false},
		{"invalid", "", "", "", true},
		{"no-at/slash", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ref, err := ParseSkillRef(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if ref.Owner != tt.owner || ref.Repo != tt.repo || ref.Skill != tt.skill {
				t.Errorf("got %+v", ref)
			}
		})
	}
}

func TestSkillRef_URLs(t *testing.T) {
	ref := &SkillRef{Owner: "mattpocock", Repo: "skills", Skill: "tdd"}
	url := ref.RawURL()
	expected := "https://raw.githubusercontent.com/mattpocock/skills/main/tdd/SKILL.md"
	if url != expected {
		t.Errorf("got %q, want %q", url, expected)
	}
}

func TestImportMultiFile(t *testing.T) {
	skillMD := "---\nname: tdd\ndescription: TDD workflow\n---\n\n# TDD\n\nWrite tests first.\n"
	helperSH := "#!/bin/bash\necho hello"

	mux := http.NewServeMux()

	// Raw content for SKILL.md
	mux.HandleFunc("/owner/repo/main/tdd/SKILL.md", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(skillMD))
	})

	// Raw content for helper.sh
	mux.HandleFunc("/raw/helper.sh", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(helperSH))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Contents API: list directory
	contentsMux := http.NewServeMux()
	contentsMux.HandleFunc("/repos/owner/repo/contents/tdd", func(w http.ResponseWriter, r *http.Request) {
		entries := []contentsEntry{
			{Name: "SKILL.md", Type: "file", DownloadURL: ts.URL + "/owner/repo/main/tdd/SKILL.md"},
			{Name: "helper.sh", Type: "file", DownloadURL: ts.URL + "/raw/helper.sh"},
			{Name: "subdir", Type: "dir", DownloadURL: ""},
		}
		json.NewEncoder(w).Encode(entries)
	})
	contentsServer := httptest.NewServer(contentsMux)
	defer contentsServer.Close()

	// Override base URLs
	origRaw := rawContentBase
	origAPI := contentsAPIBase
	rawContentBase = ts.URL
	contentsAPIBase = contentsServer.URL
	defer func() {
		rawContentBase = origRaw
		contentsAPIBase = origAPI
	}()

	ref := &SkillRef{Owner: "owner", Repo: "repo", Skill: "tdd"}
	skill, err := Import(ref)
	if err != nil {
		t.Fatal(err)
	}

	if skill.Name != "tdd" {
		t.Errorf("name: got %q, want %q", skill.Name, "tdd")
	}
	if skill.Description != "TDD workflow" {
		t.Errorf("description: got %q", skill.Description)
	}
	if skill.Body != "# TDD\n\nWrite tests first." {
		t.Errorf("body: got %q", skill.Body)
	}
	if len(skill.Files) != 1 {
		t.Fatalf("expected 1 extra file, got %d", len(skill.Files))
	}
	if string(skill.Files["helper.sh"]) != helperSH {
		t.Errorf("helper.sh: got %q", skill.Files["helper.sh"])
	}
}
