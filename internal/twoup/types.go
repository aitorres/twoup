package twoup

type config struct {
	root    string
	dryRun  bool
	verbose bool
}

type actionRef struct {
	Owner string
	Repo  string
	Ref   string
}

type resolvedAction struct {
	LatestTag string
	Digest    string
}

type runStats struct {
	UpdatedFiles int
	UpdatedLines int
}
