// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The SECOND self-touching boundary (Oblikovati/Oblikovati#3519).
//
// A torus cut by the plane tangent to its inner equator splits at a point, and each piece keeps a torus
// face whose boundary passes through that point twice — the one boundary shape a band mesher cannot
// treat as two rims. Exactly one such body was in the corpus: R=5, r=2 (torus_figure_eight_band_test.go).
// One is not a family, and the corner angle at the pinch is set by the ASPECT RATIO: the two branches
// leave the pinch with dv/du = ±√((R−r)/r), so the material corner is 180° − 2·arctan√(r/(R−r)) —
// 101.5° at R=5 r=2, and exactly 90° here, because R − r = r makes the arctan 45°.
//
// This row is R=5, r=2.5. It is the second self-touching boundary, and it is the row that found #3551
// and #3553 — both of which turned out to be the reason the boundary clearance appeared to have a lower
// failure edge at all. Its intersect piece used to tear at DefaultQuality for every swept k ≤ 0.70 and
// its cut piece was the one aspect ratio whose analytic volume the integrator would answer for. Neither
// was a tolerance: the covering laid two vertices at one location at the pinch, and the side test probed
// a chart's slit. With both fixed, every swept k from 0.1 up meshes the whole corpus watertight and the
// clearance is set by its cost instead (chart_face_clearance.go).
//
// It is also the row that FOUND Oblikovati/Oblikovati#3551 and now guards the fix. Its intersect piece
// used to be declined at PropertyQuality and to ship the whole torus, 408 free edges; see
// TestTheSquarePinchIntersectPieceIsWatertightAtBothQualities for what the cause turned out to be.
const (
	squarePinchRingRadius = 5.0  // R
	squarePinchTubeRadius = 2.5  // r; R − r = r puts the pinch corner at exactly 90°
	squarePinchCutY       = 2.5  // = R − r, the plane tangent to the inner equator
	squarePinchCorner     = 90.0 // degrees, against 101.5° at R=5 r=2
	squarePinchTorusArea  = 4 * stdmath.Pi * stdmath.Pi * squarePinchRingRadius * squarePinchTubeRadius
	squarePinchTorusVol   = 2 * stdmath.Pi * stdmath.Pi * squarePinchRingRadius *
		squarePinchTubeRadius * squarePinchTubeRadius
)

// squarePinch*{Area,Volume} are the analytic shares of the two pieces, from the same quadrature the
// R=5 r=2 row uses (torusTangentPlaneAbove{Area,Volume}); TestTheSquarePinchOraclesAreConverged
// recomputes them every run so neither literal can be a transcription.
const (
	squarePinchAboveArea   = 159.451671516 // the y > 2.5 piece's torus face
	squarePinchBelowArea   = 334.028548538 // its complement; the two sum to the torus
	squarePinchAboveVolume = 203.626154953
	squarePinchBelowVolume = 413.224120115
)

// squarePinchPiece builds one side of the square-cornered figure-eight split.
func squarePinchPiece(t *testing.T, op ops.PartFeatureOperation) *topo.Body {
	t.Helper()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), squarePinchRingRadius, squarePinchTubeRadius, "ring")
	if err != nil {
		t.Fatalf("torus R=%g r=%g: %v", squarePinchRingRadius, squarePinchTubeRadius, err)
	}
	box, err := brep.SolidBlock(math.P3(-20, squarePinchCutY, -20), math.P3(20, 20, 20), "box")
	if err != nil {
		t.Fatalf("half-space box: %v", err)
	}
	body, err := ops.Boolean(op, ring, box)
	if err != nil {
		t.Fatalf("square-pinch %v: %v", op, err)
	}
	return body
}

// TestTheSquarePinchOraclesAreConverged is the oracle's receipt for this aspect ratio: the quadrature
// has converged and lands on the four literals, and each pair partitions the whole torus.
func TestTheSquarePinchOraclesAreConverged(t *testing.T) {
	t.Parallel()
	area := torusTangentPlaneAboveArea(squarePinchRingRadius, squarePinchTubeRadius, torusTangentPlanePanels)
	vol := torusTangentPlaneAboveVolume(squarePinchRingRadius, squarePinchTubeRadius, torusTangentPlanePanels)
	assertOracleMatchesItsLiteral(t, "square-pinch area", area, squarePinchAboveArea)
	assertOracleMatchesItsLiteral(t, "square-pinch volume", vol, squarePinchAboveVolume)
	assertOracleMatchesItsLiteral(t, "square-pinch area partition", squarePinchAboveArea+squarePinchBelowArea,
		squarePinchTorusArea)
	assertOracleMatchesItsLiteral(t, "square-pinch volume partition", squarePinchAboveVolume+squarePinchBelowVolume,
		squarePinchTorusVol)
}

