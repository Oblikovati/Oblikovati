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
// exposed the trim-classification gaps. brep.PointInside is now the generalized-winding solid-angle flux
// (winding_flux.go): the total solid angle each analytic face subtends at p, ≈4π inside and ≈0 outside,
// robust to an imported B-rep's seam gaps (a continuous field) and to unreliable face orientation. Every
// clear in/out probe must agree with the mesh winding oracle.
//
// An ON-boundary probe is EXCLUDED (via ClassifyPoint == OnSurface): there the mesh oracle is not ground
// truth — a point on a curved face lands on whichever side the chord facets happen to fall, so the mesh's
// verdict is a faceting coin-flip. That the analytic classifier declines to reproduce that artifact is
// the point of the effort, not a regression, so those probes are classified ON and not compared.
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
			if brep.ClassifyPoint(body, p) == brep.OnSurface {
				continue // on the boundary: the mesh oracle is a faceting coin-flip here, not ground truth
			}
			if got, want := brep.PointInside(body, p), ops.PointInsideBody(body, p); got != want {
				t.Errorf("%s: brep.PointInside(%v) = %v, mesh oracle = %v", r.Case, p, got, want)
			}
		}
	}
}
