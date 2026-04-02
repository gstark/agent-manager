# agm

A package manager for AI agent skills and rules.

## Why

Claude Code and Codex store skills and rules as files inside each repo or in a global config directory. This works for small setups, but breaks down quickly:

- **Global configs get cluttered** — your Go rules don't belong in your React project, but global `CLAUDE.md` and skills apply everywhere.
- **Per-repo configs don't travel** — recreating the same skills across 10 repos is tedious and they drift over time.
- **Sharing is manual** — copying markdown files between repos or from someone's GitHub is fragile.

`agm` solves this by acting as a package manager. You maintain a central library of skills, rules, and packs on your machine. Each project gets a small `.agent-manager.toml` declaring what it needs. Run `agm install` and the right files are written to `.claude/` and `.agents/` — nothing more, nothing less.

This means you can:

- **Curate once, use everywhere** — write a skill or rule once, add it to any project with `agm add`
- **Import and customize** — pull skills from GitHub, then edit them to fit your workflow
- **Bundle with packs** — group all your React skills and rules into a `react` pack, your Go ones into a `go` pack, and apply the right set per-project
- **Keep repos clean** — generated agent configs stay out of your way; your source of truth lives in `~/.config/agent-manager/`

## Features

- **Skills** — reusable markdown instructions installed into `.claude/skills/` and `.agents/skills/`
- **Rules** — project guidelines with optional file-path globs, written to `.claude/rules/` and `AGENTS.md`
- **Packs** — named bundles of skills and rules for one-command setup
- **Import** — pull skills from GitHub repos (`agm import owner/repo@skill`)
- **Dual output** — generates config for both Claude Code and Codex simultaneously
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

This writes `.claude/skills/tdd/SKILL.md`, `.claude/rules/concise-output.md`, `.agents/skills/tdd/SKILL.md`, and `AGENTS.md` — ready for both Claude Code and Codex.

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

`agm` stores skills, rules, and packs in `~/.config/agent-manager/`:

```
~/.config/agent-manager/
├── skills/       # Markdown files with YAML frontmatter
├── rules/        # Markdown files with YAML frontmatter
├── packs/        # TOML bundles of skills + rules
└── config.toml   # Global settings
```

Each project gets an `.agent-manager.toml` that references items from this central database:

```toml
skills = ["tdd", "debugging"]
rules = ["concise-output"]
packs = ["ruby"]

[[local_rules]]
name = "use-rspec"
description = "Always use RSpec"
paths = ["**/*.rb"]
content = "Use RSpec for all Ruby tests"
```

Running `agm install` resolves packs, deduplicates, and writes platform-specific files.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `$AGM_CONFIG_DIR` | `~/.config/agent-manager` | Override central database location |
| `$VISUAL` / `$EDITOR` | `vim` | Editor for create/edit commands |

## License

MIT
