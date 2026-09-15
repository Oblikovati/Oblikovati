// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The tube-wrapping torus band has TWO bespoke lofts, and this is the plant for that fact
// (Oblikovati/Oblikovati#3517).
//
// spiricBandMesh, behind the classification arm kindSpiricBand, and torusTubeBandLoftMesh, a rung of
// meshSeamCrossingFace's ordered router, mesh the SAME shape: a torus face bounded by two edges that
// each go the whole way round the tube, bridged by one seam the loop walks twice. Stage 5 of ADR-0061
// put the first in FRONT of the second rather than replacing it, so the classification wins every face
// they both claim and the router rung is shadowed — measured over ./kernel/... and ./model/..., zero
// calls.
//
// Why that has to be written down rather than left to a reader: #3517 asks for kindSpiricBand to be
// DELETED, and deleting it does not absorb the shape into the general pipeline. It hands these faces to
// the other loft. Measured on the two bodies the arm still serves (occtparity simple/J3 and
// bfuseblend/A4, the spiric closed-rim canal hosts): with the arm removed, torusTubeBandLoftMesh claims
// each host torus, and their byte-identity pins move back to exactly the pre-stage-5 values
// (J3 1 115 132 triangles, A4 1 180 684, against 340 988 and 406 540 with the arm). The `recognizers`
// ratchet would fall 12 → 11 for a shape that is still special-cased. ADR-0061's
// "G13 stays open" section carries the full measurement.
//
// So the rows below are the two halves of that fact, on one synthetic band: both recognizers claim it,
// and the classification is what decides which loft gets it.

// tubeWrappingBandFace builds the shape both lofts recognise: a torus band between the meridian circles
// at u=0 and u=3π/2, bridged by the outer-equator arc between them, which the loop walks in both
// directions (the artificial seam a tube-wrapping band cannot avoid).
func tubeWrappingBandFace(t *testing.T, majorR, minorR float64) *topo.Face {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "tubeband", 0))
	tor, err := geom.NewTorusWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), majorR, minorR)
	if err != nil {
		t.Fatal(err)
	}
	return buildTubeWrappingBand(t, tor, lin)
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
	lo := bld.AddEdge(meridianCircle(tor, uLo), vLo, vLo, lin)
	hi := bld.AddEdge(meridianCircle(tor, uHi), vHi, vHi, lin)
	bld.AddFace(tor, lin, topo.OuterLoop(topo.Fwd(lo), topo.Fwd(seam), topo.Rev(hi), topo.Rev(seam)))
	return bld.Build().Faces()[0]
}

// meridianCircle is the torus's tube circle at azimuth u — the boundary that "goes the whole way round
// the tube". Its RefDir is the radial direction, so its own angle-0 point is tor.PointAt(u, 0).
func meridianCircle(tor geom.Torus, u float64) geom.Circle {
	radial := tor.Ref.AsVector().Scale(math.Scalar(stdmath.Cos(u))).
		Add(tor.AxisDir.AsVector().Cross(tor.Ref.AsVector()).Scale(math.Scalar(stdmath.Sin(u))))
	center := tor.Center.TranslateBy(radial.Scale(math.Scalar(tor.MajorRadius)))
	normal, _ := math.UnitVector3FromVector(radial.Cross(tor.AxisDir.AsVector()))
	ref, _ := math.UnitVector3FromVector(radial)
	return geom.Circle{Center: center, Normal: normal, RefDir: ref, Radius: tor.MinorRadius}
}

// TestBothTubeBandLoftsClaimOneBand is the plant: the classification arm's recognizer and the router
// rung's recognizer BOTH accept the same face, so deleting either one leaves the shape special-cased by
// the other. It fails if the two ever stop agreeing — which is what a real absorb would look like.
func TestBothTubeBandLoftsClaimOneBand(t *testing.T) {
	t.Parallel()
	f := tubeWrappingBandFace(t, 20, 5)
	q := DefaultQuality()
	if _, ok := spiricTubeTrimOf(f, f.Geometry(), q); !ok {
		t.Fatal("spiricTubeTrimOf did not claim the tube-wrapping band — the plant covers nothing")
	}
	if _, _, _, ok := tubeBandRingsAndSeam(f, f.Geometry().(geom.Torus), q); !ok {
		t.Error("torusTubeBandLoftMesh's recognizer did not claim the band the classification arm claims; " +
			"if that is now true of every such face, the router rung is dead and #3517 can delete it")
	}
}

// TestTheClassificationDecidesWhichTubeBandLoftRuns pins the shadowing: the classification claims the
// band, so the router rung never sees it. That is why deleting kindSpiricBand RELOCATES rather than
// absorbs — the rung is still there, waiting.
func TestTheClassificationDecidesWhichTubeBandLoftRuns(t *testing.T) {
	t.Parallel()
	f := tubeWrappingBandFace(t, 20, 5)
	q := DefaultQuality()
	got := classifyCurvedTrim(f, f.Geometry(), FaceOuterBoundary(f, q), faceHoleBoundaries(f, q), q)
	if got.kind != kindSpiricBand {
		t.Errorf("the tube-wrapping band classifies as %s, want %s — the router rung's shadow moved",
			got.kind, kindSpiricBand)
	}
}

// TestAChartedTubeBandLeavesTheClassificationButNotTheRouter is why the rung is NOT dead code to be
// deleted alongside the arm: spiricTubeTrimOf gives a CHARTED band to the general chart-driven mesher,
// and if that mesher declines the face, the router rung is the next thing that can claim it. No corpus
// body reaches it today, so the rung is a live seam rather than a proved-unreachable one.
func TestAChartedTubeBandLeavesTheClassificationButNotTheRouter(t *testing.T) {
	t.Parallel()
	f := tubeWrappingBandFace(t, 20, 5)
	f.SetChart([][]math.Point2{{math.P2(0, 0), math.P2(1, 0), math.P2(1, 1), math.P2(0, 1)}})
	q := DefaultQuality()
	if _, ok := spiricTubeTrimOf(f, f.Geometry(), q); ok {
		t.Error("spiricTubeTrimOf claimed a CHARTED band; the chart-driven mesher owns that face")
	}
	if _, _, _, ok := tubeBandRingsAndSeam(f, f.Geometry().(geom.Torus), q); !ok {
		t.Error("the router rung stopped claiming a charted band — then it cannot be the seam a chart " +
			"decline falls to, and its doc comment must say so")
	}
}
