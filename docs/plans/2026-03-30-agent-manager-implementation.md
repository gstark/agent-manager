# Agent Manager (`agm`) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI+TUI tool that manages AI agent skills, rules, and packs across Claude Code and Codex projects.

**Architecture:** Cobra CLI dispatches subcommands. Central DB is flat files in `~/.config/agent-manager/`. Installer reads per-project `.agent-manager.toml`, resolves packs, and generates agent-specific output files. TUI is Bubble Tea with tabbed dashboard.

**Tech Stack:** Go 1.26, Cobra, Bubble Tea v2, Lip Gloss v2, Bubbles v2, BurntSushi/toml, adrg/frontmatter

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/agm/main.go`
- Create: `.gitignore`
- Create: `Makefile`

**Step 1: Initialize Go module**

Run: `cd /Users/gstark/dev/personal/agent-manager && go mod init github.com/gstark/agent-manager`

**Step 2: Create main.go with root command**

Create `cmd/agm/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "agm",
	Short: "Agent Manager — manage AI agent configurations",
	Long:  "A CLI and TUI tool for managing skills, rules, and packs across Claude Code and Codex.",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("agm", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

**Step 3: Create .gitignore**

Create `.gitignore`:

```
/agm
/dist/
```

**Step 4: Create Makefile**

Create `Makefile`:

```makefile
.PHONY: build run test clean

build:
	go build -o agm ./cmd/agm

run: build
	./agm

test:
	go test ./...

clean:
	rm -f agm
```

**Step 5: Install dependencies**

Run: `go get github.com/spf13/cobra@latest`

**Step 6: Verify it builds and runs**

Run: `make build && ./agm version`
Expected: `agm dev`

**Step 7: Initialize git and commit**

Run:
```bash
git init
git add go.mod go.sum cmd/agm/main.go .gitignore Makefile docs/
git commit -m "feat: scaffold agm project with cobra root command"
```

---

### Task 2: Central Config & DB Layer

**Files:**
- Create: `internal/config/paths.go`
- Create: `internal/config/global.go`
- Create: `internal/config/global_test.go`

**Step 1: Write tests for paths**

Create `internal/config/paths_test.go`:

```go
package config

import "testing"

func TestConfigDir(t *testing.T) {
	dir := ConfigDir()
	if dir == "" {
		t.Fatal("ConfigDir returned empty string")
	}
}

func TestSubdirs(t *testing.T) {
	for _, fn := range []func() string{SkillsDir, RulesDir, PacksDir} {
		d := fn()
		if d == "" {
			t.Fatal("subdir returned empty string")
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -v`
Expected: FAIL — package doesn't exist

**Step 3: Implement paths.go**

Create `internal/config/paths.go`:

```go
package config

import (
	"os"
	"path/filepath"
)

func ConfigDir() string {
	if d := os.Getenv("AGM_CONFIG_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "agent-manager")
}

func SkillsDir() string { return filepath.Join(ConfigDir(), "skills") }
func RulesDir() string  { return filepath.Join(ConfigDir(), "rules") }
func PacksDir() string  { return filepath.Join(ConfigDir(), "packs") }
func GlobalConfigPath() string { return filepath.Join(ConfigDir(), "config.toml") }
```

**Step 4: Run tests**

Run: `go test ./internal/config/ -v`
Expected: PASS

**Step 5: Write tests for global config**

Create `internal/config/global_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGlobalConfig_Default(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", tmp)

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Defaults.Editor != "" {
		t.Errorf("expected empty editor, got %q", cfg.Defaults.Editor)
	}
}

func TestLoadGlobalConfig_FromFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", tmp)

	content := `[defaults]
editor = "nvim"
`
	os.WriteFile(filepath.Join(tmp, "config.toml"), []byte(content), 0644)

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Defaults.Editor != "nvim" {
		t.Errorf("expected nvim, got %q", cfg.Defaults.Editor)
	}
}

func TestEnsureDirs(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", tmp)

	if err := EnsureDirs(); err != nil {
		t.Fatal(err)
	}

	for _, sub := range []string{"skills", "rules", "packs"} {
		p := filepath.Join(tmp, sub)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected dir %s to exist", sub)
		}
	}
}
```

**Step 6: Run test to verify it fails**

Run: `go test ./internal/config/ -v -run TestLoadGlobalConfig`
Expected: FAIL

**Step 7: Implement global.go**

Create `internal/config/global.go`:

```go
package config

import (
	"errors"
	"os"

	"github.com/BurntSushi/toml"
)

type GlobalConfig struct {
	GitHub   GitHubConfig   `toml:"github"`
	Defaults DefaultsConfig `toml:"defaults"`
}

type GitHubConfig struct {
	Token string `toml:"token"`
}

type DefaultsConfig struct {
	Editor string `toml:"editor"`
}

func LoadGlobalConfig() (*GlobalConfig, error) {
	cfg := &GlobalConfig{}
	path := GlobalConfigPath()

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func EnsureDirs() error {
	for _, dir := range []string{ConfigDir(), SkillsDir(), RulesDir(), PacksDir()} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
```

**Step 8: Install toml dependency and run tests**

Run: `go get github.com/BurntSushi/toml@latest && go test ./internal/config/ -v`
Expected: PASS

**Step 9: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat: add config paths and global config loader"
```

---

### Task 3: Skill, Rule, and Pack Data Types + CRUD

**Files:**
- Create: `internal/db/types.go`
- Create: `internal/db/skills.go`
- Create: `internal/db/skills_test.go`
- Create: `internal/db/rules.go`
- Create: `internal/db/rules_test.go`
- Create: `internal/db/packs.go`
- Create: `internal/db/packs_test.go`

**Step 1: Write types.go**

Create `internal/db/types.go`:

```go
package db

type Skill struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Source      string `yaml:"source"`
	Body        string `yaml:"-"`
}

