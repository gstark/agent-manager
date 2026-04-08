# Shared Agent Install Execution Plan

Date: 2026-04-08

## Goal

Refactor `agm install` so Claude Code and Codex share one canonical install for portable artifacts, while tool-specific adapters generate the files each runtime actually requires.

This plan follows the research in [docs/plans/2026-04-08-shared-agent-install-architecture.md](/Users/gstark/dev/personal/agent-manager/docs/plans/2026-04-08-shared-agent-install-architecture.md).

## Success Criteria

- `agm` no longer models Claude prompt rules and Codex execution rules as the same artifact.
- Shared skills are installed once and exposed to both tools.
- `CLAUDE.md` is generated as a wrapper that imports `@AGENTS.md`.
- Codex gets real `.codex/rules/` output for execution policies.
- Installer logic is structured so a third tool can be added via a new adapter instead of branching core resolution logic again.
- Existing install behavior keeps working during migration unless explicitly replaced.

## Non-Goals

- Full support for every Claude/Codex surface in one pass.
- Hooks, subagents, plugins, and settings fragments in v1 of this refactor.
- Migrating all historical user config automatically without an explicit compatibility layer.

## Current Problems To Fix

- [internal/db/types.go](/Users/gstark/dev/personal/agent-manager/internal/db/types.go) has one `Rule` type that is acting as both:
  - a prompt-instruction fragment for Claude and `AGENTS.md`
  - a supposed Codex rule, which is incorrect
- [internal/installer/installer.go](/Users/gstark/dev/personal/agent-manager/internal/installer/installer.go) resolves one mixed artifact set and pushes it into two hardcoded installers.
- [internal/installer/claude.go](/Users/gstark/dev/personal/agent-manager/internal/installer/claude.go) creates `CLAUDE.md` as a symlink to `AGENTS.md`, which is weaker than Anthropic's documented import pattern.
- [internal/installer/codex.go](/Users/gstark/dev/personal/agent-manager/internal/installer/codex.go) only generates `AGENTS.md` and `.agents/skills/`, so Codex execution-policy setup is missing.
- Shared skills are duplicated into both `.claude/skills/` and `.agents/skills/`.

## Phase 1: Split The Domain Model

### Outcome

Introduce separate artifact concepts for instructions vs execution policy, while preserving current behavior through compatibility shims.

### Tasks

1. Add new types for neutral artifacts.
   Files:
   - update [internal/db/types.go](/Users/gstark/dev/personal/agent-manager/internal/db/types.go)
   - add supporting CRUD files if needed under [internal/db](/Users/gstark/dev/personal/agent-manager/internal/db)

2. Keep the existing prompt-oriented rule storage working, but rename the concept in code toward `Instruction` or `InstructionFragment`.
   Files:
   - update [internal/db/rules.go](/Users/gstark/dev/personal/agent-manager/internal/db/rules.go)
   - update [internal/config/project.go](/Users/gstark/dev/personal/agent-manager/internal/config/project.go)
   - update command surfaces only if needed for compatibility

3. Introduce a new execution-policy model for Codex-native rules.
   Files:
   - add new DB type and storage, likely under [internal/db](/Users/gstark/dev/personal/agent-manager/internal/db)
   - add project config support for policy references in [internal/config/project.go](/Users/gstark/dev/personal/agent-manager/internal/config/project.go)

4. Decide compatibility strategy for existing `.agent-manager.toml`.
   Recommendation:
   - keep `rules = [...]` as prompt instructions for now
   - add a new field such as `policies = [...]` for Codex execution rules
   - defer any CLI rename until after installer refactor

### Verification

- unit tests for new types and config decoding
- backward-compatibility test proving existing `rules = [...]` projects still install

## Phase 2: Create A Canonical Shared Skill Tree

### Outcome

Install each skill once, then project it into Claude and Codex skill directories.

### Tasks

1. Define a canonical generated path for shared skills.
   Recommendation:
   - `.agent-manager/generated/skills/<name>/`

2. Move skill materialization into shared logic.
   Files:
   - add shared installer helpers under [internal/installer](/Users/gstark/dev/personal/agent-manager/internal/installer)
   - reduce skill-writing duplication in [internal/installer/claude.go](/Users/gstark/dev/personal/agent-manager/internal/installer/claude.go) and [internal/installer/codex.go](/Users/gstark/dev/personal/agent-manager/internal/installer/codex.go)

3. Create projection directories for each tool.
   Files:
   - `.claude/skills/<name>` -> symlink to canonical skill dir
   - `.agents/skills/<name>` -> symlink to canonical skill dir

4. Add a fallback strategy if symlink creation fails.
   Recommendation:
   - copy as a fallback
   - surface the degraded mode in install output

### Verification

- installer test proving both tool paths resolve to the same canonical skill contents
- test for extra skill files like helper scripts
- test for idempotent re-install

## Phase 3: Separate Shared Instructions From Tool-Specific Instruction Surfaces

### Outcome

Use one shared instruction body where possible, while preserving Claude’s richer scoped-rule support.

### Tasks

1. Keep `AGENTS.md` as the shared root instruction artifact.
   Files:
   - refactor generation in [internal/installer/codex.go](/Users/gstark/dev/personal/agent-manager/internal/installer/codex.go) or a shared instruction renderer

2. Replace the `CLAUDE.md` symlink with a generated wrapper file.
   Wrapper shape:
   ```md
   # Project Agent Instructions

   @AGENTS.md
   ```
   Files:
   - update [internal/installer/claude.go](/Users/gstark/dev/personal/agent-manager/internal/installer/claude.go)
   - update [internal/installer/installer_test.go](/Users/gstark/dev/personal/agent-manager/internal/installer/installer_test.go)

