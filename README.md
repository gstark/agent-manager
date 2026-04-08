# agm

A package manager for AI agent skills, rules, and policies.

## Why

Claude Code and Codex store skills, rules, and policies as files inside each repo or in a global config directory. This works for small setups, but breaks down quickly:

- **Global configs get cluttered** — your Go rules don't belong in your React project, but global `CLAUDE.md` and skills apply everywhere.
- **Per-repo configs don't travel** — recreating the same skills across 10 repos is tedious and they drift over time.
- **Sharing is manual** — copying markdown files between repos or from someone's GitHub is fragile.

`agm` solves this by acting as a package manager. You maintain a central library of skills, rules, and packs on your machine. Each project gets a small `.agent-manager.toml` declaring what it needs. Run `agm install` and the right files are written to `.claude/`, `.agents/`, and `.codex/` — nothing more, nothing less.

This means you can:

- **Curate once, use everywhere** — write a skill or rule once, add it to any project with `agm add`
- **Import and customize** — pull skills from GitHub, then edit them to fit your workflow
- **Bundle with packs** — group skills, rules, and policies into a `react` pack, a `go` pack, and apply the right set per-project
- **Keep repos clean** — generated agent configs stay out of your way; your source of truth lives in `~/.config/agent-manager/`

## Concepts

`agm` manages three distinct artifact types:

- **Skills** — reusable markdown instructions installed once into `.agm/skills/` and symlinked into `.claude/skills/` and `.agents/skills/`
- **Rules** — prompt instructions with optional file-path globs, written to `.claude/rules/` and assembled into `AGENTS.md`
- **Policies** — Codex execution-policy rules, written to `.codex/rules/` (these are _not_ prompt instructions — they control sandbox escalation)
- **Packs** — named bundles of skills, rules, and policies for one-command setup

## Features

- **Dual output** — generates config for both Claude Code and Codex simultaneously
- **Shared skill installation** — skills are installed once to `.agm/skills/` and projected into each tool via symlinks
- **Import** — pull skills from GitHub repos (`agm import owner/repo@skill`)
- **Interactive TUI** — browse, toggle, and manage items per-project
- **JSON output** — pipe-friendly `--json` flag on all list commands

## Installation

### Homebrew

```sh
brew install gstark/tap/agm
```

### Go

```sh
go install github.com/gstark/agent-manager/cmd/agm@latest
```

### From source

```sh
git clone https://github.com/gstark/agent-manager.git
cd agent-manager
make build
```

## Quick start

```sh
# Initialize a project
cd your-project
agm init

# Create a skill and a rule
agm skills create tdd
agm rules create concise-output

# Add them to the project
agm add skill tdd
agm add rule concise-output

# Generate agent config files
agm install
```

This produces:

| Output | Purpose |
|---|---|
| `AGENTS.md` | Shared prompt instructions assembled from all rules |
| `CLAUDE.md` | Wrapper that imports `@AGENTS.md` for Claude Code |
| `.claude/rules/<name>.md` | Per-rule files with `paths:` frontmatter for Claude Code |
| `.agm/skills/<name>/SKILL.md` | Canonical skill (single copy) |
| `.claude/skills/<name>` | Symlink → `.agm/skills/<name>` |
| `.agents/skills/<name>` | Symlink → `.agm/skills/<name>` |
| `.codex/rules/<name>.md` | Execution policies for Codex |

## Commands

| Command | Description |
|---|---|
| `agm init` | Create `.agent-manager.toml` in the current directory |
| `agm install` | Generate agent config files from project config |
| `agm skills list\|create\|edit\|cat\|delete` | Manage skills |
| `agm rules list\|create\|edit\|delete` | Manage rules |
| `agm packs list\|create\|edit\|delete` | Manage packs |
| `agm add [skill\|rule\|pack] <name>` | Add item to project |
| `agm remove [skill\|rule\|pack] <name>` | Remove item from project |
| `agm import <owner/repo@skill>` | Import skill from GitHub |
| `agm tui` | Launch interactive dashboard |

## How it works

`agm` stores skills, rules, policies, and packs in `~/.config/agent-manager/`:

```
~/.config/agent-manager/
├── skills/       # Markdown files with YAML frontmatter
├── rules/        # Prompt instructions with YAML frontmatter
├── policies/     # Codex execution-policy rules with YAML frontmatter
├── packs/        # TOML bundles of skills + rules + policies
└── config.toml   # Global settings
```

Each project gets an `.agent-manager.toml` that references items from this central database:

```toml
skills = ["tdd", "debugging"]
rules = ["concise-output"]
policies = ["no-network"]
packs = ["ruby"]

[[local_rules]]
name = "use-rspec"
description = "Always use RSpec"
paths = ["**/*.rb"]
content = "Use RSpec for all Ruby tests"
```

Running `agm install` resolves packs, deduplicates, and writes platform-specific files:

- **Rules** become prompt instructions: `.claude/rules/*.md` for Claude and sections in `AGENTS.md` for Codex
- **Policies** become execution rules: `.codex/rules/*.md` for Codex (not written to Claude output)
- **Skills** are installed once to `.agm/skills/` then symlinked into `.claude/skills/` and `.agents/skills/`
- **`CLAUDE.md`** is generated as a wrapper that imports `@AGENTS.md`, following Anthropic's recommended pattern

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `$AGM_CONFIG_DIR` | `~/.config/agent-manager` | Override central database location |
| `$VISUAL` / `$EDITOR` | `vim` | Editor for create/edit commands |

## License

MIT
