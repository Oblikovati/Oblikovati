// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The tube-wrapping torus band now has ONE mesher, and this file is what holds that (#3517).
//
// It had two. spiricBandMesh, behind the classification arm kindSpiricBand, and torusTubeBandLoftMesh,
// a rung of meshSeamCrossingFace's ordered router, meshed the SAME shape: a torus face bounded by two
// edges that each go the whole way round the tube, bridged by one seam the loop walks twice. ADR-0061
// stage 5 put the first in FRONT of the second rather than replacing it, and the second then built
// nothing — 0 builds over ./kernel/... and ./model/... — so #3517 deleted it under the delete-first
// rule. What that buys is row two below: with the duplicate gone, removing the arm is no longer a
// quiet handover to a second loft; the band falls to the surface's WHOLE domain, and says so.
//
// The fixture's far rim is a fitted BSpline rather than a second circle for exactly that reason — see
// meridianRail. With two circles an earlier rung of the same router claims the face and row two would
// be measuring the wrong mesher.
//
// Row three is why the deletion is safe. A band that records a chart leaves the arm on purpose, and
// the general chart-driven mesher takes it — so nothing the arm gives up needed a second loft to
// catch it.

// tubeWrappingBandFace builds the shape the arm recognises: a torus band between the meridian circles
// at u=0 and u=3π/2, bridged by the outer-equator arc between them, which the loop walks in both
// directions (the artificial seam a tube-wrapping band cannot avoid).
func tubeWrappingBandFace(t *testing.T, majorR, minorR float64) *topo.Face {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "tubeband", 0))
	return buildTubeWrappingBand(t, weTestTorus(t, majorR, minorR), lin)
}

// buildTubeWrappingBand assembles that face: two meridian circles, one equator seam arc used twice.
func buildTubeWrappingBand(t *testing.T, tor geom.Torus, lin topo.Lineage) *topo.Face {
	t.Helper()
	uLo, uHi := 0.0, 3*stdmath.Pi/2
	bld := topo.NewBuilder(false, lin)
	vLo := bld.AddVertex(tor.PointAt(uLo, 0), lin)
	vHi := bld.AddVertex(tor.PointAt(uHi, 0), lin)
	seamArc, err := geom.Arc3dByThreePoints(tor.PointAt(uLo, 0), tor.PointAt((uLo+uHi)/2, 0), tor.PointAt(uHi, 0))
	if err != nil {
		t.Fatal(err)
	}
	seam := bld.AddEdge(seamArc, vLo, vHi, lin)
	lo := bld.AddEdge(meridianCircle(t, tor, uLo), vLo, vLo, lin)
	hi := bld.AddEdge(meridianRail(t, tor, uHi), vHi, vHi, lin)
	bld.AddFace(tor, lin, topo.OuterLoop(topo.Fwd(lo), topo.Fwd(seam), topo.Rev(hi), topo.Rev(seam)))
	return bld.Build().Faces()[0]
}

// meridianCircle is the torus's tube circle at azimuth u — the boundary that "goes the whole way round
// the tube". Its RefDir is the radial direction, so its own angle-0 point is tor.PointAt(u, 0).
func meridianCircle(t *testing.T, tor geom.Torus, u float64) geom.Circle {
	t.Helper()
	radial := tor.Ref.AsVector().Scale(math.Scalar(stdmath.Cos(u))).
		Add(tor.AxisDir.AsVector().Cross(tor.Ref.AsVector()).Scale(math.Scalar(stdmath.Sin(u))))
	normal, errN := math.UnitVector3FromVector(radial.Cross(tor.AxisDir.AsVector()))
	ref, errR := math.UnitVector3FromVector(radial)
	if errN != nil || errR != nil {
		t.Fatalf("meridian frame at u=%g is degenerate: normal %v, ref %v", u, errN, errR)
	}
	center := tor.Center.TranslateBy(radial.Scale(math.Scalar(tor.MajorRadius)))
	return geom.Circle{Center: center, Normal: normal, RefDir: ref, Radius: tor.MinorRadius}
}

// meridianRail is the SECOND boundary, and it is deliberately a fitted closed BSplineCurve rather than
// a second circle: on the two real hosts (occtparity simple/J3, bfuseblend/A4) the far rim is the canal
// contact rail, a BSpline, and closedBandLoftMesh's bandRingsAndSeam reads only geom.Circle and
// full-turn geom.Arc3d as rings. With two circles the fixture is claimed by THAT loft before the
// seam-crossing router ever gets to the tube-band rungs, and the rows below would be measuring the
// wrong mesher. Fitted through 48 samples of the meridian circle, so it lies on the torus to ~1e-6.
func meridianRail(t *testing.T, tor geom.Torus, u float64) geom.Curve3 {
	t.Helper()
	const samples = 48
	pts := make([]math.Point3, samples)
	for i := range pts {
		pts[i] = tor.PointAt(u, 2*stdmath.Pi*float64(i)/samples)
	}
	rail, _, err := geom.NewClosedFittedBSplineCurve(pts, geom.FitChordLength)
	if err != nil {
		t.Fatalf("fitting the meridian rail at u=%g: %v", u, err)
	}
	return rail
}

