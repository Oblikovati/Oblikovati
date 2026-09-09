// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// TestSeamWelderMergesAcrossACellBoundary pins the grid-boundary defect: a cell-exact lookup leaves
// two coincident points unmerged whenever they straddle a cell boundary, and which side they fall on
// is an accident of where the grid lies. Measured before the fix, two points a TENTH of the grid apart
// welded to distinct vertices.
//
// welder3 has searched its 26 neighbours since #879, where the cell-exact version shredded a fine-pitch
// coil join into unpaired coincident open edges. This is the same defect in the (u,v) welder.
func TestSeamWelderMergesAcrossACellBoundary(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		apart float64
		want  bool
	}{
		{"a tenth of the grid, straddling a boundary", seamWeldGrid / 10, true},
		{"most of the grid, straddling a boundary", seamWeldGrid * 0.9, true},
		{"three grids apart", seamWeldGrid * 3, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newSeamWelder(false, false)
			mid := seamWeldGrid / 2 // exactly on a cell boundary
			a := math.P2(mid-tc.apart/2, 1)
			b := math.P2(mid+tc.apart/2, 1)
			if merged := w.add(a) == w.add(b); merged != tc.want {
				t.Errorf("points %g apart (grid %g) merged = %v, want %v",
					float64(a.DistanceTo(b)), seamWeldGrid, merged, tc.want)
			}
		})
	}
}