// assertOracleMatchesItsLiteral fails unless got and want agree to eleven digits — the band that
// separates "the same number" from "a mistyped digit" at the precision these literals are written to.
func assertOracleMatchesItsLiteral(t *testing.T, what string, got, want float64) {
	t.Helper()
	if rel := stdmath.Abs(got-want) / want; rel > 1e-11 {
		t.Errorf("%s: %.9f against %.9f (rel %.3e); the quadrature and its literal must agree", what, got, want, rel)
	}
}

// TestTheSquarePinchBoundaryTouchesItself is the FIXTURE's own assertion, and it is what makes this a
// second self-touching boundary rather than a second torus. Every vertex of each piece's torus face
// sits at the tangency point, and the face is bounded by exactly two edges — so its whole boundary
// meets itself there and nowhere else. The intersect piece carries ONE loop through the point twice;
// the cut piece carries two lobes that touch at it.
//
// Without this row the shape could quietly become an ordinary two-rim band — a moved plane, a different
// radius — and the mesh gates below would still pass while testing nothing they were written for.
func TestTheSquarePinchBoundaryTouchesItself(t *testing.T) {
	t.Parallel()
	pinch := math.P3(0, squarePinchCutY, 0)
	for _, row := range []struct {
		op    ops.PartFeatureOperation
		loops int
	}{{ops.Cut, 2}, {ops.Intersect, 1}} {
		f := squarePinchTorusFace(t, squarePinchPiece(t, row.op))
		if n := len(f.Loops()); n != row.loops {
			t.Errorf("%v piece: the torus face has %d boundary loop(s), want %d", row.op, n, row.loops)
		}
		if n := len(f.Edges()); n != 2 {
			t.Errorf("%v piece: the torus face has %d boundary edge(s), want 2 (the two spiric branches)", row.op, n)
		}
		assertEveryVertexIsAtThePinch(t, f, pinch, row.op)
	}
}

// assertEveryVertexIsAtThePinch fails unless every boundary vertex of the face is the tangency point.
func assertEveryVertexIsAtThePinch(t *testing.T, f *topo.Face, pinch math.Point3, op ops.PartFeatureOperation) {
	t.Helper()
	vs := f.Vertices()
	if len(vs) == 0 {
		t.Fatalf("%v piece: the torus face has no boundary vertex; the self-touch assertion covers nothing", op)
	}
	for _, v := range vs {
		if d := float64(v.Point().DistanceTo(pinch)); d > 1e-9 {
			t.Errorf("%v piece: a torus-face vertex sits %.3e from the tangency %v; the boundary no longer "+
				"touches itself only there", op, d, pinch)
		}
	}
}

// squarePinchTorusFace is a piece's single torus face.
func squarePinchTorusFace(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	var found *topo.Face
	for _, f := range b.Faces() {
		if _, isTorus := f.Geometry().(geom.Torus); !isTorus {
			continue
		}
		if found != nil {
			t.Fatal("the square-pinch piece has more than one torus face; the per-face gates sum the wrong thing")
		}
		found = f
	}
	if found == nil {
		t.Fatal("the square-pinch piece has no torus face")
	}
	return found
}

// TestTheSquarePinchPiecesMeshTheirOwnShare is the per-face and per-body gate at the faceting where the
// chart mesher serves BOTH pieces. Each piece's torus face meshes its analytic share less a chord
// deficit, each body's volume does the same, and the two shares partition the torus — the assertion a
// mesher that covers the pinch twice fails even when each piece looks plausible on its own.
func TestTheSquarePinchPiecesMeshTheirOwnShare(t *testing.T) {
	t.Parallel()
	q := ops.DefaultQuality()
	below, above := squarePinchPiece(t, ops.Cut), squarePinchPiece(t, ops.Intersect)
	belowArea := squarePinchFaceArea(t, below, q)
	aboveArea := squarePinchFaceArea(t, above, q)
	assertChordDeficit(t, "square-pinch below face", belowArea, squarePinchBelowArea)
	assertChordDeficit(t, "square-pinch above face", aboveArea, squarePinchAboveArea)
	assertChordDeficit(t, "square-pinch face sum", belowArea+aboveArea, squarePinchTorusArea)
	assertChordDeficitWithin(t, "square-pinch below volume", squarePinchMeshVolume(t, below, q),
		squarePinchBelowVolume, chordDeficitVolumeCoarse)
	assertChordDeficitWithin(t, "square-pinch above volume", squarePinchMeshVolume(t, above, q),
		squarePinchAboveVolume, chordDeficitVolumeCoarse)
}

