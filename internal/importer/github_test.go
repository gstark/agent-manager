package importer

import "testing"

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
