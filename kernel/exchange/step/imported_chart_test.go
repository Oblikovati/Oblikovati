// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// An imported face on a PERIODIC surface can record its parametric trim (Oblikovati/Oblikovati#3550).
//
// ADR-0063 made the chart part of a face's definition and put it on the producer that wound the face,
// because the 3-D loops of a face on a periodic surface do not say which of two complementary regions
// it is. kernel/brep records one on every face its (u, v) arrangement builds; nothing else did, so an
// imported body reached the tessellator with chart = nil on every face and the general chart-driven
// mesher declined it outright — which is what keeps the bespoke arms of the curved-trim classification
// alive.
//
// The assembler records it (recordImportedChart), and this row drives the SHIPPED body rather than the
// derivation, so it holds the producer being on and not merely available.
//
// The cylinder fixture is the smallest body that exercises the distinction: its wall is a face on a
// periodic surface whose loop bridges its two rims with a seam, and its two caps are planes, which need
// no chart and must not get one.
func TestAnImportedPeriodicFaceCarriesItsChart(t *testing.T) {
	t.Parallel()
	body := importOneSolid(t, "cylinder.step")
	charted, planar := 0, 0
	for i, f := range body.Faces() {
		chart := f.Chart()
		ok := len(chart) > 0
		if _, isPlane := f.Geometry().(geom.Plane); isPlane {
			planar++
			if ok {
				t.Errorf("face %d is a %T and records a chart; only a periodic surface needs one",
					i, f.Geometry())
			}
			continue
		}
		if !ok {
			t.Errorf("face %d is a %T and records NO chart — the general mesher declines such a face",
				i, f.Geometry())
			continue
		}
		charted++
		assertChartIsTheWall(t, i, chart)
	}
	if charted != 1 || planar != 2 {
		t.Fatalf("the cylinder presented %d charted and %d planar faces, want 1 and 2 — the fixture drifted",
			charted, planar)
	}
}

// assertChartIsTheWall checks the derived contour is the wall's own region and not the surface's whole
// domain: one closed contour, a full turn in the periodic parameter, and the wall's height in the other.
func assertChartIsTheWall(t *testing.T, i int, chart [][]math.Point2) {
	t.Helper()
	if len(chart) != 1 {
		t.Errorf("face %d records %d contours, want 1", i, len(chart))
		return
	}
	u0, u1, v0, v1 := stdmath.Inf(1), stdmath.Inf(-1), stdmath.Inf(1), stdmath.Inf(-1)
	for _, p := range chart[0] {
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

// TestEveryImportedPeriodicFaceIsCharted is the other direction: no periodic face may arrive without
// one, because an uncharted periodic face is exactly what the deleted bespoke arms existed to serve.
// The cost of it reaching the covering mesher is held by kernel/ops/tessellate's TestTessellationBudget
// (1.49 s against 2.15 s with this on; 6.37 s before coverShear).
func TestEveryImportedPeriodicFaceIsCharted(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"cylinder.step", "box_hole.step"} {
		for i, f := range importOneSolid(t, name).Faces() {
			if _, isPlane := f.Geometry().(geom.Plane); isPlane || len(f.Chart()) > 0 {
				continue
			}
			t.Errorf("%s face %d is a %T and records no chart", name, i, f.Geometry())
		}
	}
	_ = topo.Face{}
}
