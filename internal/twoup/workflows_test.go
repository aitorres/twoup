package twoup

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFindWorkflowFiles_MissingDir(t *testing.T) {
	root := t.TempDir()
	files, err := findWorkflowFiles(root)
	if err != nil {
		t.Fatalf("findWorkflowFiles() error = %v", err)
	}
	if files != nil {
		t.Fatalf("findWorkflowFiles() = %v, want nil", files)
	}
}

func TestFindWorkflowFiles_FindsAndSorts(t *testing.T) {
	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(filepath.Join(workflowDir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}

	writeFile := func(rel string) {
		t.Helper()
		full := filepath.Join(workflowDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for file: %v", err)
		}
		if err := os.WriteFile(full, []byte("name: test\n"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
	}

	writeFile("z-last.yml")
	writeFile("a-first.yaml")
	writeFile("nested/middle.YML")
	writeFile("ignored.txt")

	files, err := findWorkflowFiles(root)
	if err != nil {
		t.Fatalf("findWorkflowFiles() error = %v", err)
	}

	want := []string{
		filepath.Join(workflowDir, "a-first.yaml"),
		filepath.Join(workflowDir, "nested", "middle.YML"),
		filepath.Join(workflowDir, "z-last.yml"),
	}

	if !reflect.DeepEqual(files, want) {
		t.Fatalf("findWorkflowFiles() = %v, want %v", files, want)
	}
}
