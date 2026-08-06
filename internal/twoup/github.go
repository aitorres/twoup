package twoup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

const githubAPIBase = "https://api.github.com"

type githubClient struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

func newGitHubClient() *githubClient {
	return &githubClient{
		httpClient: &http.Client{Timeout: 12 * time.Second},
		token:      strings.TrimSpace(os.Getenv("GITHUB_TOKEN")),
		baseURL:    githubAPIBase,
	}
}

func (c *githubClient) resolveLatest(ctx context.Context, a actionRef) (resolvedAction, error) {
	tag, err := c.latestReleaseTag(ctx, a.Owner, a.Repo)
	if err != nil {
		return resolvedAction{}, err
	}
	sha, err := c.tagToCommitSHA(ctx, a.Owner, a.Repo, tag)
	if err != nil {
		return resolvedAction{}, err
	}
	return resolvedAction{LatestTag: tag, Digest: sha}, nil
}

func resolveAll(ctx context.Context, client *githubClient, refs []actionRef) (map[string]resolvedAction, error) {
	unique := make(map[string]actionRef, len(refs))
	for _, ref := range refs {
		unique[ref.Owner+"/"+ref.Repo] = ref
	}

	results := make(map[string]resolvedAction, len(unique))
	var mu sync.Mutex
	var wg sync.WaitGroup
	errCh := make(chan error, len(unique))
	sem := make(chan struct{}, 8)

	for _, ref := range unique {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			resolved, err := client.resolveLatest(ctx, ref)
			if err != nil {
				errCh <- fmt.Errorf("resolve %s/%s: %w", ref.Owner, ref.Repo, err)
				return
			}

			mu.Lock()
			results[ref.Owner+"/"+ref.Repo] = resolved
			mu.Unlock()
		})
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (c *githubClient) latestReleaseTag(ctx context.Context, owner, repo string) (string, error) {
	type response struct {
		TagName string `json:"tag_name"`
	}
	var out response
	u := c.baseURL + path.Join("/repos", owner, repo, "releases", "latest")
	if err := c.getJSON(ctx, u, &out); err != nil {
		var apiErr *githubAPIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return c.latestTag(ctx, owner, repo)
		}
		return "", err
	}
	if out.TagName == "" {
		return "", fmt.Errorf("latest release has no tag")
	}
	return out.TagName, nil
}

func (c *githubClient) latestTag(ctx context.Context, owner, repo string) (string, error) {
	type tag struct {
		Name string `json:"name"`
	}
	var out []tag
	u := c.baseURL + path.Join("/repos", owner, repo, "tags") + "?per_page=1"
	if err := c.getJSON(ctx, u, &out); err != nil {
		return "", err
	}
	if len(out) == 0 || out[0].Name == "" {
		return "", fmt.Errorf("repository has no tags")
	}
	return out[0].Name, nil
}

func (c *githubClient) tagToCommitSHA(ctx context.Context, owner, repo, tag string) (string, error) {
	type refObject struct {
		SHA  string `json:"sha"`
		Type string `json:"type"`
	}
	type gitObjectResponse struct {
		Object refObject `json:"object"`
	}

	var rr gitObjectResponse
	u := c.baseURL + path.Join("/repos", owner, repo, "git", "ref", "tags", tag)
	if err := c.getJSON(ctx, u, &rr); err != nil {
		return "", err
	}

	switch rr.Object.Type {
	case "commit":
		return rr.Object.SHA, nil
	case "tag":
		uTag := c.baseURL + path.Join("/repos", owner, repo, "git", "tags", rr.Object.SHA)
		if err := c.getJSON(ctx, uTag, &rr); err != nil {
			return "", err
		}
		if rr.Object.Type != "commit" || rr.Object.SHA == "" {
			return "", fmt.Errorf("tag %s does not resolve to a commit", tag)
		}
		return rr.Object.SHA, nil
	default:
		return "", fmt.Errorf("unsupported ref object type: %s", rr.Object.Type)
	}
}

func (c *githubClient) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "twoup")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &githubAPIError{URL: url, StatusCode: resp.StatusCode, Status: resp.Status}
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(out); err != nil {
		return err
	}
	return nil
}

type githubAPIError struct {
	URL        string
	StatusCode int
	Status     string
}

func (e *githubAPIError) Error() string {
	return fmt.Sprintf("GitHub API %s: %s", e.URL, e.Status)
}
