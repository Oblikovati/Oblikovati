// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestChartOfFaceLiftsALoopThatWalksItsSeamTwice is the row #3550 exists for: a torus band bridged by
// an artificial seam the loop walks in both directions. A point-to-point unwrap folds the second
// traversal onto the first's branch and the ring comes back two periods tall; lifting each edge use on
// its own and placing the uses by whole-period junction shifts gives the keyhole that closes.
//
// The band runs u ∈ [0, 3π/2] and the whole tube in v, so the chart must be one period tall and three
// quarters of a period wide — the shape of occtparity simple/J3's host torus.
func TestChartOfFaceLiftsALoopThatWalksItsSeamTwice(t *testing.T) {
	t.Parallel()
	f, tor := seamBridgedTorusBand(t)
	chart, ok := ChartOfFace(f)
	if !ok {
		t.Fatal("ChartOfFace declined a torus band whose loop closes through its own seam")
	}
	if len(chart) != 1 {
		t.Fatalf("chart has %d contours, want 1", len(chart))
	}
	u0, u1, v0, v1 := polyBoundsOf(chart[0])
	if got, want := u1-u0, 3*stdmath.Pi/2; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("chart spans %.6f in u, want the band's own %.6f", got, want)
	}
	if got := v1 - v0; stdmath.Abs(got-twoPi) > 1e-6 {
		t.Errorf("chart spans %.6f in v, want exactly one period %.6f — the seam collapsed or doubled",
			got, twoPi)
	}
	assertChartCoversTheBand(t, chart, tor)
}

// assertChartCoversTheBand checks the contour is the REGION and not merely the right size: a point
// inside the band reads material and a point in the complement does not.
func assertChartCoversTheBand(t *testing.T, chart [][]math.Point2, tor geom.Torus) {
	t.Helper()
	inside := math.P2(3*stdmath.Pi/4, stdmath.Pi) // mid-band, mid-tube
	outside := math.P2(7*stdmath.Pi/4, stdmath.Pi)
	if !chartContains(chart, inside, true, true) {
		t.Errorf("a point in the middle of the band reads as outside its own chart (torus R=%g r=%g)",
			tor.MajorRadius, tor.MinorRadius)
	}
	if chartContains(chart, outside, true, true) {
		t.Error("a point in the band's complement reads as material")
	}
}

// TestChartOfFaceDeclinesAnAperiodicSurface keeps the derivation off the faces ToUVLoops already
// charts: a plane has no branch to choose, so there is nothing for a chart to record.
func TestChartOfFaceDeclinesAnAperiodicSurface(t *testing.T) {
	t.Parallel()
	box, err := SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "test")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range box.Faces() {
		if _, ok := ChartOfFace(f); ok {
			t.Fatalf("ChartOfFace charted a %T face; only a periodic surface needs one", f.Geometry())
		}
	}
}

// seamBridgedTorusBand builds the fixture: a torus band between the meridian circles at u=0 and
// u=3π/2, bridged by the outer-equator arc between them, which the loop walks in both directions.
func seamBridgedTorusBand(t *testing.T) (*topo.Face, geom.Torus) {
	t.Helper()
	tor, err := geom.NewTorusWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 20, 5)
	if err != nil {
		t.Fatal(err)
	}
	uLo, uHi := 0.0, 3*stdmath.Pi/2
	lin := topo.NewLineage(topo.Tok("test", "seamband", 0))
	bld := topo.NewBuilder(false, lin)
	vLo := bld.AddVertex(tor.PointAt(uLo, 0), lin)
	vHi := bld.AddVertex(tor.PointAt(uHi, 0), lin)
	seamArc, err := geom.Arc3dByThreePoints(tor.PointAt(uLo, 0), tor.PointAt((uLo+uHi)/2, 0), tor.PointAt(uHi, 0))
	if err != nil {
		t.Fatal(err)
	}
	seam := bld.AddEdge(seamArc, vLo, vHi, lin)
	lo := bld.AddEdge(torusMeridian(t, tor, uLo), vLo, vLo, lin)
	hi := bld.AddEdge(torusMeridian(t, tor, uHi), vHi, vHi, lin)
	bld.AddFace(tor, lin, topo.OuterLoop(topo.Fwd(lo), topo.Fwd(seam), topo.Rev(hi), topo.Rev(seam)))
	return bld.Build().Faces()[0], tor
}

// torusMeridian is the tube circle at azimuth u, framed so its own angle-0 point is tor.PointAt(u, 0).
func torusMeridian(t *testing.T, tor geom.Torus, u float64) geom.Circle {
	t.Helper()
	radial := tor.Ref.AsVector().Scale(math.Scalar(stdmath.Cos(u))).
		Add(tor.AxisDir.AsVector().Cross(tor.Ref.AsVector()).Scale(math.Scalar(stdmath.Sin(u))))
	normal, errN := math.UnitVector3FromVector(radial.Cross(tor.AxisDir.AsVector()))
	ref, errR := math.UnitVector3FromVector(radial)
	if errN != nil || errR != nil {
		t.Fatalf("meridian frame at u=%g is degenerate: %v / %v", u, errN, errR)
	}
	return geom.Circle{
		Center: tor.Center.TranslateBy(radial.Scale(math.Scalar(tor.MajorRadius))),
		Normal: normal, RefDir: ref, Radius: tor.MinorRadius,
	}
}
