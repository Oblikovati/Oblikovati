// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// TestWelder3MergesAcrossGridBoundary is the unit regression for #879: two coincident points
// that hash to DIFFERENT grid cells (they straddle a cell boundary) must still weld to one
// vertex. The cell-exact lookup used before left such pairs unmerged, which shredded dense
// self-proximate geometry (a fine-pitch coil-join) into coincident, unpaired open edges.
func TestWelder3MergesAcrossGridBoundary(t *testing.T) {
	t.Parallel()
	w := newWelder3(1e-6)
	// 0.00050049 and 0.00050051 are 2e-8 cm apart (well within weldGrid) but round to cells
	// 500 and 501 — opposite sides of a 1e-6 grid boundary.
	a := w.add(math.P3(0.00050049, 0, 0))
	b := w.add(math.P3(0.00050051, 0, 0))
	if a != b {
		t.Errorf("coincident points across a grid boundary welded to %d and %d, want one vertex", a, b)
	}
	// A point farther than the grid stays a distinct vertex.
	if c := w.add(math.P3(0.01, 0, 0)); c == a {
		t.Errorf("a point %g cm away merged with the first; weld tolerance is %g", 0.01, w.grid)
	}
}

// TestRingSignatureImmuneToCellStraddle: two copies of one ring whose vertices sit a hair either
// side of a weld-grid cell boundary (the #879 straddle) must produce the SAME signature, so
// mergeFilledHoles still dissolves the filled hole. The retired exact-cell string keys rounded the
// two copies into different cells (#1602).
func TestRingSignatureImmuneToCellStraddle(t *testing.T) {
	t.Parallel()
	pw := newWelder3(planarStitchGrid)
	ring := []math.Point3{math.P3(0.00050049, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0)}
	copyRing := []math.Point3{math.P3(0.00050051, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0)}
	if ringSignature(ring, pw) != ringSignature(copyRing, pw) {
		t.Error("two coincident rings straddling a weld-grid cell boundary produced different signatures")
	}
	other := []math.Point3{math.P3(0.5, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0)}
	if ringSignature(ring, pw) == ringSignature(other, pw) {
		t.Error("genuinely different rings produced the same signature")
	}
}
