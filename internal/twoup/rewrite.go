package twoup

import (
	"regexp"
	"strings"
)

var usesPattern = regexp.MustCompile(`^(\s*-?\s*uses\s*:\s*)(['\"]?)([^'\"\s#]+)(['\"]?)(\s*(#.*)?)$`)

func parseUsesLine(line string) (actionRef, bool) {
	match := usesPattern.FindStringSubmatch(line)
	if match == nil {
		return actionRef{}, false
	}

	full := strings.TrimSpace(match[3])
	if strings.HasPrefix(full, "./") || strings.HasPrefix(full, "docker://") {
		return actionRef{}, false
	}

	action, ref, ok := splitActionRef(full)
	if !ok {
		return actionRef{}, false
	}

	ownerRepo := strings.Split(action, "/")
	if len(ownerRepo) != 2 {
		return actionRef{}, false
	}
	return actionRef{Owner: ownerRepo[0], Repo: ownerRepo[1], Ref: ref}, true
}

func rewriteUsesLine(line string, resolved resolvedAction) (string, bool) {
	match := usesPattern.FindStringSubmatch(line)
	if match == nil {
		return line, false
	}

	prefix := match[1]
	openQuote := match[2]
	full := strings.TrimSpace(match[3])
	closeQuote := match[4]

	action, _, ok := splitActionRef(full)
	if !ok {
		return line, false
	}

	rewritten := prefix + openQuote + action + "@" + resolved.Digest + closeQuote + " # " + resolved.LatestTag

	if rewritten == line {
		return line, false
	}
	return rewritten, true
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
