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

// The tube-wrapping torus band has NO bespoke mesher any more, and this file is what holds that
// (#3517).
//
// It had two. spiricBandMesh, behind the classification arm kindSpiricBand, and torusTubeBandLoftMesh,
// a rung of meshSeamCrossingFace's ordered router, meshed the SAME shape: a torus face bounded by two
// edges that each go the whole way round the tube, bridged by one seam the loop walks twice. ADR-0061
// stage 5 put the first in FRONT of the second rather than replacing it. Round 1 of #3517 deleted the
// shadowed rung; this round deleted the arm, because the general chart-driven mesher can finally serve
// the shape — #3550 gave the two real hosts a chart, fillet_rim_build.go's winding let the convex one
// CLOSE in the covering space, and coverShear made the covering affordable.
//
// So the two rows are the two halves of "absorbed": a CHARTED band is meshed by the general mesher
// over its own region, and an UNCHARTED one — the case no producer has recorded, which is what the arm
// used to hide — falls to the surface's whole domain and SAYS SO.
//
// The fixture's far rim is a fitted BSpline rather than a second circle — see meridianRail. With two
// circles an earlier rung of the seam-crossing router claims the face and the rows below would be
// measuring the wrong mesher.

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

// TestTheGeneralMesherServesTheTubeWrappingBand is the first half of "absorbed": the band records its
// own region and the GENERAL chart-driven mesher meshes it — no classification arm, no router rung.
// The chart is the face's real rectangle (u ∈ [0, 3π/2] × v ∈ [0, 2π]), so the row exercises the mesher
// and not a placeholder.
func TestTheGeneralMesherServesTheTubeWrappingBand(t *testing.T) {
	t.Parallel()
	f := tubeWrappingBandFace(t, 20, 5)
	f.SetChart([][]math.Point2{tubeWrappingBandChart()})
	q, s := DefaultQuality(), f.Geometry()
	outer, holes := FaceOuterBoundary(f, q), faceHoleBoundaries(f, q)
	got := classifyCurvedTrim(f, s, outer, holes, q)
	if got.kind != kindChart {
		t.Fatalf("the charted tube-wrapping band classifies as %s, want %s — a bespoke arm has "+
			"re-appeared for this shape", got.kind, kindChart)
	}
	m, ok := chartFaceMesh(f, s, q, &chartDeclineLog{})
	if !ok {
		t.Fatal("the chart-driven mesher gave up the band it is now the only mesher for")
	}
	band := 0.75 * wholeTorusArea
	if area := MeshGeometryProperties(m).Area; !withinChordDeficit(area, band) {
		t.Errorf("the general mesher meshed %.4f mm², want the band's own %.4f less a chord deficit",
			area, band)
	}
}

// TestWithoutAChartTheBandFallsToTheReportedWholeDomain is the other half. An uncharted band is the
// case the deleted arm existed to serve, and the honest answer for it is the surface's WHOLE domain
// with the degradation REPORTED — not a second loft that meshes it silently by its own rule. Both
// halves are asserted, because either alone can pass for the wrong reason.
func TestWithoutAChartTheBandFallsToTheReportedWholeDomain(t *testing.T) {
	t.Parallel()
	f := tubeWrappingBandFace(t, 20, 5)
	q, s := DefaultQuality(), f.Geometry()
	outer, holes := FaceOuterBoundary(f, q), faceHoleBoundaries(f, q)
	if _, special, _ := specialCurvedMesh(f, s, outer, holes, q, &chartDeclineLog{}); special {
		t.Fatal("a bespoke arm claimed the uncharted tube-wrapping band; the shape has one again")
	}
	m := meshSeamCrossingFace(f, s, outer, holes, q, "", &chartDeclineLog{})
	if area := MeshGeometryProperties(m).Area; !withinChordDeficit(area, wholeTorusArea) {
		t.Errorf("the seam-crossing router meshed %.4f mm², want the whole torus %.4f", area, wholeTorusArea)
	}
	if !carriesDefect(m, CodeTrimIgnoredFullDomain) {
		t.Errorf("the whole-domain fall-back was not reported as a defect: %v", m.Diagnostics)
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

// tubeWrappingBandChart is the face's region in the torus's covering space: u from the first meridian
// to the second, v the whole tube.
func tubeWrappingBandChart() []math.Point2 {
	uHi, twoPi := 3*stdmath.Pi/2, 2*stdmath.Pi
	return []math.Point2{math.P2(0, 0), math.P2(uHi, 0), math.P2(uHi, twoPi), math.P2(0, twoPi)}
}
