package db

import (
	"crypto/sha256"
	"fmt"
	"sort"
)

// ComputeContentHash returns a hex-encoded SHA-256 hash of a skill's content.
// It concatenates the SKILL.md body and all extra files in sorted-by-filename
// order to produce a deterministic hash.
func ComputeContentHash(body string, files map[string][]byte) string {
	h := sha256.New()

	// Collect all filenames (SKILL.md + extras) and sort
	names := make([]string, 0, len(files)+1)
	names = append(names, "SKILL.md")
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if name == "SKILL.md" {
			h.Write([]byte(body))
		} else {
			h.Write(files[name])
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