type Rule struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Paths       []string `yaml:"paths"`
	Body        string   `yaml:"-"`
}

type Pack struct {
	Name        string   `toml:"name"`
	Description string   `toml:"description"`
	Skills      []string `toml:"skills"`
	Rules       []string `toml:"rules"`
}
```

**Step 2: Write skills_test.go**

Create `internal/db/skills_test.go`:

```go
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
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/db/ -v -run TestSkillCRUD`
Expected: FAIL

**Step 4: Implement skills.go**

Create `internal/db/skills.go`:

```go
package db

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/gstark/agent-manager/internal/config"
	"gopkg.in/yaml.v3"
)

func skillPath(name string) string {
	return filepath.Join(config.SkillsDir(), name+".md")
}

func SaveSkill(s *Skill) error {
	fm, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n\n%s\n", string(fm), strings.TrimSpace(s.Body))
	return os.WriteFile(skillPath(s.Name), []byte(content), 0644)
}

func LoadSkill(name string) (*Skill, error) {
	data, err := os.ReadFile(skillPath(name))
	if err != nil {
		return nil, err
	}
	s := &Skill{}
	rest, err := frontmatter.Parse(bytes.NewReader(data), s)
	if err != nil {
		return nil, err
	}
	s.Body = strings.TrimSpace(string(rest))
	return s, nil
}

func ListSkills() ([]*Skill, error) {
	entries, err := os.ReadDir(config.SkillsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var skills []*Skill
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		s, err := LoadSkill(name)
		if err != nil {
			continue
		}
		skills = append(skills, s)
	}
	return skills, nil
}

func DeleteSkill(name string) error {
	return os.Remove(skillPath(name))
}
```

**Step 5: Install frontmatter dep and run tests**

Run: `go get github.com/adrg/frontmatter@latest && go test ./internal/db/ -v -run TestSkillCRUD`
Expected: PASS

**Step 6: Write rules_test.go**

Create `internal/db/rules_test.go`:

```go
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
```

**Step 7: Implement rules.go (mirrors skills.go pattern)**

Create `internal/db/rules.go`:

```go
package db

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/gstark/agent-manager/internal/config"
	"gopkg.in/yaml.v3"
)

func rulePath(name string) string {
	return filepath.Join(config.RulesDir(), name+".md")
}

func SaveRule(r *Rule) error {
	fm, err := yaml.Marshal(r)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n\n%s\n", string(fm), strings.TrimSpace(r.Body))
	return os.WriteFile(rulePath(r.Name), []byte(content), 0644)
}

func LoadRule(name string) (*Rule, error) {
	data, err := os.ReadFile(rulePath(name))
	if err != nil {
		return nil, err
	}
	r := &Rule{}
	rest, err := frontmatter.Parse(bytes.NewReader(data), r)
	if err != nil {
		return nil, err
	}
	r.Body = strings.TrimSpace(string(rest))
	return r, nil
}

func ListRules() ([]*Rule, error) {
	entries, err := os.ReadDir(config.RulesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var rules []*Rule
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		r, err := LoadRule(name)
		if err != nil {
			continue
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func DeleteRule(name string) error {
	return os.Remove(rulePath(name))
}
```

**Step 8: Run rule tests**

Run: `go test ./internal/db/ -v -run TestRuleCRUD`
Expected: PASS

**Step 9: Write packs_test.go**

Create `internal/db/packs_test.go`:

```go
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
```

**Step 10: Implement packs.go**

Create `internal/db/packs.go`:

```go
package db

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gstark/agent-manager/internal/config"
)

func packPath(name string) string {
	return filepath.Join(config.PacksDir(), name+".toml")
}

func SavePack(p *Pack) error {
	f, err := os.Create(packPath(p.Name))
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(p)
}

func LoadPack(name string) (*Pack, error) {
	p := &Pack{}
	if _, err := toml.DecodeFile(packPath(name), p); err != nil {
		return nil, err
	}
	return p, nil
}

func ListPacks() ([]*Pack, error) {
	entries, err := os.ReadDir(config.PacksDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var packs []*Pack
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".toml")
		p, err := LoadPack(name)
		if err != nil {
			continue
		}
		packs = append(packs, p)
	}
	return packs, nil
}

func DeletePack(name string) error {
	return os.Remove(packPath(name))
}
```

**Step 11: Run all db tests**

Run: `go test ./internal/db/ -v`
Expected: All PASS

**Step 12: Commit**

```bash
git add internal/db/ go.mod go.sum
git commit -m "feat: add skill, rule, and pack CRUD with flat file storage"
```

---

### Task 4: Project Config

**Files:**
- Create: `internal/config/project.go`
- Create: `internal/config/project_test.go`

**Step 1: Write project config tests**

Create `internal/config/project_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfig(t *testing.T) {
	tmp := t.TempDir()
	content := `
skills = ["tdd", "debugging"]
rules = ["concise-output"]
packs = ["ruby"]

[[local_rules]]
name = "use-rspec"
description = "Use RSpec"
paths = ["**/*.rb"]
content = "Always use RSpec"
`
	path := filepath.Join(tmp, ".agent-manager.toml")
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := LoadProjectConfig(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(cfg.Skills))
	}
	if len(cfg.Packs) != 1 {
		t.Errorf("expected 1 pack, got %d", len(cfg.Packs))
	}
	if len(cfg.LocalRules) != 1 {
		t.Errorf("expected 1 local rule, got %d", len(cfg.LocalRules))
	}
	if cfg.LocalRules[0].Name != "use-rspec" {
		t.Errorf("expected use-rspec, got %q", cfg.LocalRules[0].Name)
	}
}

