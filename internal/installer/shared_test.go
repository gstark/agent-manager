package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileIfChanged_NewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	changed, err := writeFileIfChanged(path, []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true for new file")
	}

	got, _ := os.ReadFile(path)
	if string(got) != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestWriteFileIfChanged_SameContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello"), 0644)

	changed, err := writeFileIfChanged(path, []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("expected changed=false for same content")
	}
}

func TestWriteFileIfChanged_DifferentContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello"), 0644)

	changed, err := writeFileIfChanged(path, []byte("world"))
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true for different content")
	}

	got, _ := os.ReadFile(path)
	if string(got) != "world" {
		t.Errorf("got %q, want %q", got, "world")
	}
}
