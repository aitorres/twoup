package twoup

import (
	"errors"
	"os"
	"path/filepath"
)

var errNotGitRepo = errors.New("directory is not inside a git repository")

func ensureGitRepo(root string) error {
	dir := root
	for {
		candidate := filepath.Join(dir, ".git")
		if _, err := os.Stat(candidate); err == nil {
			return nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return errNotGitRepo
		}
		dir = parent
	}
}