// squarePinchFaceArea is a piece's torus-face mesh area at one faceting.
func squarePinchFaceArea(t *testing.T, b *topo.Body, q ops.Quality) float64 {
	t.Helper()
	f := squarePinchTorusFace(t, b)
	return tessellate.MeshGeometryProperties(tessellate.TessellateFace(f, q)).Area
}

// squarePinchMeshVolume is a piece's whole-body mesh volume at one faceting.
func squarePinchMeshVolume(t *testing.T, b *topo.Body, q ops.Quality) float64 {
	t.Helper()
	mesh, _ := tessellate.TessellateBody(b, q)
	return tessellate.MeshGeometryProperties(mesh).Volume
}

// TestTheSquarePinchCutPieceIsWatertightAtBothQualities is the half of the row that is GREEN, and it is
// the half that pins the boundary clearance. Measured across the sweep in chart_face_clearance.go, the
// intersect piece is the one that moves with k; the cut piece holds everywhere, and holding at both
// facetings is what says the pinch itself is meshed rather than merely survived at one sampling.
func TestTheSquarePinchCutPieceIsWatertightAtBothQualities(t *testing.T) {
	t.Parallel()
	t.Run("default", func(t *testing.T) {
		t.Parallel()
		assertSquarePinchCutPieceHolds(t, "default", ops.DefaultQuality())
	})
	t.Run("property", func(t *testing.T) {
		t.Parallel()
		assertSquarePinchCutPieceHolds(t, "property", ops.PropertyQuality())
	})
}

// assertSquarePinchCutPieceHolds is one faceting's half of that row. The two facetings are subtests
// rather than a loop so they run in PARALLEL: the row is the slowest unguarded test in its package and
// its elapsed is load-sensitive (22 s quiet, 37 s under a load average of 20, against a 60 s budget),
// and splitting the two booleans across goroutines is the cheapest headroom available without giving
// up either faceting (review 2, M2).
func assertSquarePinchCutPieceHolds(t *testing.T, name string, q ops.Quality) {
	t.Helper()
	piece := squarePinchPiece(t, ops.Cut)
	mesh, _ := tessellate.TessellateBody(piece, q)
	if free := tessellate.FreeEdgeCount(mesh); free != 0 {
		t.Errorf("%s quality: the square-pinch cut piece meshes with %d free edges, want 0", name, free)
	}
	assertChordDeficit(t, name+" square-pinch cut face", squarePinchFaceArea(t, piece, q), squarePinchBelowArea)
}

