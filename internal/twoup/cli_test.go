package twoup

import (
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func runMainForTest(t *testing.T, args ...string) int {
	t.Helper()

	originalArgs := os.Args
	originalCommandLine := flag.CommandLine
	defer func() {
		os.Args = originalArgs
		flag.CommandLine = originalCommandLine
	}()

	os.Args = append([]string{"twoup"}, args...)
	flag.CommandLine = flag.NewFlagSet("twoup", flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	return Main()
}

func TestMain_NoWorkflowDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	if code := runMainForTest(t, "-repo", root); code != 0 {
		t.Fatalf("Main() = %d, want 0", code)
	}
}

func TestMain_NotGitRepo(t *testing.T) {
	if code := runMainForTest(t, "-repo", t.TempDir()); code != 2 {
		t.Fatalf("Main() = %d, want 2", code)
	}
}

func TestExitErr(t *testing.T) {
	if code := exitErr(errNotGitRepo); code != 2 {
		t.Fatalf("exitErr(errNotGitRepo) = %d, want 2", code)
	}
	if code := exitErr(errors.New("boom")); code != 1 {
		t.Fatalf("exitErr(generic) = %d, want 1", code)
	}
}
