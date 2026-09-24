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

// The tube-wrapping torus band has ONE mesher, and it is the general one (#3517).
//
// It had three. spiricBandMesh, behind the classification arm kindSpiricBand, and
// torusTubeBandLoftMesh, a rung of meshSeamCrossingFace's ordered router, meshed the SAME shape: a
// torus face bounded by two edges that each go the whole way round the tube, bridged by one seam the
// loop walks twice. ADR-0061 stage 5 put the first in FRONT of the second rather than replacing it,
// and the second then built nothing — 0 builds over ./kernel/... and ./model/... — so #3517 deleted it
// under the delete-first rule, and then deleted the arm itself.
//
// What made the arm deletable is that its two real hosts now record a chart. The arm existed for the
// UNCHARTED band: occtparity simple/J3 and bfuseblend/A4 are a STEP import plus a fillet, and neither
// the importer nor the rim rebuild wrote a chart, so the general mesher declined them outright. #3550
// put one on the importer, and the rim rebuild now carries it onto the face it re-winds (carryChart,
// fillet_rim_build.go). The covering that consumes it triangulates only its boundary band
// (chart_structured_interior.go), so the general path meshes those hosts at the faceting the per-face
// oracle reads: 292 959.152 mm² against DRAWEXE 292 961 where the arm read 292 950.19 — six times
// closer, zero diagnostics (ADR-0061).
//
// The fixture's far rim is a fitted BSpline rather than a second circle — see meridianRail. With two
// circles an earlier rung of the same router claims the face and row two would be measuring the wrong
// mesher.
//
// The three rows are the three answers the band can now get: a CHARTED band is the chart mesher's; an
// UNCHARTED one has no special mesher left and falls to the surface's whole domain, loudly; and
// nothing in between claims it.

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

// TestNoSpecialMesherClaimsTheTubeWrappingBand is the classification half: with the spiric arm gone,
// an UNCHARTED tube-wrapping band is claimed by no arm at all, and a CHARTED one is the chart
// mesher's. A regression that re-adds a shape-specific recognizer for this band fails here.
func TestNoSpecialMesherClaimsTheTubeWrappingBand(t *testing.T) {
	t.Parallel()
	f := tubeWrappingBandFace(t, 20, 5)
	q, s := DefaultQuality(), f.Geometry()
	outer, holes := FaceOuterBoundary(f, q), faceHoleBoundaries(f, q)
	if got := classifyCurvedTrim(f, s, outer, holes, q); got.kind != kindUncharted {
		t.Errorf("the uncharted tube-wrapping band classifies as %s, want %s", got.kind, kindUncharted)
	}
	f.SetChart([][]math.Point2{tubeWrappingBandChart()})
	if got := classifyCurvedTrim(f, s, outer, holes, q); got.kind != kindChart {
		t.Errorf("the charted tube-wrapping band classifies as %s, want %s", got.kind, kindChart)
	}
}

// TestAnUnchartedBandFallsToTheReportedWholeDomain is the guard the deletion leaves in its place, and
// it is the reason the deletion is worth making rather than a quiet handover.
//
// While torusTubeBandLoftMesh stood behind the classification, removing the arm would have handed the
// band to a second loft and the only thing that noticed would have been two occtparity pins moving.
// The answer now is the surface's WHOLE domain with the degradation REPORTED — the loud corpus failure
// the gate exists for: 3947.84 mm² against the band's 2960.88, one diag.Defect.
func TestAnUnchartedBandFallsToTheReportedWholeDomain(t *testing.T) {
	t.Parallel()
	f := tubeWrappingBandFace(t, 20, 5)
	q, s := DefaultQuality(), f.Geometry()
	outer, holes := FaceOuterBoundary(f, q), faceHoleBoundaries(f, q)
	if _, special, _ := specialCurvedMesh(f, s, outer, holes, q, &chartDeclineLog{}); special {
		t.Fatal("a special mesher claimed the uncharted band; the spiric arm is back")
	}
	behind := meshSeamCrossingFace(f, s, outer, holes, q, "", &chartDeclineLog{})
	if got := MeshGeometryProperties(behind).Area; !withinChordDeficit(got, wholeTorusArea) {
		t.Errorf("the uncharted band meshed %.4f mm², want the whole torus %.4f — a second loft for "+
			"this shape has re-appeared in meshSeamCrossingFace", got, wholeTorusArea)
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

// TestAChartedTubeWrappingBandGoesToTheChartMesher is the row the whole deletion rests on: a band that
// records its own region is meshed by the GENERAL chart-driven mesher, at its own area. The chart is
// the face's real rectangle (u ∈ [0, 3π/2] × v ∈ [0, 2π]), not a placeholder, so the row exercises the
// mesher rather than a "is there a chart" early return.
func TestAChartedTubeWrappingBandGoesToTheChartMesher(t *testing.T) {
	t.Parallel()
	f := tubeWrappingBandFace(t, 20, 5)
	f.SetChart([][]math.Point2{tubeWrappingBandChart()})
	q, s := DefaultQuality(), f.Geometry()
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
