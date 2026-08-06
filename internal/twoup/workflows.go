package twoup

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func findWorkflowFiles(root string) ([]string, error) {
	workflowRoot := filepath.Join(root, ".github", "workflows")
	if _, err := os.Stat(workflowRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []string
	err := filepath.WalkDir(workflowRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		lower := strings.ToLower(d.Name())
		if strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}
