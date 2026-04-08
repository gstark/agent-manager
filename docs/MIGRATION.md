# Migration Guide

## Shared Install Model (April 2026)

This release refactors `agm install` to separate prompt instructions from execution policies and to install skills once instead of duplicating them.

### CLAUDE.md is now a wrapper file

Previously, `CLAUDE.md` was a symlink to `AGENTS.md`. It is now a generated file that imports shared instructions:

```md
# Project Agent Instructions

@AGENTS.md
```

This follows Anthropic's recommended pattern for projects that support both Claude Code and Codex. No action needed — `agm install` replaces the old symlink automatically.

### New `policies` config field

Codex execution-policy rules (sandbox escalation, network access) are now a separate artifact type from prompt instructions.

**Before:** no way to manage Codex execution policies through `agm`.

**After:** add a `policies` field to `.agent-manager.toml`:

```toml
skills = ["tdd"]
rules = ["concise-output"]
policies = ["no-network"]
```

Policies are installed to `.codex/rules/<name>.md`. They do not appear in `AGENTS.md` or `.claude/rules/`.

Packs can also include policies:

```toml
name = "secure"
skills = ["tdd"]
rules = ["concise-output"]
policies = ["no-network"]
```

Existing configs without `policies` continue to work unchanged.

### Shared skill installation via `.agm/skills/`

Skills are now installed once to a canonical location (`.agm/skills/<name>/`) and projected into each tool directory via symlinks:

```
.agm/skills/tdd/SKILL.md          # canonical copy (single source of truth)
.claude/skills/tdd -> ../../.agm/skills/tdd   # symlink
.agents/skills/tdd -> ../../.agm/skills/tdd   # symlink
```

If symlink creation fails (e.g. on a filesystem that doesn't support them), `agm` falls back to copying.

The `.agm/` directory is gitignored. The symlinks in `.claude/skills/` and `.agents/skills/` are also gitignored. Only `agm install` manages these paths.

### .gitignore changes

`agm init` now adds `.agm/` to `.gitignore` alongside the existing skill directory entries:

```
# agent-manager (generated)
.agm/
.claude/skills/
.agents/skills/
```

Existing projects will get this update on the next `agm init` or you can add `.agm/` manually.
