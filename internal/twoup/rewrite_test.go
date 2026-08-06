package twoup

import "testing"

func TestParseUsesLine(t *testing.T) {
	tests := []struct {
		line string
		ok   bool
		ref  actionRef
	}{
		{
			line: "      - uses: actions/checkout@v5",
			ok:   true,
			ref:  actionRef{Owner: "actions", Repo: "checkout", Ref: "v5"},
		},
		{
			line: "- uses: './.github/actions/setup'",
			ok:   false,
		},
		{
			line: "- uses: docker://alpine:3.20",
			ok:   false,
		},
		{
			line: "run: echo hello",
			ok:   false,
		},
	}

	for _, tc := range tests {
		got, ok := parseUsesLine(tc.line)
		if ok != tc.ok {
			t.Fatalf("parseUsesLine(%q) ok=%v want=%v", tc.line, ok, tc.ok)
		}
		if ok && got != tc.ref {
			t.Fatalf("parseUsesLine(%q) ref=%+v want=%+v", tc.line, got, tc.ref)
		}
	}
}

func TestRewriteUsesLine(t *testing.T) {
	line := "      - uses: actions/checkout@v4"
	resolved := resolvedAction{LatestTag: "v7.0.1", Digest: "3d3c42e5aac5ba805825da76410c181273ba90b1"}

	got, changed := rewriteUsesLine(line, resolved)
	if !changed {
		t.Fatal("expected rewrite to change line")
	}
	want := "      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1"
	if got != want {
		t.Fatalf("rewriteUsesLine()=%q want=%q", got, want)
	}
}

func TestIsAlreadyPinned(t *testing.T) {
	line := "- uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1"
	ref := actionRef{Owner: "actions", Repo: "checkout", Ref: "3d3c42e5aac5ba805825da76410c181273ba90b1"}
	resolved := resolvedAction{LatestTag: "v7.0.1", Digest: "3d3c42e5aac5ba805825da76410c181273ba90b1"}
	if !isAlreadyPinned(line, ref, resolved) {
		t.Fatal("expected true for already-pinned line")
	}
}
