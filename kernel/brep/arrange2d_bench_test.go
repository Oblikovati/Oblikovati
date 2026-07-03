// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"
	"testing"

	"oblikovati.org/math"
)

// crossingGridSegments builds n/2 horizontal × n/2 vertical segments whose pairwise crossings
// number (n/2)² — the dense-imprint arrangement shape (a faceted-curve cut across a face) that
// made the O(S²) planarize scan the boolean's hot spot (#1607).
func crossingGridSegments(n int) [][2]math.Point2 {
	segs := make([][2]math.Point2, 0, n)
	for i := 0; i < n/2; i++ {
		v := float64(i) + 0.5
		segs = append(segs,
			[2]math.Point2{math.P2(0, v), math.P2(float64(n/2), v)},
			[2]math.Point2{math.P2(v, 0), math.P2(v, float64(n/2))})
	}
	return segs
}

// BenchmarkPlanarizeCrossingGrid locks the arrangement front end's asymptotics at 1×/4×/16×
// segment counts: with the grid-hash candidacy the per-tier cost must grow with the crossing
// count (output size), not the S² pair count.
func BenchmarkPlanarizeCrossingGrid(b *testing.B) {
	for _, n := range []int{16, 64, 256} {
		segs := crossingGridSegments(n)
		b.Run(fmt.Sprintf("segments_%d", len(segs)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				planarize(segs)
			}
		})
	}
}
