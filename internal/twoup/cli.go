package twoup

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// Main runs the twoup CLI flow and returns a process exit code.
func Main() int {
	cfg := config{}
	flag.StringVar(&cfg.root, "repo", ".", "path to the repository root")
	flag.BoolVar(&cfg.dryRun, "dry-run", false, "print proposed changes without writing files")
	flag.BoolVar(&cfg.verbose, "v", false, "print additional progress information to stdout")
	flag.Parse()

	absRoot, err := filepath.Abs(cfg.root)
	if err != nil {
		return exitErr(fmt.Errorf("resolve repo path: %w", err))
	}
	cfg.root = absRoot

	if err := ensureGitRepo(cfg.root); err != nil {
		return exitErr(err)
	}

	stats, err := run(cfg)
	if err != nil {
		return exitErr(err)
	}

	if stats.UpdatedLines == 0 {
		fmt.Println("twoup: all GitHub Actions are already pinned to latest release digests")
		return 0
	}

	mode := "updated"
	if cfg.dryRun {
		mode = "would update"
	}
	fmt.Printf("twoup: %s %d line(s) across %d file(s)\n", mode, stats.UpdatedLines, stats.UpdatedFiles)
	return 0
}

func exitErr(err error) int {
	if errors.Is(err, errNotGitRepo) {
		fmt.Fprintln(os.Stderr, "twoup:", err)
		return 2
	}
	fmt.Fprintln(os.Stderr, "twoup error:", err)
	return 1
}
