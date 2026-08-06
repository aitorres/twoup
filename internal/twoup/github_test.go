package twoup

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveLatest_AnnotatedTag(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/actions/checkout/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v7.0.1"}`))
	})
	mux.HandleFunc("/repos/actions/checkout/git/ref/tags/v7.0.1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":{"sha":"tagsha123","type":"tag"}}`))
	})
	mux.HandleFunc("/repos/actions/checkout/git/tags/tagsha123", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":{"sha":"3d3c42e5aac5ba805825da76410c181273ba90b1","type":"commit"}}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newGitHubClient()
	client.baseURL = srv.URL

	out, err := client.resolveLatest(context.Background(), actionRef{Owner: "actions", Repo: "checkout", Ref: "v4"})
	if err != nil {
		t.Fatalf("resolveLatest() error = %v", err)
	}
	if out.LatestTag != "v7.0.1" {
		t.Fatalf("LatestTag = %q want %q", out.LatestTag, "v7.0.1")
	}
	if out.Digest != "3d3c42e5aac5ba805825da76410c181273ba90b1" {
		t.Fatalf("Digest = %q", out.Digest)
	}
}

func TestResolveLatest_DirectCommitTag(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/a/b/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3"}`))
	})
	mux.HandleFunc("/repos/a/b/git/ref/tags/v1.2.3", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":{"sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","type":"commit"}}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newGitHubClient()
	client.baseURL = srv.URL

	out, err := client.resolveLatest(context.Background(), actionRef{Owner: "a", Repo: "b", Ref: "v1"})
	if err != nil {
		t.Fatalf("resolveLatest() error = %v", err)
	}
	if out.Digest != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("Digest = %q", out.Digest)
	}
}

func TestResolveLatest_FallsBackToTags(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	})
	mux.HandleFunc("/repos/o/r/tags", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"name":"v9.9.9"}]`))
	})
	mux.HandleFunc("/repos/o/r/git/ref/tags/v9.9.9", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":{"sha":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","type":"commit"}}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newGitHubClient()
	client.baseURL = srv.URL

	out, err := client.resolveLatest(context.Background(), actionRef{Owner: "o", Repo: "r", Ref: "v1"})
	if err != nil {
		t.Fatalf("resolveLatest() error = %v", err)
	}
	if out.LatestTag != "v9.9.9" {
		t.Fatalf("LatestTag = %q", out.LatestTag)
	}
}

func TestLatestReleaseTag_EmptyTagName(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":""}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newGitHubClient()
	client.baseURL = srv.URL

	_, err := client.latestReleaseTag(context.Background(), "o", "r")
	if err == nil {
		t.Fatal("expected error for empty tag name")
	}
}

func TestLatestReleaseTag_PropagatesNon404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newGitHubClient()
	client.baseURL = srv.URL

	_, err := client.latestReleaseTag(context.Background(), "o", "r")
	var apiErr *githubAPIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("expected githubAPIError 403, got %v", err)
	}
}

func TestLatestTag_NoTags(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/tags", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newGitHubClient()
	client.baseURL = srv.URL

	_, err := client.latestTag(context.Background(), "o", "r")
	if err == nil {
		t.Fatal("expected error for empty tag list")
	}
}

func TestTagToCommitSHA_UnsupportedObjectType(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/git/ref/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":{"sha":"abc","type":"blob"}}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newGitHubClient()
	client.baseURL = srv.URL

	_, err := client.tagToCommitSHA(context.Background(), "o", "r", "v1.0.0")
	if err == nil {
		t.Fatal("expected unsupported type error")
	}
}

func TestTagToCommitSHA_TagDoesNotResolveToCommit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/git/ref/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":{"sha":"tagsha","type":"tag"}}`))
	})
	mux.HandleFunc("/repos/o/r/git/tags/tagsha", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":{"sha":"","type":"tree"}}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newGitHubClient()
	client.baseURL = srv.URL

	_, err := client.tagToCommitSHA(context.Background(), "o", "r", "v1.0.0")
	if err == nil {
		t.Fatal("expected non-commit resolution error")
	}
}

func TestGetJSON_Non2xxReturnsTypedError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/boom", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(`{"message":"teapot"}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newGitHubClient()
	client.baseURL = srv.URL

	var out map[string]any
	err := client.getJSON(context.Background(), srv.URL+"/boom", &out)
	var apiErr *githubAPIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusTeapot {
		t.Fatalf("expected githubAPIError 418, got %v", err)
	}
}

func TestResolveAll_ResolvesDistinctActions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/a/one/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v1.0.0"}`))
	})
	mux.HandleFunc("/repos/a/one/git/ref/tags/v1.0.0", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":{"sha":"1111111111111111111111111111111111111111","type":"commit"}}`))
	})
	mux.HandleFunc("/repos/b/two/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v2.0.0"}`))
	})
	mux.HandleFunc("/repos/b/two/git/ref/tags/v2.0.0", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"object":{"sha":"2222222222222222222222222222222222222222","type":"commit"}}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := newGitHubClient()
	client.baseURL = srv.URL

	refs := []actionRef{
		{Owner: "a", Repo: "one", Ref: "v1"},
		{Owner: "b", Repo: "two", Ref: "v2"},
	}
	out, err := resolveAll(context.Background(), client, refs)
	if err != nil {
		t.Fatalf("resolveAll() error = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("resolveAll() len = %d, want 2", len(out))
	}
	if out["a/one"].Digest != "1111111111111111111111111111111111111111" {
		t.Fatalf("a/one digest mismatch: %q", out["a/one"].Digest)
	}
	if out["b/two"].Digest != "2222222222222222222222222222222222222222" {
		t.Fatalf("b/two digest mismatch: %q", out["b/two"].Digest)
	}
}

func TestNewGitHubClient_UsesToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", " test-token ")

	client := newGitHubClient()
	if client.token != "test-token" {
		t.Fatalf("client token = %q, want %q", client.token, "test-token")
	}
	if client.httpClient == nil {
		t.Fatal("client httpClient is nil")
	}
}

func TestGitHubAPIError_Error(t *testing.T) {
	err := &githubAPIError{URL: "https://example.test", StatusCode: http.StatusTeapot, Status: "418 I'm a teapot"}
	if got, want := err.Error(), "GitHub API https://example.test: 418 I'm a teapot"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestGetJSON_InvalidURL(t *testing.T) {
	client := newGitHubClient()
	var out map[string]any
	if err := client.getJSON(context.Background(), "://invalid", &out); err == nil {
		t.Fatal("expected invalid URL error")
	}
}

func TestGetJSON_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	client := newGitHubClient()
	var out map[string]any
	if err := client.getJSON(context.Background(), srv.URL, &out); err == nil {
		t.Fatal("expected JSON decode error")
	}
}

func TestResolveAll_PropagatesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"failure"}`))
	}))
	defer srv.Close()

	client := newGitHubClient()
	client.baseURL = srv.URL
	_, err := resolveAll(context.Background(), client, []actionRef{{Owner: "o", Repo: "r", Ref: "v1"}})
	if err == nil {
		t.Fatal("expected resolveAll error")
	}
	var apiErr *githubAPIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected wrapped 500 API error, got %v", err)
	}
}
