// SPDX-License-Identifier: GPL-2.0-only

package release

import (
	"regexp"
	"strconv"
	"strings"
)

// Scope is the release-relevant scope of a set of commits, strongest-wins. MANUAL_MAJOR
// is the application's only "major" (hand-set), so a breaking change maps to a MINOR
// bump here, not a MAJOR one — the same 0.x policy the api contract uses.
type Scope int

const (
	ScopeNone     Scope = iota // docs/chore/ci/test/style/build/refactor — no release
	ScopePatch                 // fix/perf/revert, or an unrecognized subject (safe floor)
	ScopeFeature               // feat — an additive change
	ScopeBreaking              // "!" marker or a "BREAKING CHANGE:" footer
)

var (
	headerRe         = regexp.MustCompile(`^([A-Za-z]+)(?:\([^)]*\))?(!)?:`)
	breakingFooterRe = regexp.MustCompile(`(?m)^BREAKING[ -]CHANGE:`)
	patchTypes       = map[string]bool{"fix": true, "perf": true, "revert": true}
	silentTypes      = map[string]bool{
		"docs": true, "chore": true, "ci": true, "test": true,
		"style": true, "build": true, "refactor": true,
	}
)

// classifyOne returns the scope of a single (full) commit message.
func classifyOne(message string) Scope {
	if breakingFooterRe.MatchString(message) {
		return ScopeBreaking
	}
	m := headerRe.FindStringSubmatch(strings.TrimSpace(message))
	if m == nil {
		return ScopePatch // non-conventional subject — conservative floor, never None
	}
	if m[2] == "!" {
		return ScopeBreaking
	}
	t := strings.ToLower(m[1])
	switch {
	case t == "feat":
		return ScopeFeature
	case patchTypes[t]:
		return ScopePatch
	case silentTypes[t]:
		return ScopeNone
	default:
		return ScopePatch // unknown type (e.g. "wire:") — a patch, not a no-op
	}
}

// Classify returns the strongest scope across all commit messages.
func Classify(messages []string) Scope {
	strongest := ScopeNone
	for _, m := range messages {
		if s := classifyOne(m); s > strongest {
			strongest = s
		}
	}
	return strongest
}

// NextMinorPatch bumps (minor, patch) for a change of the given scope, per SemVer:
// feat/breaking bump MINOR and reset PATCH; fix bumps PATCH; none holds.
func NextMinorPatch(minor, patch int, s Scope) (int, int) {
	switch s {
	case ScopeNone:
		return minor, patch
	case ScopePatch:
		return minor, patch + 1
	default: // feature or breaking
		return minor + 1, 0
	}
}

// ParseVersionTag extracts (minor, patch) from a tag that begins with prefix
// ("v{major}.{apiField}.") and whose remainder is exactly "{minor}.{patch}". The bool
// is false for any tag that does not match that exact shape.
func ParseVersionTag(tag, prefix string) (minor, patch int, ok bool) {
	rest, found := strings.CutPrefix(tag, prefix)
	if !found {
		return 0, 0, false
	}
	parts := strings.Split(rest, ".")
	if len(parts) != 2 {
		return 0, 0, false
	}
	mi, err1 := strconv.Atoi(parts[0])
	pa, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || mi < 0 || pa < 0 {
		return 0, 0, false
	}
	return mi, pa, true
}
