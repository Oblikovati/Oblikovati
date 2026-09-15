// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
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
// This row drives the DERIVATION on a real imported face, which is the half that is finished. The
// assembler's switch is off (see recordImportedChart and the row below): a face whose trim does not
// DEVELOP still reaches the covering, and the covering is not affordable yet.
//
// The cylinder fixture is the smallest body that exercises the distinction: its wall is a face on a
// periodic surface whose loop bridges its two rims with a seam, and its two caps are planes, which need
// no chart and must not get one.
func TestAnImportedPeriodicFaceCanDeriveItsChart(t *testing.T) {
	t.Parallel()
	body := importOneSolid(t, "cylinder.step")
	charted, planar := 0, 0
	for i, f := range body.Faces() {
		chart, ok := brep.ChartOfFace(f)
		if _, isPlane := f.Geometry().(geom.Plane); isPlane {
			planar++
			if ok {
				t.Errorf("face %d is a %T and derived a chart; only a periodic surface needs one",
					i, f.Geometry())
			}
			continue
		}
		if !ok {
			t.Errorf("face %d is a %T derived NO chart — the general mesher declines such a face",
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
		t.Errorf("face %d derives %d contours, want 1", i, len(chart))
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

// TestImportedChartsAreNotRecordedYet pins the switch and, more importantly, pins WHY — so the next
// worker flips it against a gate rather than a hunch. A face whose trim does not develop reaches the
// covering, and kernel/ops/tessellate's TestImportedAnalyticPrimitivesWatertight measured that at 73 s
// with the producer on against 0.95 s with it off, on a 60 s tier-1 guard budget. That gate going
// green is the condition.
func TestImportedChartsAreNotRecordedYet(t *testing.T) {
	t.Parallel()
	for i, f := range importOneSolid(t, "cylinder.step").Faces() {
		if len(f.Chart()) > 0 {
			t.Errorf("face %d records a chart: the producer is on, so both TestTessellationBudget and "+
				"TestImportedAnalyticPrimitivesWatertight in kernel/ops/tessellate must be inside their "+
				"budgets with it on — check them before deleting this row", i)
		}
		_ = topo.Face{}
	}
}
