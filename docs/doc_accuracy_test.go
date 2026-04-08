package docs_test

import (
	"os"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	return string(data)
}

func TestREADME_DistinguishesInstructionsFromPolicies(t *testing.T) {
	readme := readFile(t, "../README.md")

	if !strings.Contains(readme, "policies") && !strings.Contains(readme, "Policies") {
		t.Error("README should mention policies as distinct from rules/instructions")
	}
	if !strings.Contains(readme, ".codex/rules/") {
		t.Error("README should describe .codex/rules/ output for policies")
	}
}

func TestREADME_DescribesSharedSkillLayout(t *testing.T) {
	readme := readFile(t, "../README.md")

	if !strings.Contains(readme, ".agm/skills/") {
		t.Error("README should describe canonical .agm/skills/ directory")
	}
}

func TestREADME_DescribesCLAUDEMDWrapper(t *testing.T) {
	readme := readFile(t, "../README.md")

	// CLAUDE.md should not be described as a symlink (skill symlinks are fine)
	if strings.Contains(readme, "Symlink to `AGENTS.md`") || strings.Contains(readme, "CLAUDE.md` | Symlink") {
		t.Error("README should not describe CLAUDE.md as a symlink — it is a wrapper file")
	}
	if !strings.Contains(readme, "@AGENTS.md") {
		t.Error("README should describe CLAUDE.md as importing @AGENTS.md")
	}
}

func TestREADME_ProjectConfigShowsPolicies(t *testing.T) {
	readme := readFile(t, "../README.md")

	if !strings.Contains(readme, `policies = [`) {
		t.Error("README project config example should include policies field")
	}
}

func TestREADME_PackFormatShowsPolicies(t *testing.T) {
	readme := readFile(t, "../README.md")

	// The pack format section or example should show policies field
	if !strings.Contains(readme, "policies") {
		t.Error("README should show policies in pack description or examples")
	}
}

func TestREADME_NoStaleCodexRulesAsPromptInstructions(t *testing.T) {
	readme := readFile(t, "../README.md")

	// "rules" in the context of Codex should refer to execution policies, not prompt instructions
	// The README should not claim Codex prompt instructions are "rules"
	if strings.Contains(readme, "written to `.claude/rules/` and `AGENTS.md`") {
		t.Error("README should not conflate Claude rules and AGENTS.md as the same concept")
	}
}

func TestDesignDoc_InstallOutputAccurate(t *testing.T) {
	design := readFile(t, "../docs/plans/2026-03-30-agent-manager-design.md")

	if strings.Contains(design, "Symlink to `AGENTS.md`") {
		t.Error("design doc should not describe CLAUDE.md as a symlink")
	}
	if !strings.Contains(design, ".agm/skills/") {
		t.Error("design doc should describe canonical .agm/skills/ directory")
	}
	if !strings.Contains(design, ".codex/rules/") {
		t.Error("design doc should describe .codex/rules/ for execution policies")
	}
}

func TestMigrationNotes_Exist(t *testing.T) {
	notes := readFile(t, "../docs/MIGRATION.md")

	if !strings.Contains(notes, "CLAUDE.md") {
		t.Error("migration notes should cover CLAUDE.md wrapper change")
	}
	if !strings.Contains(notes, "policies") {
		t.Error("migration notes should cover the policy config field")
	}
	if !strings.Contains(notes, ".agm/skills/") {
		t.Error("migration notes should cover the generated skill path")
	}
}