// TestTheSpiricArmClaimsTheTubeWrappingBand is the classification half: the band is kindSpiricBand's,
// and it is the only mesher left for the shape.
func TestTheSpiricArmClaimsTheTubeWrappingBand(t *testing.T) {
	t.Parallel()
	f := tubeWrappingBandFace(t, 20, 5)
	q := DefaultQuality()
	if _, ok := spiricTubeTrimOf(f, f.Geometry(), q); !ok {
		t.Fatal("spiricTubeTrimOf did not claim the tube-wrapping band — the rows below cover nothing")
	}
	got := classifyCurvedTrim(f, f.Geometry(), FaceOuterBoundary(f, q), faceHoleBoundaries(f, q), q)
	if got.kind != kindSpiricBand {
		t.Errorf("the tube-wrapping band classifies as %s, want %s", got.kind, kindSpiricBand)
	}
}

// TestWithoutTheArmTheBandFallsToTheReportedWholeDomain is the guard the deletion leaves in its place,
// and it is the reason the deletion is worth making.
//
// meshSeamCrossingFace is the route a face takes when the classification does NOT claim it, so driving
// it directly is exactly "what happens if kindSpiricBand goes". While torusTubeBandLoftMesh stood in
// it, the answer was a second loft and the only thing that noticed was two occtparity pins moving. The
// answer now is the surface's WHOLE domain with the degradation REPORTED — which is the loud corpus
// failure the gate exists for, and it reproduces on the real hosts: 2 097 152 triangles,
// 394 781.31 mm² against the band's 292 951, one diag.Defect (measured on J3 and A4, review 1 §1.1).
//
// Both halves are asserted, because either alone can pass for the wrong reason: the ARM must mesh the
// band's own region, and the router behind it must mesh the whole torus and say so.
func TestWithoutTheArmTheBandFallsToTheReportedWholeDomain(t *testing.T) {
	t.Parallel()
	f := tubeWrappingBandFace(t, 20, 5)
	q, s := DefaultQuality(), f.Geometry()
	outer, holes := FaceOuterBoundary(f, q), faceHoleBoundaries(f, q)
	band, torus := 0.75*wholeTorusArea, wholeTorusArea // the band is three quarters of the torus
	arm, special, _ := specialCurvedMesh(f, s, outer, holes, q, &chartDeclineLog{})
	if !special {
		t.Fatal("the classification did not mesh the band — the row covers nothing")
	}
	if got := MeshGeometryProperties(arm).Area; !withinChordDeficit(got, band) {
		t.Fatalf("the arm meshed %.4f mm², want the band's own %.4f less a chord deficit", got, band)
	}
	behind := meshSeamCrossingFace(f, s, outer, holes, q, "", &chartDeclineLog{})
	if got := MeshGeometryProperties(behind).Area; !withinChordDeficit(got, torus) {
		t.Errorf("the router BEHIND the classification meshed %.4f mm², want the whole torus %.4f — a "+
			"second loft for this shape has re-appeared in meshSeamCrossingFace", got, torus)
	}
	if !carriesDefect(behind, CodeTrimIgnoredFullDomain) {
		t.Errorf("the whole-domain fall-back was not reported as a defect: %v", behind.Diagnostics)
	}
}

// wholeTorusArea is 4π²Rr for the fixture's torus (R=20, r=5): 3947.8418 mm².
const wholeTorusArea = 4 * stdmath.Pi * stdmath.Pi * 20 * 5

// withinChordDeficit reports whether a faceted area is an analytic one less a chord deficit — under it,
// and by no more than 3 %.
func withinChordDeficit(got, want float64) bool { return got >= 0.97*want && got <= want }

// carriesDefect reports whether the mesh carries the given code at Defect severity.
func carriesDefect(m *Mesh, code diag.Code) bool {
	for _, d := range m.Diagnostics {
		if d.Code == code && d.Severity == diag.Defect {
			return true
		}
	}
	return false
}

// TestAChartedTubeWrappingBandGoesToTheChartMesher is why deleting the second loft is safe: a band that
// records its own region leaves the arm by design, and the GENERAL chart-driven mesher takes it — the
// router never reaches the seam-crossing rungs at all. The chart is the face's real rectangle
// (u ∈ [0, 3π/2] × v ∈ [0, 2π]), not a placeholder, so the row exercises the mesher and not just
// spiricTubeTrimOf's "is there a chart" early return.
func TestAChartedTubeWrappingBandGoesToTheChartMesher(t *testing.T) {
	t.Parallel()
	f := tubeWrappingBandFace(t, 20, 5)
	f.SetChart([][]math.Point2{tubeWrappingBandChart()})
	q, s := DefaultQuality(), f.Geometry()
	if _, ok := spiricTubeTrimOf(f, s, q); ok {
		t.Error("spiricTubeTrimOf claimed a CHARTED band; the chart-driven mesher owns that face")
	}
	m, ok := chartFaceMesh(f, s, q, &chartDeclineLog{})
	if !ok {
		t.Fatal("the chart-driven mesher gave up the charted band — then something must catch it, and " +
			"the second loft this task deleted was the only thing that could")
	}
	band := 0.75 * wholeTorusArea // three quarters of the torus
	if area := MeshGeometryProperties(m).Area; !withinChordDeficit(area, band) {
		t.Errorf("the chart mesher meshed %.4f mm², want the band's own %.4f less a chord deficit", area, band)
	}
}

// tubeWrappingBandChart is the face's region in the torus's covering space: u from the first meridian
// to the second, v the whole tube.
func tubeWrappingBandChart() []math.Point2 {
	uHi, twoPi := 3*stdmath.Pi/2, 2*stdmath.Pi
	return []math.Point2{math.P2(0, 0), math.P2(uHi, 0), math.P2(uHi, twoPi), math.P2(0, twoPi)}
}
