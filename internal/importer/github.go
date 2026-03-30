package importer

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/gstark/agent-manager/internal/db"
)

type SkillRef struct {
	Owner string
	Repo  string
	Skill string
}

func ParseSkillRef(s string) (*SkillRef, error) {
	// Format: owner/repo@skill
	atIdx := strings.LastIndex(s, "@")
	if atIdx < 0 {
		return nil, fmt.Errorf("invalid skill ref %q: expected owner/repo@skill", s)
	}
	repoPath := s[:atIdx]
	skill := s[atIdx+1:]

	parts := strings.SplitN(repoPath, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || skill == "" {
		return nil, fmt.Errorf("invalid skill ref %q: expected owner/repo@skill", s)
	}

	return &SkillRef{Owner: parts[0], Repo: parts[1], Skill: skill}, nil
}

func (r *SkillRef) RawURL() string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s/SKILL.md",
		r.Owner, r.Repo, r.Skill)
}

func (r *SkillRef) Source() string {
	return fmt.Sprintf("skills.sh/%s/%s@%s", r.Owner, r.Repo, r.Skill)
}

func Import(ref *SkillRef) (*db.Skill, error) {
	// Try multiple path patterns repos use
	base := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main", ref.Owner, ref.Repo)
	paths := []string{
		ref.Skill + "/SKILL.md",          // <skill>/SKILL.md (mattpocock/skills)
		"skills/" + ref.Skill + "/SKILL.md", // skills/<skill>/SKILL.md (obra/superpowers)
		ref.Skill + "/skill.md",          // <skill>/skill.md
		"skills/" + ref.Skill + "/skill.md", // skills/<skill>/skill.md
		".claude/skills/" + ref.Skill + "/SKILL.md", // .claude/skills/<skill>/SKILL.md
		"SKILL.md",                        // root SKILL.md (single-skill repos)
	}

	var body []byte
	var err error
	for _, p := range paths {
		body, err = fetchURL(base + "/" + p)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("could not fetch skill from %s: %w", ref.Source(), err)
	}

	// Parse frontmatter
	s := &db.Skill{}
	rest, err := frontmatter.Parse(bytes.NewReader(body), s)
	if err != nil {
		// No frontmatter — use raw content
		s.Body = strings.TrimSpace(string(body))
	} else {
		s.Body = strings.TrimSpace(string(rest))
	}

	if s.Name == "" {
		s.Name = ref.Skill
	}
	s.Source = ref.Source()

	return s, nil
}

func fetchURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}