func TestLoadProjectConfig_NotFound(t *testing.T) {
	_, err := LoadProjectConfig(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestSaveProjectConfig(t *testing.T) {
	tmp := t.TempDir()
	cfg := &ProjectConfig{
		Skills: []string{"tdd"},
		Rules:  []string{"concise"},
		Packs:  []string{"ruby"},
	}
	if err := SaveProjectConfig(tmp, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadProjectConfig(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Skills) != 1 || loaded.Skills[0] != "tdd" {
		t.Errorf("round-trip failed: %v", loaded.Skills)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -v -run TestLoadProjectConfig`
Expected: FAIL

**Step 3: Implement project.go**

Create `internal/config/project.go`:

```go
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const ProjectConfigFile = ".agent-manager.toml"

type LocalRule struct {
	Name        string   `toml:"name"`
	Description string   `toml:"description"`
	Paths       []string `toml:"paths"`
	Content     string   `toml:"content"`
}

type ProjectConfig struct {
	Skills     []string    `toml:"skills"`
	Rules      []string    `toml:"rules"`
	Packs      []string    `toml:"packs"`
	LocalRules []LocalRule `toml:"local_rules"`
}

func ProjectConfigPath(dir string) string {
	return filepath.Join(dir, ProjectConfigFile)
}

func LoadProjectConfig(dir string) (*ProjectConfig, error) {
	cfg := &ProjectConfig{}
	if _, err := toml.DecodeFile(ProjectConfigPath(dir), cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func SaveProjectConfig(dir string, cfg *ProjectConfig) error {
	f, err := os.Create(ProjectConfigPath(dir))
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
```

**Step 4: Run tests**

Run: `go test ./internal/config/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/project.go internal/config/project_test.go
git commit -m "feat: add project config loader for .agent-manager.toml"
```

---

### Task 5: Installer — Resolve & Generate

**Files:**
- Create: `internal/installer/installer.go`
- Create: `internal/installer/installer_test.go`
- Create: `internal/installer/claude.go`
- Create: `internal/installer/codex.go`

**Step 1: Write installer test**

Create `internal/installer/installer_test.go`:

```go
package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
)

func setupTestEnv(t *testing.T) (configDir, projectDir string) {
	t.Helper()
	configDir = t.TempDir()
	projectDir = t.TempDir()
	t.Setenv("AGM_CONFIG_DIR", configDir)
	config.EnsureDirs()

	// Create a skill
	db.SaveSkill(&db.Skill{
		Name:        "tdd",
		Description: "TDD workflow",
		Source:      "local",
		Body:        "# TDD\nWrite tests first.",
	})

	// Create a rule
	db.SaveRule(&db.Rule{
		Name:        "concise",
		Description: "Be concise",
		Body:        "Be extremely concise.",
	})

	// Create a rule with paths
	db.SaveRule(&db.Rule{
		Name:        "ruby-style",
		Description: "Ruby style",
		Paths:       []string{"**/*.rb"},
		Body:        "Use snake_case.",
	})

	// Create a pack
	db.SavePack(&db.Pack{
		Name:   "ruby",
		Skills: []string{"tdd"},
		Rules:  []string{"ruby-style"},
	})

	return configDir, projectDir
}

func TestInstall(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	cfg := &config.ProjectConfig{
		Skills: []string{"tdd"},
		Rules:  []string{"concise"},
		Packs:  []string{"ruby"},
	}

	if err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	// AGENTS.md should exist
	if _, err := os.Stat(filepath.Join(projectDir, "AGENTS.md")); err != nil {
		t.Error("AGENTS.md not created")
	}

	// CLAUDE.md should be a symlink to AGENTS.md
	target, err := os.Readlink(filepath.Join(projectDir, "CLAUDE.md"))
	if err != nil {
		t.Error("CLAUDE.md is not a symlink")
	} else if target != "AGENTS.md" {
		t.Errorf("CLAUDE.md points to %q, expected AGENTS.md", target)
	}

	// Claude rules should exist
	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "rules", "ruby-style.md")); err != nil {
		t.Error("claude rule not created")
	}

	// Claude skills should exist
	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "skills", "tdd.md")); err != nil {
		t.Error("claude skill not created")
	}

	// Codex skills should exist
	if _, err := os.Stat(filepath.Join(projectDir, ".agents", "skills", "tdd", "SKILL.md")); err != nil {
		t.Error("codex skill not created")
	}
}

func TestInstallWithLocalRules(t *testing.T) {
	_, projectDir := setupTestEnv(t)

	cfg := &config.ProjectConfig{
		LocalRules: []config.LocalRule{
			{
				Name:        "use-rspec",
				Description: "Use RSpec",
				Paths:       []string{"**/*.rb"},
				Content:     "Always use RSpec.",
			},
		},
	}

	if err := Install(projectDir, cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "rules", "use-rspec.md")); err != nil {
		t.Error("local rule not created in claude rules")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/installer/ -v`
Expected: FAIL

**Step 3: Implement installer.go**

Create `internal/installer/installer.go`:

```go
package installer

import (
	"fmt"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
)

type resolved struct {
	skills     []*db.Skill
	rules      []*db.Rule
	localRules []config.LocalRule
}

func resolve(cfg *config.ProjectConfig) (*resolved, error) {
	r := &resolved{localRules: cfg.LocalRules}
	seen := struct {
		skills map[string]bool
		rules  map[string]bool
	}{make(map[string]bool), make(map[string]bool)}

	addSkill := func(name string) error {
		if seen.skills[name] {
			return nil
		}
		s, err := db.LoadSkill(name)
		if err != nil {
			return fmt.Errorf("skill %q: %w", name, err)
		}
		r.skills = append(r.skills, s)
		seen.skills[name] = true
		return nil
	}

	addRule := func(name string) error {
		if seen.rules[name] {
			return nil
		}
		rule, err := db.LoadRule(name)
		if err != nil {
			return fmt.Errorf("rule %q: %w", name, err)
		}
		r.rules = append(r.rules, rule)
		seen.rules[name] = true
		return nil
	}

	// Expand packs first
	for _, packName := range cfg.Packs {
		p, err := db.LoadPack(packName)
		if err != nil {
			return nil, fmt.Errorf("pack %q: %w", packName, err)
		}
		for _, s := range p.Skills {
			if err := addSkill(s); err != nil {
				return nil, err
			}
		}
		for _, r := range p.Rules {
			if err := addRule(r); err != nil {
				return nil, err
			}
		}
	}

	// Then explicit skills/rules (dedup against packs)
	for _, name := range cfg.Skills {
		if err := addSkill(name); err != nil {
			return nil, err
		}
	}
	for _, name := range cfg.Rules {
		if err := addRule(name); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func Install(projectDir string, cfg *config.ProjectConfig) error {
	r, err := resolve(cfg)
	if err != nil {
		return err
	}

	if err := installClaude(projectDir, r); err != nil {
		return fmt.Errorf("claude: %w", err)
	}
	if err := installCodex(projectDir, r); err != nil {
		return fmt.Errorf("codex: %w", err)
	}
	return nil
}
```

**Step 4: Implement claude.go**

Create `internal/installer/claude.go`:

```go
package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
)

func installClaude(projectDir string, r *resolved) error {
	// Write .claude/rules/*.md
	rulesDir := filepath.Join(projectDir, ".claude", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return err
	}

	for _, rule := range r.rules {
		if err := writeClaudeRule(rulesDir, rule); err != nil {
			return err
		}
	}

	for _, lr := range r.localRules {
		rule := &db.Rule{
			Name:        lr.Name,
			Description: lr.Description,
			Paths:       lr.Paths,
			Body:        lr.Content,
		}
		if err := writeClaudeRule(rulesDir, rule); err != nil {
			return err
		}
	}

	// Write .claude/skills/*.md
	skillsDir := filepath.Join(projectDir, ".claude", "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return err
	}

	for _, skill := range r.skills {
		content := fmt.Sprintf("---\nname: %s\ndescription: %q\n---\n\n%s\n",
			skill.Name, skill.Description, strings.TrimSpace(skill.Body))
		path := filepath.Join(skillsDir, skill.Name+".md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	// Create CLAUDE.md symlink
	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	os.Remove(claudePath) // remove existing file/symlink
	if err := os.Symlink("AGENTS.md", claudePath); err != nil {
		return err
	}

	return nil
}

func writeClaudeRule(dir string, rule *db.Rule) error {
	var b strings.Builder
	b.WriteString("---\n")
	if len(rule.Paths) > 0 {
		b.WriteString("paths:\n")
		for _, p := range rule.Paths {
			fmt.Fprintf(&b, "  - %q\n", p)
		}
	}
	b.WriteString("---\n\n")
	if rule.Description != "" {
		fmt.Fprintf(&b, "# %s\n\n", rule.Description)
	}
	b.WriteString(strings.TrimSpace(rule.Body))
	b.WriteString("\n")

	return os.WriteFile(filepath.Join(dir, rule.Name+".md"), []byte(b.String()), 0644)
}
```

**Step 5: Implement codex.go**

Create `internal/installer/codex.go`:

```go
package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gstark/agent-manager/internal/config"
)

func installCodex(projectDir string, r *resolved) error {
	// Generate AGENTS.md from all rules
	var b strings.Builder
	b.WriteString("# Project Agent Instructions\n\n")
	b.WriteString("<!-- Generated by agm. Do not edit manually. -->\n\n")

	for _, rule := range r.rules {
		if len(rule.Paths) > 0 {
			patterns := strings.Join(rule.Paths, ", ")
			fmt.Fprintf(&b, "## %s (applies to: %s)\n\n", rule.Description, patterns)
		} else if rule.Description != "" {
			fmt.Fprintf(&b, "## %s\n\n", rule.Description)
		}
		b.WriteString(strings.TrimSpace(rule.Body))
		b.WriteString("\n\n")
	}

	for _, lr := range r.localRules {
		if len(lr.Paths) > 0 {
			patterns := strings.Join(lr.Paths, ", ")
			fmt.Fprintf(&b, "## %s (applies to: %s)\n\n", lr.Description, patterns)
		} else if lr.Description != "" {
			fmt.Fprintf(&b, "## %s\n\n", lr.Description)
		}
		b.WriteString(strings.TrimSpace(lr.Content))
		b.WriteString("\n\n")
	}

	agentsPath := filepath.Join(projectDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte(b.String()), 0644); err != nil {
		return err
	}

	// Write .agents/skills/<name>/SKILL.md for Codex
	for _, skill := range r.skills {
		skillDir := filepath.Join(projectDir, ".agents", "skills", skill.Name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}
		content := fmt.Sprintf("---\nname: %s\ndescription: %q\n---\n\n%s\n",
			skill.Name, skill.Description, strings.TrimSpace(skill.Body))
		path := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}
```

**Step 6: Run tests**

Run: `go test ./internal/installer/ -v`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/installer/ go.mod go.sum
git commit -m "feat: add installer that generates Claude and Codex output files"
```

---

### Task 6: GitHub Importer

**Files:**
- Create: `internal/importer/github.go`
- Create: `internal/importer/github_test.go`

**Step 1: Write importer test**

Create `internal/importer/github_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/importer/ -v`
Expected: FAIL

**Step 3: Implement github.go**

Create `internal/importer/github.go`:

```go
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
	// Try SKILL.md first, then skill.md
	body, err := fetchURL(ref.RawURL())
	if err != nil {
		// Try lowercase
		url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s/skill.md",
			ref.Owner, ref.Repo, ref.Skill)
		body, err = fetchURL(url)
		if err != nil {
			return nil, fmt.Errorf("could not fetch skill from %s: %w", ref.Source(), err)
		}
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
```

**Step 4: Run tests**

Run: `go test ./internal/importer/ -v`
Expected: PASS (unit tests only, no network)

**Step 5: Commit**

```bash
git add internal/importer/
git commit -m "feat: add GitHub importer for skills.sh"
```

---

### Task 7: CLI Commands

**Files:**
- Modify: `cmd/agm/main.go`
- Create: `cmd/agm/init.go`
- Create: `cmd/agm/install.go`
- Create: `cmd/agm/add.go`
- Create: `cmd/agm/remove.go`
- Create: `cmd/agm/skills.go`
- Create: `cmd/agm/rules.go`
- Create: `cmd/agm/packs.go`
- Create: `cmd/agm/import.go`

**Step 1: Create init command**

Create `cmd/agm/init.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .agent-manager.toml in current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := os.Getwd()
		path := config.ProjectConfigPath(dir)

		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists", config.ProjectConfigFile)
		}

		cfg := &config.ProjectConfig{}
		if err := config.SaveProjectConfig(dir, cfg); err != nil {
			return err
		}

		// Append to .gitignore
		gitignorePath := filepath.Join(dir, ".gitignore")
		f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		f.WriteString("\n# agent-manager (generated)\n.claude/skills/\n.agents/skills/\n")

		fmt.Println("Created", config.ProjectConfigFile)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
```

**Step 2: Create install command**

Create `cmd/agm/install.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/installer"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install skills and rules per .agent-manager.toml",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := os.Getwd()
		cfg, err := config.LoadProjectConfig(dir)
		if err != nil {
			return fmt.Errorf("no %s found (run 'agm init' first): %w", config.ProjectConfigFile, err)
		}

		if err := installer.Install(dir, cfg); err != nil {
			return err
		}

		fmt.Println("Installed successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
```

**Step 3: Create add command**

Create `cmd/agm/add.go`:

```go
package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [skill|rule|pack] <name>",
	Short: "Add a skill, rule, or pack to the project config",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, name := args[0], args[1]
		dir, _ := os.Getwd()

		cfg, err := config.LoadProjectConfig(dir)
		if err != nil {
			return fmt.Errorf("no %s found: %w", config.ProjectConfigFile, err)
		}

		switch kind {
		case "skill":
			if !slices.Contains(cfg.Skills, name) {
				cfg.Skills = append(cfg.Skills, name)
			}
		case "rule":
			if !slices.Contains(cfg.Rules, name) {
				cfg.Rules = append(cfg.Rules, name)
			}
		case "pack":
			if !slices.Contains(cfg.Packs, name) {
				cfg.Packs = append(cfg.Packs, name)
			}
		default:
			return fmt.Errorf("unknown type %q (use skill, rule, or pack)", kind)
		}

		if err := config.SaveProjectConfig(dir, cfg); err != nil {
			return err
		}
		fmt.Printf("Added %s %q\n", kind, name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
```

**Step 4: Create remove command**

Create `cmd/agm/remove.go`:

```go
package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove [skill|rule|pack] <name>",
	Short: "Remove a skill, rule, or pack from the project config",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, name := args[0], args[1]
		dir, _ := os.Getwd()

		cfg, err := config.LoadProjectConfig(dir)
		if err != nil {
			return fmt.Errorf("no %s found: %w", config.ProjectConfigFile, err)
		}

		switch kind {
		case "skill":
			cfg.Skills = slices.DeleteFunc(cfg.Skills, func(s string) bool { return s == name })
		case "rule":
			cfg.Rules = slices.DeleteFunc(cfg.Rules, func(s string) bool { return s == name })
		case "pack":
			cfg.Packs = slices.DeleteFunc(cfg.Packs, func(s string) bool { return s == name })
		default:
			return fmt.Errorf("unknown type %q (use skill, rule, or pack)", kind)
		}

		if err := config.SaveProjectConfig(dir, cfg); err != nil {
			return err
		}
		fmt.Printf("Removed %s %q\n", kind, name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
```

**Step 5: Create skills CRUD command**

Create `cmd/agm/skills.go`:

```go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
	"github.com/spf13/cobra"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage skills in the central database",
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		skills, err := db.ListSkills()
		if err != nil {
			return err
		}
		if len(skills) == 0 {
			fmt.Println("No skills found. Create one with 'agm skills create <name>'.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDESCRIPTION\tSOURCE")
		for _, s := range skills {
			source := s.Source
			if source == "" {
				source = "local"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, s.Description, source)
		}
		return w.Flush()
	},
}

var skillsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new skill",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		config.EnsureDirs()

		s := &db.Skill{
			Name:   name,
			Source: "local",
			Body:   "# " + name + "\n\nDescribe this skill here.",
		}
		if err := db.SaveSkill(s); err != nil {
			return err
		}
		fmt.Printf("Created skill %q. Edit with 'agm skills edit %s'.\n", name, name)
		return nil
	},
}

var skillsEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit a skill in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := config.SkillsDir() + "/" + name + ".md"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("skill %q not found", name)
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}
		c := exec.Command(editor, path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var skillsDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a skill",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := db.DeleteSkill(name); err != nil {
			return err
		}
		fmt.Printf("Deleted skill %q\n", name)
		return nil
	},
}

func init() {
	skillsCmd.AddCommand(skillsListCmd, skillsCreateCmd, skillsEditCmd, skillsDeleteCmd)
	rootCmd.AddCommand(skillsCmd)
}
```

**Step 6: Create rules CRUD command**

Create `cmd/agm/rules.go`:

```go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
	"github.com/spf13/cobra"
)

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Manage rules in the central database",
}

var rulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		rules, err := db.ListRules()
		if err != nil {
			return err
		}
		if len(rules) == 0 {
			fmt.Println("No rules found. Create one with 'agm rules create <name>'.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDESCRIPTION\tPATHS")
		for _, r := range rules {
			paths := strings.Join(r.Paths, ", ")
			if paths == "" {
				paths = "*"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.Name, r.Description, paths)
		}
		return w.Flush()
	},
}

var rulesCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		config.EnsureDirs()

		r := &db.Rule{
			Name: name,
			Body: "Describe this rule here.",
		}
		if err := db.SaveRule(r); err != nil {
			return err
		}
		fmt.Printf("Created rule %q. Edit with 'agm rules edit %s'.\n", name, name)
		return nil
	},
}

var rulesEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit a rule in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := config.RulesDir() + "/" + name + ".md"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("rule %q not found", name)
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}
		c := exec.Command(editor, path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var rulesDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := db.DeleteRule(name); err != nil {
			return err
		}
		fmt.Printf("Deleted rule %q\n", name)
		return nil
	},
}

func init() {
	rulesCmd.AddCommand(rulesListCmd, rulesCreateCmd, rulesEditCmd, rulesDeleteCmd)
	rootCmd.AddCommand(rulesCmd)
}
```

**Step 7: Create packs CRUD command**

Create `cmd/agm/packs.go`:

```go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
	"github.com/spf13/cobra"
)

var packsCmd = &cobra.Command{
	Use:   "packs",
	Short: "Manage packs in the central database",
}

var packsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all packs",
	RunE: func(cmd *cobra.Command, args []string) error {
		packs, err := db.ListPacks()
		if err != nil {
			return err
		}
		if len(packs) == 0 {
			fmt.Println("No packs found. Create one with 'agm packs create <name>'.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSKILLS\tRULES")
		for _, p := range packs {
			fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name,
				strings.Join(p.Skills, ", "),
				strings.Join(p.Rules, ", "))
		}
		return w.Flush()
	},
}

var packsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new pack",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		config.EnsureDirs()

		p := &db.Pack{
			Name:        name,
			Description: name + " pack",
		}
		if err := db.SavePack(p); err != nil {
			return err
		}
		fmt.Printf("Created pack %q. Edit with 'agm packs edit %s'.\n", name, name)
		return nil
	},
}

var packsEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit a pack in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := config.PacksDir() + "/" + name + ".toml"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("pack %q not found", name)
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}
		c := exec.Command(editor, path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var packsDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a pack",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := db.DeletePack(name); err != nil {
			return err
		}
		fmt.Printf("Deleted pack %q\n", name)
		return nil
	},
}

func init() {
	packsCmd.AddCommand(packsListCmd, packsCreateCmd, packsEditCmd, packsDeleteCmd)
	rootCmd.AddCommand(packsCmd)
}
```

**Step 8: Create import command**

Create `cmd/agm/import.go`:

```go
package main

import (
	"fmt"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
	"github.com/gstark/agent-manager/internal/importer"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <owner/repo@skill>",
	Short: "Import a skill from skills.sh",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		config.EnsureDirs()

		ref, err := importer.ParseSkillRef(args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Fetching %s...\n", ref.Source())
		skill, err := importer.Import(ref)
		if err != nil {
			return err
		}

		if err := db.SaveSkill(skill); err != nil {
			return err
		}

		fmt.Printf("Imported skill %q from %s\n", skill.Name, skill.Source)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
```

**Step 9: Update main.go to call EnsureDirs**

Modify `cmd/agm/main.go` — add a `PersistentPreRunE` to rootCmd:

```go
var rootCmd = &cobra.Command{
	Use:   "agm",
	Short: "Agent Manager — manage AI agent configurations",
	Long:  "A CLI and TUI tool for managing skills, rules, and packs across Claude Code and Codex.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return config.EnsureDirs()
	},
}
```

Add `"github.com/gstark/agent-manager/internal/config"` to imports.

**Step 10: Build and smoke test**

Run:
```bash
make build
./agm --help
./agm skills list
./agm rules list
./agm packs list
```

Expected: Help shows all subcommands. Lists show empty state messages.

**Step 11: Commit**

```bash
git add cmd/agm/
git commit -m "feat: add all CLI commands (init, install, add, remove, skills, rules, packs, import)"
```

---

### Task 8: TUI — Styles & App Shell

**Files:**
- Create: `internal/tui/styles.go`
- Create: `internal/tui/app.go`
- Create: `cmd/agm/tui.go`

**Step 1: Create styles.go**

Create `internal/tui/styles.go`:

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primary   = lipgloss.Color("#7D56F4")
	secondary = lipgloss.Color("#6C6C6C")
	accent    = lipgloss.Color("#04B575")
	subtle    = lipgloss.Color("#383838")
	text      = lipgloss.Color("#FAFAFA")
	dimText   = lipgloss.Color("#888888")

	// Tabs
	activeTab = lipgloss.NewStyle().
			Bold(true).
			Foreground(primary).
			Padding(0, 2)

	inactiveTab = lipgloss.NewStyle().
			Foreground(dimText).
			Padding(0, 2)

	// Title
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(text).
			Background(primary).
			Padding(0, 2).
			MarginBottom(1)

	// Help bar
	helpStyle = lipgloss.NewStyle().
			Foreground(dimText).
			MarginTop(1)

	// Status
	statusStyle = lipgloss.NewStyle().
			Foreground(accent)

	// Borders
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(1, 2)
)
```

**Step 2: Create app.go — main Bubble Tea model**

Create `internal/tui/app.go`:

```go
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/gstark/agent-manager/internal/db"
)

type tab int

const (
	tabSkills tab = iota
	tabRules
	tabPacks
)

var tabNames = []string{"Skills", "Rules", "Packs"}

// list item adapters
type skillItem struct{ s *db.Skill }

func (i skillItem) Title() string       { return i.s.Name }
func (i skillItem) Description() string { src := i.s.Source; if src == "" { src = "local" }; return src + " — " + i.s.Description }
func (i skillItem) FilterValue() string { return i.s.Name }

type ruleItem struct{ r *db.Rule }

func (i ruleItem) Title() string       { return i.r.Name }
func (i ruleItem) Description() string { return i.r.Description }
func (i ruleItem) FilterValue() string { return i.r.Name }

type packItem struct{ p *db.Pack }

func (i packItem) Title() string       { return i.p.Name }
func (i packItem) Description() string { return i.p.Description }
func (i packItem) FilterValue() string { return i.p.Name }

type model struct {
	activeTab  tab
	skillsList list.Model
	rulesList  list.Model
	packsList  list.Model
	width      int
	height     int
	status     string
}

func newModel() model {
	m := model{}
	m.skillsList = newList("Skills")
	m.rulesList = newList("Rules")
	m.packsList = newList("Packs")
	m.loadData()
	return m
}

func newList(title string) list.Model {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = title
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	return l
}

func (m *model) loadData() {
	if skills, err := db.ListSkills(); err == nil {
		items := make([]list.Item, len(skills))
		for i, s := range skills {
			items[i] = skillItem{s}
		}
		m.skillsList.SetItems(items)
	}

	if rules, err := db.ListRules(); err == nil {
		items := make([]list.Item, len(rules))
		for i, r := range rules {
			items[i] = ruleItem{r}
		}
		m.rulesList.SetItems(items)
	}

	if packs, err := db.ListPacks(); err == nil {
		items := make([]list.Item, len(packs))
		for i, p := range packs {
			items[i] = packItem{p}
		}
		m.packsList.SetItems(items)
	}
}

func (m model) activeList() *list.Model {
	switch m.activeTab {
	case tabRules:
		return &m.rulesList
	case tabPacks:
		return &m.packsList
	default:
		return &m.skillsList
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		listHeight := m.height - 8
		m.skillsList.SetSize(m.width-4, listHeight)
		m.rulesList.SetSize(m.width-4, listHeight)
		m.packsList.SetSize(m.width-4, listHeight)
		return m, nil

	case tea.KeyMsg:
		// Don't capture keys when filtering
		if m.activeList().FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "right":
			m.activeTab = (m.activeTab + 1) % 3
			return m, nil
		case "shift+tab", "left":
			m.activeTab = (m.activeTab + 2) % 3
			return m, nil
		case "d":
			return m, m.deleteSelected()
		}
	}

	// Delegate to active list
	var cmd tea.Cmd
	switch m.activeTab {
	case tabSkills:
		m.skillsList, cmd = m.skillsList.Update(msg)
	case tabRules:
		m.rulesList, cmd = m.rulesList.Update(msg)
	case tabPacks:
		m.packsList, cmd = m.packsList.Update(msg)
	}
	return m, cmd
}

func (m *model) deleteSelected() tea.Cmd {
	switch m.activeTab {
	case tabSkills:
		if i, ok := m.skillsList.SelectedItem().(skillItem); ok {
			db.DeleteSkill(i.s.Name)
			m.status = fmt.Sprintf("Deleted skill %q", i.s.Name)
			m.loadData()
		}
	case tabRules:
		if i, ok := m.rulesList.SelectedItem().(ruleItem); ok {
			db.DeleteRule(i.r.Name)
			m.status = fmt.Sprintf("Deleted rule %q", i.r.Name)
			m.loadData()
		}
	case tabPacks:
		if i, ok := m.packsList.SelectedItem().(packItem); ok {
			db.DeletePack(i.p.Name)
			m.status = fmt.Sprintf("Deleted pack %q", i.p.Name)
			m.loadData()
		}
	}
	return nil
}

func (m model) View() string {
	// Tabs
	var tabs []string
	for i, name := range tabNames {
		if tab(i) == m.activeTab {
			tabs = append(tabs, activeTab.Render(name))
		} else {
			tabs = append(tabs, inactiveTab.Render(name))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Active list
	var content string
	switch m.activeTab {
	case tabSkills:
		content = m.skillsList.View()
	case tabRules:
		content = m.rulesList.View()
	case tabPacks:
		content = m.packsList.View()
	}

	// Status
	status := ""
	if m.status != "" {
		status = statusStyle.Render(m.status)
	}

	help := helpStyle.Render("tab: switch  /: filter  d: delete  q: quit")

	return titleStyle.Render("agent-manager") + "\n" +
		tabBar + "\n\n" +
		content + "\n" +
		status + "\n" +
		help
}

func Run() error {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
```

**Step 3: Create tui.go CLI command**

Create `cmd/agm/tui.go`:

```go
package main

import (
	"github.com/gstark/agent-manager/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run()
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
```

**Step 4: Install Charm dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/lipgloss@latest
```

**Step 5: Build and verify TUI launches**

Run: `make build && ./agm tui`
Expected: TUI launches with empty lists, tab switching works, q quits.

**Step 6: Commit**

```bash
git add internal/tui/ cmd/agm/tui.go go.mod go.sum
git commit -m "feat: add TUI dashboard with tabbed skill/rule/pack lists"
```

---

### Task 9: Goreleaser + Homebrew Tap

**Files:**
- Create: `.goreleaser.yaml`

**Step 1: Create goreleaser config**

Create `.goreleaser.yaml`:

```yaml
version: 2

builds:
  - main: ./cmd/agm
    binary: agm
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - formats: [tar.gz]

brews:
  - repository:
      owner: gstark
      name: homebrew-tap
    directory: Formula
    homepage: https://github.com/gstark/agent-manager
    description: Manage AI agent configurations across Claude Code and Codex
    install: |
      bin.install "agm"

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
```

**Step 2: Verify goreleaser config**

Run: `goreleaser check` (if installed) or just verify YAML is valid.

**Step 3: Commit**

```bash
git add .goreleaser.yaml
git commit -m "feat: add goreleaser config with homebrew tap"
```

---

### Task 10: End-to-End Smoke Test

**Step 1: Build**

Run: `make build`

**Step 2: Create a test skill and rule**

Run:
```bash
./agm skills create test-skill
./agm rules create test-rule
./agm packs create test-pack
./agm skills list
./agm rules list
./agm packs list
```

Expected: Each list shows the created item.

**Step 3: Test import from skills.sh**

Run: `./agm import mattpocock/skills@tdd`
Expected: "Imported skill "tdd" from skills.sh/mattpocock/skills@tdd"

**Step 4: Test project init and install**

Run:
```bash
cd /tmp && mkdir agm-test && cd agm-test
/Users/gstark/dev/personal/agent-manager/agm init
```

Edit `.agent-manager.toml` to add the skill:
```bash
echo 'skills = ["tdd"]
rules = ["test-rule"]' > .agent-manager.toml
```

Run: `/Users/gstark/dev/personal/agent-manager/agm install`

Verify:
```bash
ls -la AGENTS.md CLAUDE.md
ls .claude/rules/
ls .claude/skills/
ls .agents/skills/
cat AGENTS.md
```

Expected: All files generated correctly. CLAUDE.md is symlink to AGENTS.md.

**Step 5: Clean up and final commit**

Run:
```bash
cd /Users/gstark/dev/personal/agent-manager
./agm skills delete test-skill
./agm rules delete test-rule
./agm packs delete test-pack
rm -rf /tmp/agm-test
```

---

Plan complete and saved to `docs/plans/2026-03-30-agent-manager-implementation.md`. Two execution options:

**1. Subagent-Driven (this session)** — I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** — Open new session with executing-plans, batch execution with checkpoints

Which approach?