3. Preserve `.claude/rules/*.md` as Claude-only path-scoped instruction files.
   Files:
   - update [internal/installer/claude.go](/Users/gstark/dev/personal/agent-manager/internal/installer/claude.go)

4. Make the relationship explicit in docs.
   Files:
   - update [README.md](/Users/gstark/dev/personal/agent-manager/README.md)
   - update any comments or CLI help that imply one shared “rule” model

### Verification

- installer test confirming `CLAUDE.md` is a regular file, not a symlink
- golden-content test for `AGENTS.md`
- test that Claude scoped rule files still include frontmatter paths

## Phase 4: Add Real Codex Execution Policy Support

### Outcome

Codex gets actual `.codex/rules/` output sourced from policy artifacts, not from prompt instructions.

### Tasks

1. Define the file format and internal representation for execution policies.
   Files:
   - new types and helpers under [internal/db](/Users/gstark/dev/personal/agent-manager/internal/db)

2. Implement policy installation into `.codex/rules/`.
   Files:
   - extend [internal/installer/codex.go](/Users/gstark/dev/personal/agent-manager/internal/installer/codex.go) or split into a dedicated policy installer module

3. Add CLI support to manage policies if needed in this phase.
   Files:
   - likely new commands under [cmd/agm](/Users/gstark/dev/personal/agent-manager/cmd/agm)
   Recommendation:
   - if CLI scope is too large, land installer support first and follow with CRUD commands

4. Add project config references for policies.
   Files:
   - update [internal/config/project.go](/Users/gstark/dev/personal/agent-manager/internal/config/project.go)

### Verification

- tests for writing `.codex/rules/*`
- compatibility test proving prompt instructions do not leak into Codex policy output
- one fixture-style example with a `prefix_rule(...)`

## Phase 5: Refactor To Adapter-Driven Installation

### Outcome

Core resolution produces neutral artifacts, and each tool consumes only what it understands.

### Tasks

1. Replace the current `resolved` struct with an artifact-oriented set.
   Files:
   - update [internal/installer/installer.go](/Users/gstark/dev/personal/agent-manager/internal/installer/installer.go)

2. Introduce a `ToolAdapter` interface and convert Claude/Codex installers to implement it.
   Files:
   - update [internal/installer/claude.go](/Users/gstark/dev/personal/agent-manager/internal/installer/claude.go)
   - update [internal/installer/codex.go](/Users/gstark/dev/personal/agent-manager/internal/installer/codex.go)

3. Move shared rendering helpers out of tool-specific files.
   Files:
   - add helper modules under [internal/installer](/Users/gstark/dev/personal/agent-manager/internal/installer)

4. Make third-tool onboarding explicit in code structure.
   Recommendation:
   - one adapter registration point
   - one artifact contract
   - zero tool checks in generic resolution

### Verification

- unit tests for adapter dispatch
- installer integration tests covering both Claude and Codex in one run
- regression tests for packs plus local project instructions

## Phase 6: Documentation And Migration Cleanup

### Outcome

The product language matches the implementation and users understand the new abstraction model.

### Tasks

1. Update README terminology.
   Files:
   - update [README.md](/Users/gstark/dev/personal/agent-manager/README.md)
   Changes:
   - distinguish instructions from execution policies
   - describe shared skill installation clearly
   - document generated paths accurately

2. Update design docs if they remain authoritative.
   Files:
   - update [docs/plans/2026-03-30-agent-manager-design.md](/Users/gstark/dev/personal/agent-manager/docs/plans/2026-03-30-agent-manager-design.md) or supersede it explicitly

3. Add migration notes for users.
   Recommendation:
   - explain `CLAUDE.md` wrapper change
   - explain new policy config field
   - explain shared generated skill layout

### Verification

- README examples match actual generated files
- no stale docs claim Codex prompt instructions are “rules”

## Suggested Delivery Order

1. Phase 1
2. Phase 3
3. Phase 2
4. Phase 4
5. Phase 5
6. Phase 6

Reasoning:

- split the model first so later code stops reinforcing the wrong abstraction
- fix `CLAUDE.md` and instruction semantics before adding shared skill symlinks
- land Codex policy support before the larger adapter refactor if we want earlier user value

## Testing Strategy

- extend [internal/installer/installer_test.go](/Users/gstark/dev/personal/agent-manager/internal/installer/installer_test.go) with migration-focused scenarios
- add unit tests for new config decoding in [internal/config](/Users/gstark/dev/personal/agent-manager/internal/config)
- add DB tests for any new policy storage in [internal/db](/Users/gstark/dev/personal/agent-manager/internal/db)
- add at least one end-to-end installer fixture covering:
  - shared instructions
  - Claude scoped rules
  - shared skills
  - Codex execution policy

## Risks

- Claude skill symlink behavior should be verified before making it the only mode.
- Changing config vocabulary too early may create unnecessary churn.
- Moving generated assets under `.agent-manager/generated/` affects `.gitignore`, docs, and teammate expectations.
- Supporting both old and new config models for too long can complicate the installer.

## Recommended First Implementation Slice

If we want the safest first change set, do this in one PR:

1. Add execution-policy artifacts and project config support.
2. Generate `CLAUDE.md` as an `@AGENTS.md` wrapper instead of a symlink.
3. Add `.codex/rules/` output.
4. Update README terminology.

That slice fixes the biggest semantic gap without forcing the larger shared-skill and adapter refactors into the same change.
