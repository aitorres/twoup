package twoup

import (
	"context"
	"fmt"
	"os"
	"strings"
)

func run(cfg config) (runStats, error) {
	return runWithClient(cfg, newGitHubClient())
}

func runWithClient(cfg config, client *githubClient) (runStats, error) {
	files, err := findWorkflowFiles(cfg.root)
	if err != nil {
		return runStats{}, fmt.Errorf("find workflows: %w", err)
	}
	if len(files) == 0 {
		return runStats{}, nil
	}

	type workflowFile struct {
		path               string
		hadTrailingNewline bool
		lines              []string
		refs               map[int]actionRef
	}

	workflowFiles := make([]workflowFile, 0, len(files))
	allRefs := make([]actionRef, 0, 32)
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return runStats{}, fmt.Errorf("read %s: %w", file, err)
		}
		lines := strings.Split(string(content), "\n")
		refs := make(map[int]actionRef)
		for i, line := range lines {
			ref, ok := parseUsesLine(line)
			if ok {
				refs[i] = ref
				allRefs = append(allRefs, ref)
			}
		}
		workflowFiles = append(workflowFiles, workflowFile{
			path:               file,
			hadTrailingNewline: strings.HasSuffix(string(content), "\n"),
			lines:              lines,
			refs:               refs,
		})
	}

	if len(allRefs) == 0 {
		return runStats{}, nil
	}

	resolvedByAction, err := resolveAll(context.Background(), client, allRefs)
	if err != nil {
		return runStats{}, err
	}

	stats := runStats{}
	for _, file := range workflowFiles {
		updatedInFile := 0
		for i, line := range file.lines {
			ref, ok := file.refs[i]
			if !ok {
				continue
			}
			resolved, ok := resolvedByAction[ref.Owner+"/"+ref.Repo]
			if !ok {
				continue
			}
			if isAlreadyPinned(line, ref, resolved) {
				continue
			}
			updated, changed := rewriteUsesLine(line, resolved)
			if !changed {
				continue
			}
			if cfg.verbose {
				fmt.Printf("%s: %s -> %s@%s # %s\n", file.path, ref.Owner+"/"+ref.Repo+"@"+ref.Ref, ref.Owner+"/"+ref.Repo, resolved.Digest, resolved.LatestTag)
			}
			file.lines[i] = updated
			updatedInFile++
		}

		if updatedInFile == 0 {
			continue
		}

		if !cfg.dryRun {
			updatedContent := strings.Join(file.lines, "\n")
			if !strings.HasSuffix(updatedContent, "\n") && file.hadTrailingNewline {
				updatedContent += "\n"
			}
			if err := os.WriteFile(file.path, []byte(updatedContent), 0o644); err != nil {
				return runStats{}, fmt.Errorf("write %s: %w", file.path, err)
			}
		}

		stats.UpdatedFiles++
		stats.UpdatedLines += updatedInFile
	}

	return stats, nil
}

func isAlreadyPinned(line string, ref actionRef, resolved resolvedAction) bool {
	if !isDigest(ref.Ref) || ref.Ref != resolved.Digest {
		return false
	}
	parts := strings.SplitN(line, "#", 2)
	if len(parts) < 2 {
		return false
	}
	return strings.TrimSpace(parts[1]) == resolved.LatestTag
}
