// SPDX-License-Identifier: GPL-2.0-only

package update

import (
	"strconv"
	"strings"
)

// IsNewer reports whether latest is a strictly greater release than current. Both are
// the project's MAJOR.MINOR.PATCH[-prerelease] strings (see scripts/version.sh): the
// PATCH is a UTC build timestamp, so within a channel the comparison is monotonic. The
// prerelease suffix is ignored — a nightly is only ever compared to other nightlies,
// which all carry the same "-nightly" tag, so the numeric core is what differs.
//
//	IsNewer("0.0.20260615030000", "0.0.20260614120000") // => true
func IsNewer(latest, current string) bool { return compareCore(latest, current) > 0 }

// compareCore compares the numeric MAJOR.MINOR.PATCH cores of two versions, returning
// -1, 0, or 1. A leading "v" and any "-prerelease" suffix are stripped first.
func compareCore(a, b string) int {
	ax, bx := coreParts(a), coreParts(b)
	for i := 0; i < 3; i++ {
		switch {
		case ax[i] > bx[i]:
			return 1
		case ax[i] < bx[i]:
			return -1
		}
	}
	return 0
}

// coreParts extracts [MAJOR, MINOR, PATCH] as int64 (PATCH is a 14-digit timestamp, too
// large for a 32-bit int). A missing or non-numeric component is treated as 0, so a
// malformed version sorts low rather than panicking the caller.
func coreParts(v string) [3]int64 {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexByte(v, '-'); i >= 0 {
		v = v[:i]
	}
	var out [3]int64
	for i, p := range strings.SplitN(v, ".", 3) {
		out[i], _ = strconv.ParseInt(strings.TrimSpace(p), 10, 64)
	}
	return out
}
