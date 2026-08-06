# twoup

A CLI tool to pin GitHub Actions to latest release digests in GitHub workflow files.

## What's this?

`twoup`[^1] scans workflow files in `.github/workflows` and finds `uses:` references, then rewrites each reference to the latest release digest resolved from the public GitHub API (or using an optionally-set `GITHUB_TOKEN` environment variable).

You can run `twoup` in any Git repository to ensure your GitHub Actions are up-to-date, and pinned to immutable commit digests for security and reproducibility.

Your workflow files will end up with `uses:` references like this:

```yaml
uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
```

[^1]: Because I already had an update-related project named [`oneup`](https://github.com/aitorres/oneup) :-)

## Installation

You can build `twoup` from source using the Go toolchain:

```bash
go build -o twoup ./cmd/twoup
```

Then run `./twoup` from inside any Git repository.

## Usage

Run in the current repository:

```bash
./twoup
```

Preview changes without modifying any file:

```bash
./twoup -dry-run
```

Use a custom repository path:

```bash
./twoup -repo /path/to/repo
```

Verbose output:

```bash
./twoup -v
```

Show all flags:

```bash
./twoup -h
```

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate; a minimum coverage of 80% is expected (and enforced by Github Actions!).

## License

No license file is currently included in this repository.

This project is licensed under the [GNU Affero General Public License v3.0](https://github.com/aitorres/twoup/blob/main/LICENSE).
