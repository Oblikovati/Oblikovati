// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// analyticParityProbes are interior/exterior probe points spanning the B1 seam bodies' bounding
// range; the analytic point classifier must agree with the (ground-truth) tessellated winding oracle.
var analyticParityProbes = []math.Point3{
	{X: 0, Y: 0, Z: -1}, {X: 0, Y: 0, Z: 0}, {X: -5, Y: -5, Z: 0}, {X: 5, Y: 5, Z: 5},
	{X: 0, Y: 0, Z: 5}, {X: 20, Y: 20, Z: 20}, {X: 3, Y: -2, Z: 1}, {X: -8, Y: 4, Z: -3},
}

// TestAnalyticClassifierMatchesMeshOnSeamBodies is the regression gate for the analytic point-in-solid
// classifier (brep.PointInside) on the B1 imported seam bodies — the curved (sphere/NURBS) faces that
// exposed the trim-classification gaps. Every case must agree with the mesh winding oracle at EVERY
// probe: the classifier cross-checks the nearest-point normal test against ray parity, so it is robust
// both near an edge (where a ray grazes) and in the clear interior (where an imported face's orientation
// fools the nearest-point test).
func TestAnalyticClassifierMatchesMeshOnSeamBodies(t *testing.T) {
	fixtureDir := CorpusFixtureDir()
	for _, r := range Corpus() {
		if r.Grid != "simple" || !b1SeamCases[r.Case] {
			continue
		}
		body, err := importInput(filepath.Join(fixtureDir, r.InputStep))
		if err != nil {
			t.Fatalf("%s: import failed: %v", r.Case, err)
		}
		for _, p := range analyticParityProbes {
			if got, want := brep.PointInside(body, p), ops.PointInsideBody(body, p); got != want {
				t.Errorf("%s: brep.PointInside(%v) = %v, mesh oracle = %v", r.Case, p, got, want)
			}
		}
	}
}
