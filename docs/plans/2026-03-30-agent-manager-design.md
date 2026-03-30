# Agent Manager (`agm`) Design

CLI + TUI tool for managing AI agent configurations across Claude Code and Codex.

## Goals

- Central database of skills and rules in `~/.config/agent-manager/`
- Per-project config file (`.agent-manager.toml`) lists which skills, rules, and packs apply
- `agm install` generates the correct output files for both Claude Code and Codex
- Import skills from skills.sh (GitHub repos) into the central DB
- Beautiful TUI built with Charm (Bubble Tea + Lip Gloss)
- Distributable via Homebrew tap

## CLI Commands

```
agm init                            # create .agent-manager.toml in current project
agm install                         # install skills/rules per .agent-manager.toml
agm add rule|skill|pack <name>      # add to project config
agm remove rule|skill|pack <name>   # remove from project config
agm rules list|create|edit|delete   # CRUD on central rule DB
agm skills list|create|edit|delete  # CRUD on central skill DB
agm packs list|create|edit|delete   # CRUD on central pack DB
agm import <owner/repo@skill>       # import from skills.sh into central DB
agm tui                             # launch interactive TUI
```

## Central Database

Location: `~/.config/agent-manager/`

```
~/.config/agent-manager/
├── skills/
│   └── <name>.md          # frontmatter (name, description, source) + markdown body
├── rules/
│   └── <name>.md          # frontmatter (name, description, paths[]) + markdown body
├── packs/
│   └── <name>.toml        # name, description, skills[], rules[]
└── config.toml            # global settings
```

### Skill file format

```markdown
---
name: tdd
description: Test-driven development workflow
source: skills.sh/mattpocock/skills@tdd
---

# TDD Workflow
...
```

### Rule file format

```markdown
---
name: ruby-conventions
description: Ruby naming, style, and linting rules
paths:
  - "**/*.rb"
---

## Ruby Conventions
...
```

### Pack file format

```toml
name = "ruby"
description = "Full Ruby/Rails development setup"

skills = ["tdd", "debugging"]
rules = ["ruby-conventions", "ruby-testing", "bundle-exec"]
```

### Global config

```toml
[github]
token = ""

[defaults]
editor = "vim"
```

## Per-Project Config

File: `.agent-manager.toml`

```toml
skills = ["tdd", "debugging"]
rules = ["concise-output", "no-test-code-in-prod"]
packs = ["ruby"]

[[local_rules]]
name = "use-rspec"
description = "Always use RSpec for tests"
paths = ["**/*.rb"]
content = "Use RSpec for all Ruby tests, never minitest"
```

## Install Output

`agm install` resolves packs into flat lists (deduped), then generates:

| Output | Purpose |
|---|---|
| `AGENTS.md` | All rules assembled. Rules with `paths:` get a prose note. |
| `CLAUDE.md` | Symlink to `AGENTS.md` |
| `.claude/rules/<name>.md` | Each rule with `paths:` frontmatter for Claude Code |
| `.claude/skills/<name>.md` | Each skill for Claude Code |
| `.agents/skills/<name>/SKILL.md` | Each skill for Codex |

`agm init` appends to `.gitignore`:

```
# agent-manager (generated)
.claude/skills/
.agents/skills/
```

Rules, `AGENTS.md`, and `CLAUDE.md` are committed so teammates without the tool still get them.

## Skills.sh Import

`agm import <owner/repo@skill>`:

1. Parse identifier into GitHub owner, repo, skill path
2. Fetch `SKILL.md` from `https://raw.githubusercontent.com/<owner>/<repo>/main/<skill>/SKILL.md`
3. Parse frontmatter + body
4. Save to `~/.config/agent-manager/skills/<name>.md` with `source:` field
5. Confirm to user

No search API in v1. User browses skills.sh, copies the identifier.

## TUI

Built with Bubble Tea + Lip Gloss + Bubbles.

### Dashboard

```
┌─ agent-manager ──────────────────────────────┐
│                                               │
│  Skills (12)    Rules (8)    Packs (3)        │
│                                               │
│  ┌ Skills ──────────────────────────────────┐ │
│  │ > tdd              mattpocock/skills     │ │
│  │   debugging         local                │ │
│  │   brainstorming     local                │ │
│  │                                          │ │
│  │  n New  e Edit  d Delete  i Import       │ │
│  └──────────────────────────────────────────┘ │
│                                               │
│  ↹ Tab  / Search  ? Help  q Quit             │
└───────────────────────────────────────────────┘
```

### Project view (inside a project)

```
┌─ Project: my-rails-app ──────────────────────┐
│                                               │
│  Active Skills (5)   Active Rules (6)         │
│  Available Skills    Available Rules          │
│                                               │
│  [+] Add  [-] Remove  [I] Install            │
└───────────────────────────────────────────────┘
```

Editing markdown bodies shells out to `$EDITOR`.

## Architecture

```
cmd/agm/main.go                    # CLI entrypoint (cobra)
internal/
  config/
    global.go                      # ~/.config/agent-manager/ read/write
    project.go                     # .agent-manager.toml read/write
  db/
    skills.go                      # CRUD for skills
    rules.go                       # CRUD for rules
    packs.go                       # CRUD for packs
  importer/
    github.go                      # fetch SKILL.md from GitHub repos
  installer/
    installer.go                   # resolve packs, generate all output files
    claude.go                      # Claude Code output (.claude/rules/, .claude/skills/)
    codex.go                       # Codex output (AGENTS.md, .agents/skills/)
  tui/
    app.go                         # Bubble Tea main model
    dashboard.go                   # home screen
    list.go                        # reusable list component
    detail.go                      # view/edit detail
    project.go                     # project view
    styles.go                      # Lip Gloss styles
```

## Dependencies

- `github.com/spf13/cobra` — CLI
- `github.com/charmbracelet/bubbletea` — TUI
- `github.com/charmbracelet/lipgloss` — styling
- `github.com/charmbracelet/bubbles` — list, textinput, viewport
- `github.com/BurntSushi/toml` — TOML
- `github.com/adrg/frontmatter` — markdown frontmatter

## Distribution

- GitHub releases with goreleaser
- Homebrew tap (`homebrew-tap` repo) for `brew install gstark/tap/agm`

## Decisions

- Pure Go, no Node/external dependencies
- skills.sh import via raw GitHub fetch (no `npx skills`)
- No search in v1 — user provides `owner/repo@skill` directly
- Codex file-type rules: prose in root `AGENTS.md` (no subdirectory generation)
- No lockfile in v1 — central DB is source of truth
- `$EDITOR` for markdown editing, no in-TUI editor
