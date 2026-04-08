# Shared Agent Install Architecture

Date: 2026-04-08

## Goal

`agm install` should stop treating Claude Code and Codex as if they expose the same setup surface.

We want:

- one canonical source for reusable agent content where that is actually possible
- thin per-tool adapters where the runtimes differ
- an internal model that can absorb a third tool without reworking the installer again

## Current State In This Repo

Today `agm` resolves project skills and rules once, then installs them twice:

- Claude output is generated in [internal/installer/claude.go](/Users/gstark/dev/personal/agent-manager/internal/installer/claude.go)
- Codex output is generated in [internal/installer/codex.go](/Users/gstark/dev/personal/agent-manager/internal/installer/codex.go)
- install orchestration lives in [internal/installer/installer.go](/Users/gstark/dev/personal/agent-manager/internal/installer/installer.go)

Observed behavior:

- Claude:
  - writes `.claude/rules/<name>.md`
  - writes `.claude/skills/<name>/SKILL.md`
  - symlinks `CLAUDE.md -> AGENTS.md`
- Codex:
  - writes a single root `AGENTS.md`
  - writes `.agents/skills/<name>/SKILL.md`

Important mismatch:

- `agm` currently models "rules" as prompt instructions for both tools.
- In Codex docs, "Rules" means sandbox escalation policy (`prefix_rule(...)`) stored under `.codex/rules/` or `~/.codex/rules/`.
- So current Codex support is incomplete even if `AGENTS.md` generation works.

## What The Tools Actually Support

### Claude Code

Instructions:

- Claude reads `CLAUDE.md`, not `AGENTS.md`.
- Anthropic explicitly recommends a `CLAUDE.md` file that imports `@AGENTS.md` when you want both tools to share instructions.
- Claude also supports modular `.claude/rules/**/*.md` files, including path-scoped rules via YAML `paths`.
- `.claude/rules/` supports symlinks.

Skills:

- Claude skills live in:
  - `~/.claude/skills/<name>/SKILL.md`
  - `.claude/skills/<name>/SKILL.md`
  - plugin `skills/`
- Claude automatically discovers nested `.claude/skills/` directories.
- Claude skills use the Agent Skills standard and extend it with Claude-specific frontmatter.

Other setup surfaces worth modeling later:

- `.claude/settings.json`
- `.claude/hooks/` and settings-backed hooks
- `.claude/agents/`

### Codex

Instructions:

- Codex reads `AGENTS.md` and `AGENTS.override.md`.
- Codex walks from repo root to current working directory and concatenates one instructions file per directory.
- Codex can also check extra fallback filenames via `project_doc_fallback_filenames` in `.codex/config.toml`.

Skills:

- Codex skills live in:
  - `.agents/skills/` from the current directory up to repo root
  - `$HOME/.agents/skills`
  - `/etc/codex/skills`
- Codex explicitly supports symlinked skill folders.
- Codex skills also use the Agent Skills standard.

Rules:

- Codex "Rules" are execution-policy files under `.codex/rules/` or `~/.codex/rules/`.
- These are not prompt-instruction files; they define `prefix_rule(...)` behavior for sandbox escalation.

Other setup surfaces worth modeling later:

- `.codex/config.toml`
- `.codex/hooks.json`
- subagents
- plugins

## Key Constraint

There is no honest "single installed file" solution for everything.

Why:

- skills are portable because both tools use the Agent Skills directory format
- root instructions are portable enough because Claude can import `AGENTS.md`
- scoped rules are not portable because Claude has first-class path-scoped rule files while Codex uses directory-scoped `AGENTS.md`
- Codex execution rules are a different concept entirely from Claude prompt rules

So the correct design target is:

- one canonical installed copy for portable artifacts
- one canonical authored copy plus thin adapters for non-portable artifacts

## Approaches

### 1. Claude-First Install Layout

Canonical install:

- `CLAUDE.md`
- `.claude/rules/**`
- `.claude/skills/**`

Adapters:

- generate `AGENTS.md` from Claude instructions
- symlink `.agents/skills/* -> .claude/skills/*`

Pros:

- aligns with Claude's richer instruction model
- keeps existing `.claude/rules` behavior intact
- Codex can consume skill symlinks because symlinked skill folders are documented

Cons:

- Codex becomes a projection of Claude concepts
- Codex-specific rules still need a separate `.codex/rules/` pipeline
- `AGENTS.md` generation becomes lossy for path-scoped rules

Verdict:

- workable, but Claude-centric rather than extensible

### 2. Codex-First Install Layout

Canonical install:

- `AGENTS.md`
- `.agents/skills/**`

Adapters:

- create `CLAUDE.md` that imports `@AGENTS.md`
- symlink `.claude/skills/* -> .agents/skills/*`
- optionally drop `.claude/rules/`

Pros:

- simplest shared root instructions story
- matches Anthropic's documented `@AGENTS.md` bridge
- gives us true single-copy skill installs if Claude accepts symlinked skill directories in practice

Cons:

- loses Claude's best instruction primitive unless we keep `.claude/rules/`
- path-scoped rules still need a Claude-only layer
- assumes symlinked `.claude/skills` is acceptable even though Anthropic docs do not explicitly document it

Verdict:

- attractive for shared instructions and skills, but incomplete for rule semantics

### 3. Neutral AGM Runtime Layer

