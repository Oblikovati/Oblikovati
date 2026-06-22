// SPDX-License-Identifier: GPL-2.0-only

package update

import (
	"strconv"
	"strings"
)

// IsNewer reports whether latest is a strictly greater release than current. Both are
// the project's {MANUAL_MAJOR}.{API_VERSION}.{MINOR}.{PATCH} strings (see RELEASING.md);
// nightlies add a "-nightly.<timestamp>" prerelease. The four numeric fields compare as
// integers (API_VERSION is zero-padded, so it orders correctly); for two nightlies that
// share a core, the sortable UTC timestamp breaks the tie.
//
//	IsNewer("0.000200.2.0", "0.000200.1.5")                                    // => true
//	IsNewer("0.000200.1.0-nightly.26061503",
//	        "0.000200.1.0-nightly.26061403")                                   // => true
func IsNewer(latest, current string) bool {
	if c := compareCore(latest, current); c != 0 {
		return c > 0
	}
	return nightlyStamp(latest) > nightlyStamp(current)
}

// compareCore compares the numeric {MAJOR, API_VERSION, MINOR, PATCH} cores of two
// versions, returning -1, 0, or 1. A leading "v" and any "-prerelease" suffix are
// stripped first.
func compareCore(a, b string) int {
	ax, bx := coreParts(a), coreParts(b)
	for i := 0; i < 4; i++ {
		switch {
		case ax[i] > bx[i]:
			return 1
		case ax[i] < bx[i]:
			return -1
		}
	}
	return 0
}

// coreParts extracts [MAJOR, API_VERSION, MINOR, PATCH] as int64. A missing or
// non-numeric component is treated as 0, so a malformed version sorts low rather than
// panicking the caller.
func coreParts(v string) [4]int64 {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexByte(v, '-'); i >= 0 {
		v = v[:i]
	}
	var out [4]int64
	for i, p := range strings.SplitN(v, ".", 4) {
		out[i], _ = strconv.ParseInt(strings.TrimSpace(p), 10, 64)
	}
	return out
}

// nightlyStamp returns the sortable timestamp a nightly version carries after
// "-nightly." (e.g. "26061503"), or "" for a stable build — the tiebreaker
// between two nightlies that share a numeric core.
func nightlyStamp(v string) string {
	_, after, found := strings.Cut(v, nightlySuffix+".")
	if !found {
		return ""
	}
	return after
}
