// SPDX-License-Identifier: GPL-2.0-only

package persistence

import "strings"

// cachePrefix marks regenerable derived data (tessellation, previews) that may be
// dropped without data loss — recompute rebuilds it (architecture core/05).
const cachePrefix = "cache/"

// Compact drops every regenerable cache stream from the package and returns the
// number of uncompressed bytes reclaimed. The recipe streams (manifest, model,
// identity, attributes) are never touched, so compaction is lossless: a compacted
// package opens identically, it just omits derived data the next recompute remakes.
func Compact(p *Package) int {
	var reclaimed int
	for _, stat := range p.Streams() {
		if strings.HasPrefix(stat.Name, cachePrefix) {
			reclaimed += int(stat.Size)
			p.DeleteStream(stat.Name)
		}
	}
	return reclaimed
}
