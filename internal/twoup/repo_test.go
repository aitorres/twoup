package twoup

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureGitRepo_DirectGitDir(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	if err := ensureGitRepo(root); err != nil {
		t.Fatalf("ensureGitRepo() error = %v", err)
	}
}

func TestEnsureGitRepo_AncestorGitDir(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	if err := ensureGitRepo(nested); err != nil {
		t.Fatalf("ensureGitRepo() error = %v", err)
	}
}

func TestEnsureGitRepo_NotRepo(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	err := ensureGitRepo(nested)
	if !errors.Is(err, errNotGitRepo) {
		t.Fatalf("ensureGitRepo() error = %v, want errNotGitRepo", err)
	}
}
