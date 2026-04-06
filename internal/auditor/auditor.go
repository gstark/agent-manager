package auditor

import (
	"github.com/gstark/agent-manager/internal/db"
)

type Status string

const (
	StatusUpToDate Status = "up-to-date"
	StatusChanged  Status = "changed"
	StatusUnknown  Status = "unknown"
	StatusError    Status = "error"
)

type AuditResult struct {
	Name   string
	Source string
	Status Status
	Error  string
}

type FetchFunc func(source string) (body string, files map[string][]byte, err error)

func Audit(skills []*db.Skill, fetch FetchFunc) []AuditResult {
	var results []AuditResult
	for _, s := range skills {
		if s.Source == "" || s.Source == "local" {
			continue
		}

		if s.ContentHash == "" {
			results = append(results, AuditResult{Name: s.Name, Source: s.Source, Status: StatusUnknown})
			continue
		}

		body, files, err := fetch(s.Source)
		if err != nil {
			results = append(results, AuditResult{Name: s.Name, Source: s.Source, Status: StatusError, Error: err.Error()})
			continue
		}
		remoteHash := db.ComputeContentHash(body, files)
		if remoteHash == s.ContentHash {
			results = append(results, AuditResult{Name: s.Name, Source: s.Source, Status: StatusUpToDate})
		} else {
			results = append(results, AuditResult{Name: s.Name, Source: s.Source, Status: StatusChanged})
		}
	}
	return results
}
