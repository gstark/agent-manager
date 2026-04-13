package importer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
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

// contentsEntry represents a file in the GitHub Contents API response.
type contentsEntry struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "file" or "dir"
	DownloadURL string `json:"download_url"`
}

// httpClient is the HTTP client used for all requests. Tests can replace it.
var httpClient = http.DefaultClient

// contentsAPIBase can be overridden in tests.
var contentsAPIBase = "https://api.github.com"

// rawContentBase can be overridden in tests.
var rawContentBase = "https://raw.githubusercontent.com"

func Import(ref *SkillRef) (*db.Skill, error) {
	// Try multiple path patterns repos use
	base := fmt.Sprintf("%s/%s/%s/main", rawContentBase, ref.Owner, ref.Repo)
	patterns := []struct {
		skillMDPath string
		dirPath     string // directory containing the skill files
	}{
		{ref.Skill + "/SKILL.md", ref.Skill},
		{"skills/" + ref.Skill + "/SKILL.md", "skills/" + ref.Skill},
		{ref.Skill + "/skill.md", ref.Skill},
		{"skills/" + ref.Skill + "/skill.md", "skills/" + ref.Skill},
		{".claude/skills/" + ref.Skill + "/SKILL.md", ".claude/skills/" + ref.Skill},
		{"SKILL.md", ""},
	}

	var body []byte
	var matchedDir string
	var err error
	for _, p := range patterns {
		body, err = fetchURL(base + "/" + p.skillMDPath)
		if err == nil {
			matchedDir = p.dirPath
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("could not fetch skill from %s: %w", ref.Source(), err)
	}

	// Parse frontmatter
	s := &db.Skill{}
	rest, parseErr := frontmatter.Parse(bytes.NewReader(body), s)
	if parseErr != nil {
		// No frontmatter — use raw content
		s.Body = strings.TrimSpace(string(body))
	} else {
		s.Body = strings.TrimSpace(string(rest))
	}

	if s.Name == "" {
		s.Name = ref.Skill
	}
	s.Source = ref.Source()

	// Fetch extra files from the skill directory
	if matchedDir != "" {
		files, err := fetchExtraFiles(ref, matchedDir)
		if err == nil && len(files) > 0 {
			s.Files = files
		}
	}

	s.ContentHash = db.ComputeContentHash(s.Body, s.Files)

	return s, nil
}

// fetchExtraFiles lists the skill directory via the GitHub Contents API and
// fetches all non-SKILL.md files.
func fetchExtraFiles(ref *SkillRef, dirPath string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	if err := fetchExtraFilesRecursive(ref, dirPath, "", files); err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}
	return files, nil
}

func fetchExtraFilesRecursive(ref *SkillRef, repoDirPath string, relDir string, files map[string][]byte) error {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=main",
		contentsAPIBase, ref.Owner, ref.Repo, repoDirPath)

	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	var entries []contentsEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return err
	}

	for _, e := range entries {
		switch e.Type {
		case "dir":
			if err := fetchExtraFilesRecursive(
				ref,
				path.Join(repoDirPath, e.Name),
				path.Join(relDir, e.Name),
				files,
			); err != nil {
				return err
			}
		case "file":
			lower := strings.ToLower(e.Name)
			if lower == "skill.md" {
				continue
			}
			if e.DownloadURL == "" {
				continue
			}
			data, err := fetchURL(e.DownloadURL)
			if err != nil {
				continue // skip files we can't fetch
			}
			name := e.Name
			if relDir != "" {
				name = path.Join(relDir, e.Name)
			}
			files[name] = data
		}
	}

	return nil
}

func fetchURL(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}