// squarePinchIntersectPropertyFreeEdges and squarePinchIntersectPropertyArea are the MEASURED defect at
// the fine faceting, pinned two-sided so it can move in neither direction unnoticed.
//
// The intersect piece's torus face is declined by the chart-driven mesher at PropertyQuality — "the mesh
// it built is not bounded by its own rim: 2 unpaired edge(s) are no rim segment and 1 rim segment(s) it
// does not bound" — and ships over the surface's WHOLE domain (493.437 mm² against the 159.452 it owns,
// a chord deficit under the whole torus's 493.480), leaving the body with 408 free edges.
//
// It is not the boundary clearance's doing: measured at every k from 0.1 to 1.5 the same 408 stands, and
// it clears only at k = 3.0, where other rows tear instead. It is
// not this slice's doing either: it reproduces with the deleted incidence retry restored. The same
// decline hits R=5 r=1.5 (352 free edges) and R=10 r=3 (372) and NOT R=5 r=2, which is why one
// aspect ratio in the corpus was enough to hide it.
// TestTheSquarePinchIntersectPieceIsWatertightAtBothQualities is the half of the row that used to be
// RED, and what closes Oblikovati/Oblikovati#3551.
//
// The intersect piece's torus face was DECLINED at PropertyQuality and shipped over the surface's
// whole domain — 493.437 mm² against the 159.452 it owns — leaving the body with 408 free edges. The
// same decline hit R=5 r=1.5 (352) and R=10 r=3 (372) and not R=5 r=2, which is why one aspect ratio
// in the corpus hid it, and 352 + 408 + 372 = 1132 was the whole corpus's free-edge total.
//
// The cause was not this mesher's classification and not any constant of it. The covering laid THREE
// vertices at one location at the pinch — the single loop passes it twice, and on a boundary that also
// wraps a whole period the shift carrying the far pass back lands a third copy on the same spot — and
// a constrained triangulation cannot recover a constraint edge incident to a vertex another vertex
// sits on: every orientation predicate the recovery walks with reads the two as one point. One rim
// segment was therefore never an edge of the triangulation, the mesh detoured round it through two
// interior nodes, and the rim gate refused the face. The covering merges coincident locations now
// (coverVertices.MergeCoincidentLocations) and the face is bounded by exactly its own rim.
//
// So this row asserts what the cut piece's does, at both facetings — and the count it asserts is ZERO,
// which is the direction that cannot be reached by accident.
func TestTheSquarePinchIntersectPieceIsWatertightAtBothQualities(t *testing.T) {
	t.Parallel()
	piece := squarePinchPiece(t, ops.Intersect)
	for _, gq := range figureEightQualities() {
		mesh, _ := tessellate.TessellateBody(piece, gq.q)
		if free := tessellate.FreeEdgeCount(mesh); free != 0 {
			t.Errorf("%s quality: the square-pinch intersect piece meshes with %d free edges, want 0 (#3551)",
				gq.name, free)
		}
		if meshReportsIgnoredTrim(mesh) {
			t.Errorf("%s quality: the square-pinch intersect piece reports a discarded trim; its torus face "+
				"is the chart mesher's and the chart mesher must bound it by its own rim: %v", gq.name, mesh.Diagnostics)
		}
		assertChordDeficit(t, gq.name+" square-pinch intersect face",
			squarePinchFaceArea(t, piece, gq.q), squarePinchAboveArea)
	}
}

// TestTheAnalyticIntegratorAgreesWithTheSquarePinchOracle is the two-way cross-check this aspect ratio
// makes available and the R=5 r=2 pair does not: the kernel's analytic B-rep integrator answers for BOTH
// square-pinch pieces, and its answers are this file's two volume literals to twelve digits —
// 413.224120115 and 203.626154953 — reached through a code path that shares no step with the quadrature.
//
// It is a cross-check, not a gate: an oracle no more exact than the number it checks proves nothing, and
// these two agree to 1e-12. What it does prove is that the quadrature and the integrator are not both
// wrong the same way, which no single derivation can.
func TestTheAnalyticIntegratorAgreesWithTheSquarePinchOracle(t *testing.T) {
	t.Parallel()
	answered := 0
	for _, row := range []struct {
		op   ops.PartFeatureOperation
		want float64
	}{{ops.Cut, squarePinchBelowVolume}, {ops.Intersect, squarePinchAboveVolume}} {
		an, ok := query.AnalyticGeometryProperties(squarePinchPiece(t, row.op))
		if !ok {
			t.Errorf("%v piece: the analytic integrator declines a body it used to answer for; the "+
				"cross-check above now covers less than it says", row.op)
			continue
		}
		answered++
		if rel := stdmath.Abs(an.Volume-row.want) / row.want; rel > 1e-9 {
			t.Errorf("%v piece: the analytic integrator reads %.9f mm³ against the quadrature's %.9f (rel %.3e)",
				row.op, an.Volume, row.want, rel)
		}
	}
	if answered != 2 {
		t.Errorf("the analytic integrator answered for %d of the two square-pinch pieces, want both", answered)
	}
}

// TestTheSquarePinchCornerIsARightAngle keeps the comment's 90° claim honest: it is not a measurement of the mesh, it
// is the closed form 180° − 2·arctan√(r/(R−r)) the doc derives, checked against the constant written
// there so a changed radius cannot leave the prose behind.
func TestTheSquarePinchCornerIsARightAngle(t *testing.T) {
	t.Parallel()
	half := stdmath.Atan(stdmath.Sqrt(squarePinchTubeRadius / (squarePinchRingRadius - squarePinchTubeRadius)))
	corner := 180 - 2*half*180/stdmath.Pi
	if stdmath.Abs(corner-squarePinchCorner) > 1e-9 {
		t.Errorf("the pinch corner is %.4f°, not the %g° this fixture is named for", corner, squarePinchCorner)
	}
}
