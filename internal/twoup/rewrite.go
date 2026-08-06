package twoup

import (
	"regexp"
	"strings"
)

var usesPattern = regexp.MustCompile(`^(\s*-?\s*uses\s*:\s*)(['\"]?)([^'\"\s#]+)(['\"]?)(\s*(#.*)?)$`)

type parsedUsesLine struct {
	prefix     string
	openQuote  string
	action     string
	ref        string
	closeQuote string
}

func parseUsesLine(line string) (actionRef, bool) {
	parsed, ok := parseUsesLineParts(line)
	if !ok {
		return actionRef{}, false
	}
	owner, repo, ok := strings.Cut(parsed.action, "/")
	if !ok {
		return actionRef{}, false
	}
	return actionRef{Owner: owner, Repo: repo, Ref: parsed.ref}, true
}

func rewriteUsesLine(line string, resolved resolvedAction) (string, bool) {
	parsed, ok := parseUsesLineParts(line)
	if !ok {
		return line, false
	}

	rewritten := parsed.prefix + parsed.openQuote + parsed.action + "@" + resolved.Digest + parsed.closeQuote + " # " + resolved.LatestTag

	if rewritten == line {
		return line, false
	}
	return rewritten, true
}

func parseUsesLineParts(line string) (parsedUsesLine, bool) {
	match := usesPattern.FindStringSubmatch(line)
	if match == nil {
		return parsedUsesLine{}, false
	}

	full := strings.TrimSpace(match[3])
	if strings.HasPrefix(full, "./") || strings.HasPrefix(full, "docker://") {
		return parsedUsesLine{}, false
	}

	action, ref, ok := splitActionRef(full)
	if !ok {
		return parsedUsesLine{}, false
	}

	if strings.Count(action, "/") != 1 {
		return parsedUsesLine{}, false
	}

	return parsedUsesLine{
		prefix:     match[1],
		openQuote:  match[2],
		action:     action,
		ref:        ref,
		closeQuote: match[4],
	}, true
}

func splitActionRef(v string) (action string, ref string, ok bool) {
	idx := strings.LastIndex(v, "@")
	if idx <= 0 || idx >= len(v)-1 {
		return "", "", false
	}
	return v[:idx], v[idx+1:], true
}

func isDigest(ref string) bool {
	if len(ref) != 40 {
		return false
	}
	for _, c := range ref {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
