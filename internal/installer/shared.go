package installer

import "os"

// writeFileIfChanged writes content to path and returns true if the file was
// created or its content changed.
func writeFileIfChanged(path string, content []byte) (changed bool, err error) {
	existing, err := os.ReadFile(path)
	if err == nil && string(existing) == string(content) {
		return false, nil
	}
	return true, os.WriteFile(path, content, 0644)
}
