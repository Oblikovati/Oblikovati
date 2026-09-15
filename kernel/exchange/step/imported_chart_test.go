// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// An imported face on a PERIODIC surface records its parametric trim (Oblikovati/Oblikovati#3550).
//
// ADR-0063 made the chart part of a face's definition and put it on the producer that wound the face,
// because the 3-D loops of a face on a periodic surface do not say which of two complementary regions
// it is. `kernel/brep` records one on every face its (u, v) arrangement builds; nothing else did, so an
// imported body reached the tessellator with chart = nil on every face and the general chart-driven
// mesher declined it outright — which is what keeps the bespoke arms of the curved-trim classification
// alive. The assembler now derives it (brep.ChartOfFace) from the loops it has just wound.
//
// The cylinder fixture is the smallest body that exercises the distinction: its wall is a face on a
// periodic surface whose two rims each turn a whole period, and its two caps are planes, which need no
// chart and must not get one.
func TestAnImportedPeriodicFaceCarriesItsChart(t *testing.T) {
	t.Parallel()
	body := importOneSolid(t, "cylinder.step")
	charted, planar := 0, 0
	for i, f := range body.Faces() {
		if _, isPlane := f.Geometry().(geom.Plane); isPlane {
			planar++
			if len(f.Chart()) > 0 {
				t.Errorf("face %d is a %T and records a chart; only a periodic surface needs one",
					i, f.Geometry())
			}
			continue
		}
		if len(f.Chart()) == 0 {
			t.Errorf("face %d is a %T and records NO chart — the general mesher declines such a face",
				i, f.Geometry())
			continue
		}
		charted++
		assertChartIsTheWall(t, i, f)
	}
	if charted != 1 || planar != 2 {
		t.Fatalf("the cylinder presented %d charted and %d planar faces, want 1 and 2 — the fixture drifted",
			charted, planar)
	}
}

// assertChartIsTheWall checks the recorded contour is the wall's own region and not the surface's whole
// domain: one closed contour, a full turn in the periodic parameter, and the wall's height in the other.
func assertChartIsTheWall(t *testing.T, i int, f *topo.Face) {
	t.Helper()
	if len(f.Chart()) != 1 {
		t.Errorf("face %d records %d contours, want 1", i, len(f.Chart()))
		return
	}
	var u0, u1, v0, v1 = stdmath.Inf(1), stdmath.Inf(-1), stdmath.Inf(1), stdmath.Inf(-1)
	for _, p := range f.Chart()[0] {
		u0, u1 = stdmath.Min(u0, float64(p.X)), stdmath.Max(u1, float64(p.X))
		v0, v1 = stdmath.Min(v0, float64(p.Y)), stdmath.Max(v1, float64(p.Y))
	}
	if got, want := u1-u0, 2*stdmath.Pi; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("face %d spans %.6f in u, want one full turn %.6f", i, got, want)
	}
	if got, want := v1-v0, 20.0; stdmath.Abs(got-want) > 1e-6 { // the fixture's height
		t.Errorf("face %d spans %.6f in v, want the wall's own height %.6f", i, got, want)
	}
}
