package twoup

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRun_NoWorkflowDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	stats, err := run(config{root: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if stats.UpdatedFiles != 0 || stats.UpdatedLines != 0 {
		t.Fatalf("run() stats = %+v, want zero updates", stats)
	}
}

func TestRun_WorkflowWithoutUses(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	workflowFile := filepath.Join(workflowDir, "ci.yml")
	content := "name: ci\n\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo hello\n"
	if err := os.WriteFile(workflowFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	stats, err := run(config{root: root})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if stats.UpdatedFiles != 0 || stats.UpdatedLines != 0 {
		t.Fatalf("run() stats = %+v, want zero updates", stats)
	}
}

func TestRun_WorkflowReadError(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	workflowDir := filepath.Join(root, ".github", "workflows")
	brokenTarget := filepath.Join(workflowDir, "target")
	if err := os.MkdirAll(brokenTarget, 0o755); err != nil {
		t.Fatalf("mkdir invalid workflow target: %v", err)
	}
	if err := os.Symlink(brokenTarget, filepath.Join(workflowDir, "broken.yml")); err != nil {
		t.Fatalf("symlink invalid workflow: %v", err)
	}

	if _, err := run(config{root: root}); err == nil {
		t.Fatal("expected workflow read error")
	}
}

func TestRunWithClient_UpdatesWorkflow(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	workflowFile := filepath.Join(workflowDir, "ci.yml")
	content := "jobs:\n  test:\n    steps:\n      - uses: actions/checkout@v4\n"
	if err := os.WriteFile(workflowFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	const digest = "3333333333333333333333333333333333333333"
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/actions/checkout/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v5.0.0"}`))
	})
	mux.HandleFunc("/repos/actions/checkout/git/ref/tags/v5.0.0", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":{"sha":"` + digest + `","type":"commit"}}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newGitHubClient()
	client.baseURL = srv.URL
	stats, err := runWithClient(config{root: root, verbose: true}, client)
	if err != nil {
		t.Fatalf("runWithClient() error = %v", err)
	}
	if stats.UpdatedFiles != 1 || stats.UpdatedLines != 1 {
		t.Fatalf("runWithClient() stats = %+v, want one file and line", stats)
	}

	updated, err := os.ReadFile(workflowFile)
	if err != nil {
		t.Fatalf("read updated workflow: %v", err)
	}
	want := "jobs:\n  test:\n    steps:\n      - uses: actions/checkout@" + digest + " # v5.0.0\n"
	if string(updated) != want {
		t.Fatalf("updated workflow = %q, want %q", string(updated), want)
	}
}
