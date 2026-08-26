// SPDX-License-Identifier: GPL-2.0-only

package addincat

import (
	"strconv"
	"strings"
)

// compareSemver orders two plain "major.minor.patch" version strings, returning -1, 0 or 1.
// A pre-release/build suffix is ignored and a missing or non-numeric component counts as 0 —
// sufficient because add-in versions are plain numeric semver (the API pins that upstream).
func compareSemver(a, b string) int {
	ai, bi := numericParts(a), numericParts(b)
	for i := range 3 {
		if ai[i] != bi[i] {
			if ai[i] < bi[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func numericParts(v string) [3]int {
	core, _, _ := strings.Cut(v, "-")
	core, _, _ = strings.Cut(core, "+")
	fields := strings.SplitN(core, ".", 3)
	var out [3]int
	for i := 0; i < len(fields) && i < 3; i++ {
		if n, err := strconv.Atoi(strings.TrimSpace(fields[i])); err == nil {
			out[i] = n
		}
	}
	return out
}