Canonical install:

- `.agent-manager/generated/instructions/`
- `.agent-manager/generated/skills/`
- `.agent-manager/generated/policies/`

Adapters:

- `CLAUDE.md` imports generated instruction index
- `.claude/rules/**` are rendered or symlinked views over generated instruction fragments when possible
- `AGENTS.md` is rendered from the same instruction graph
- `.claude/skills/*` and `.agents/skills/*` point at the same generated skill directories
- `.codex/rules/**` is rendered from policy data, not from prompt-rule content

Pros:

- clearly separates portable content from tool-specific materialization
- scales to a third tool by adding another adapter instead of changing authored content
- avoids forcing Claude concepts onto Codex or vice versa
- gives us a place to model additional surfaces like hooks, agents, and settings

Cons:

- more implementation work now
- wrappers still exist, so this is "one canonical copy plus adapters" rather than literally one file for every consumer

Verdict:

- best long-term architecture

## Recommendation

Adopt approach 3, with a pragmatic phase order:

### Phase 1: Separate artifact types in the data model

Replace the current overloaded "rule" concept with explicit artifact classes:

- `instruction_fragment`
- `skill`
- `execution_policy`
- later: `hook`, `subagent`, `plugin_asset`, `settings_fragment`

This is the most important change. Without it, Codex will continue to be modeled incorrectly.

### Phase 2: Make shared skills truly single-copy

Install one canonical skill directory tree, then expose it to both tools:

- canonical location: `.agent-manager/generated/skills/<name>/`
- adapter views:
  - `.claude/skills/<name>` -> symlink
  - `.agents/skills/<name>` -> symlink

If we want lower risk for Claude, flip the canonical location to `.claude/skills/` first and symlink Codex into it. Codex symlink support is documented; Claude symlink support should be verified before we depend on it.

### Phase 3: Split prompt instructions from Codex execution rules

Prompt instructions:

- canonical authored fragments in AGM
- rendered into:
  - `AGENTS.md`
  - `CLAUDE.md` wrapper that imports `@AGENTS.md`
  - optional `.claude/rules/**` for path-scoped Claude behavior

Execution policy:

- new AGM policy artifacts render into:
  - `.codex/rules/*.rules`
  - later, tool-native policy surfaces for other agents

This keeps "be concise" style guidance separate from "allow `gh pr view` outside sandbox" policy.

### Phase 4: Introduce adapter-driven installer interfaces

The installer should stop hardcoding `installClaude` and `installCodex` around a single resolved blob.

Instead define something like:

```go
type ArtifactSet struct {
	Instructions     []InstructionFragment
	Skills           []*db.Skill
	ExecutionPolicy  []ExecutionPolicy
	Hooks            []Hook
	Subagents        []Subagent
}

type ToolAdapter interface {
	Name() string
	Install(projectDir string, artifacts ArtifactSet) ([]ItemResult, error)
}
```

This lets a third tool declare:

- which artifact classes it consumes
- which output paths it needs
- whether it can use shared symlinks or needs rendered files

## Recommended Install Shape

Short term:

- keep root `AGENTS.md` as the shared instruction body
- replace the `CLAUDE.md -> AGENTS.md` symlink with a real `CLAUDE.md` wrapper that contains `@AGENTS.md`
- keep `.claude/rules/**` for Claude-specific scoped instructions
- add a new `.codex/rules/**` pipeline for actual Codex rules
- convert skills to one canonical tree plus adapter symlinks

Long term:

- move generated shared assets under `.agent-manager/generated/`
- make `.claude/*`, `.agents/*`, and `.codex/*` thin projections of that shared layer

## Why This Extends Cleanly To A Third Tool

A third tool will almost certainly mix:

- portable artifacts it shares with the Agent Skills standard
- project instruction files with its own discovery rules
- its own native policy/config/hook surface

If AGM stores "Claude rule" and "Codex rule" as the same thing, every new tool forces another semantic overload.

If AGM stores neutral artifact classes and lets adapters render them, a third tool is just:

- one new adapter
- optional new artifact classes if the tool exposes something truly novel

That is the right abstraction boundary.

## Immediate Follow-Up Work

1. Update product language in README and code comments so "rules" no longer implies the same thing for Claude and Codex.
2. Add a new Codex policy model and installer output under `.codex/rules/`.
3. Convert `CLAUDE.md` generation from symlink to `@AGENTS.md` wrapper.
4. Prototype shared skill installation with symlinks and verify Claude behavior locally.
5. Refactor installer resolution around artifact classes plus tool adapters.

## Sources

- OpenAI Codex `AGENTS.md`: https://developers.openai.com/codex/guides/agents-md
- OpenAI Codex skills: https://developers.openai.com/codex/skills
- OpenAI Codex rules: https://developers.openai.com/codex/rules
- OpenAI Codex config basics: https://developers.openai.com/codex/config-basic
- OpenAI Codex hooks: https://developers.openai.com/codex/hooks
- Anthropic Claude Code memory / `CLAUDE.md` / `.claude/rules`: https://code.claude.com/docs/en/memory
- Anthropic Claude Code skills: https://code.claude.com/docs/en/skills
- Anthropic Claude Code settings: https://docs.anthropic.com/en/docs/claude-code/settings
- Anthropic Claude Code subagents: https://code.claude.com/docs/en/sub-agents
