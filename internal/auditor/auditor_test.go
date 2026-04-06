package auditor

import (
	"fmt"
	"testing"

	"github.com/gstark/agent-manager/internal/db"
)

func TestAuditUpToDate(t *testing.T) {
	body := "# My Skill"
	hash := db.ComputeContentHash(body, nil)

	skills := []*db.Skill{
		{
			Name:        "tdd",
			Source:      "skills.sh/owner/repo@tdd",
			ContentHash: hash,
			Body:        body,
		},
	}

	fetch := func(source string) (string, map[string][]byte, error) {
		return body, nil, nil
	}

	results := Audit(skills, fetch)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusUpToDate {
		t.Errorf("expected %q, got %q", StatusUpToDate, results[0].Status)
	}
	if results[0].Name != "tdd" {
		t.Errorf("expected name %q, got %q", "tdd", results[0].Name)
	}
	if results[0].Source != "skills.sh/owner/repo@tdd" {
		t.Errorf("expected source %q, got %q", "skills.sh/owner/repo@tdd", results[0].Source)
	}
}

func TestAuditSkipsLocalSkills(t *testing.T) {
	body := "# Skill"
	hash := db.ComputeContentHash(body, nil)

	skills := []*db.Skill{
		{Name: "local1", Source: "local", ContentHash: hash},
		{Name: "local2", Source: "", ContentHash: hash},
		{Name: "remote", Source: "skills.sh/owner/repo@remote", ContentHash: hash},
	}

	fetch := func(source string) (string, map[string][]byte, error) {
		return body, nil, nil
	}

	results := Audit(skills, fetch)
	if len(results) != 1 {
		t.Fatalf("expected 1 result (only remote), got %d", len(results))
	}
	if results[0].Name != "remote" {
		t.Errorf("expected %q, got %q", "remote", results[0].Name)
	}
}

func TestAuditUnknownHash(t *testing.T) {
	skills := []*db.Skill{
		{Name: "legacy", Source: "skills.sh/owner/repo@legacy", ContentHash: ""},
	}

	fetchCalled := false
	fetch := func(source string) (string, map[string][]byte, error) {
		fetchCalled = true
		return "", nil, nil
	}

	results := Audit(skills, fetch)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusUnknown {
		t.Errorf("expected %q, got %q", StatusUnknown, results[0].Status)
	}
	if fetchCalled {
		t.Error("fetch should not be called for skills with no content hash")
	}
}

func TestAuditFetchError(t *testing.T) {
	body := "# Skill"
	hash := db.ComputeContentHash(body, nil)

	skills := []*db.Skill{
		{Name: "broken", Source: "skills.sh/owner/repo@broken", ContentHash: hash},
		{Name: "ok", Source: "skills.sh/owner/repo@ok", ContentHash: hash},
	}

	fetch := func(source string) (string, map[string][]byte, error) {
		if source == "skills.sh/owner/repo@broken" {
			return "", nil, fmt.Errorf("HTTP 404")
		}
		return body, nil, nil
	}

	results := Audit(skills, fetch)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Status != StatusError {
		t.Errorf("broken: expected %q, got %q", StatusError, results[0].Status)
	}
	if results[0].Error != "HTTP 404" {
		t.Errorf("broken: expected error %q, got %q", "HTTP 404", results[0].Error)
	}
	if results[1].Status != StatusUpToDate {
		t.Errorf("ok: expected %q, got %q", StatusUpToDate, results[1].Status)
	}
}

func TestAuditChanged(t *testing.T) {
	localBody := "# My Skill"
	hash := db.ComputeContentHash(localBody, nil)

	skills := []*db.Skill{
		{
			Name:        "tdd",
			Source:      "skills.sh/owner/repo@tdd",
			ContentHash: hash,
		},
	}

	fetch := func(source string) (string, map[string][]byte, error) {
		return "# Updated Skill", nil, nil
	}

	results := Audit(skills, fetch)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusChanged {
		t.Errorf("expected %q, got %q", StatusChanged, results[0].Status)
	}
}
