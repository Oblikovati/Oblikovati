// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// simple/Q5's ONE-ENDED far-end split — the second configuration the multi-face trim routes, and the
// case that convicted the gate's "BOTH ends split" clause.
//
// WHAT Q5 IS. A 12000 x 6000 x 6000 block with a radius-3000 cylindrical wall about the vertical line
// x = 6000, y = 0, and a radius-2500 blend on the edge where the y = 0 wall meets the z = 6000 top. The
// blend's band is a radius-2500 cylinder about (y, z) = (2500, 3500) running along x. At its HIGH-x end
// the band simply runs out onto the x = 12000 end plane — a stop plane perpendicular to the band's own
// axis, which trimTerminalSection returns byte-for-byte, so that end never splits. At its LOW-x end it
// stops on the radius-3000 wall, and that wall is TWO faces in the input, meeting along the ruling at
// y = 1500; the band's section crosses from one to the other. So Q5 splits at exactly one end.
//
// WHAT THE CLAUSE COST, MEASURED. `splitEndCount(ef) == 2` declined the whole split, so nine of the
// section's 33 stations were slid onto the small wall face's IMPLICIT cylinder past its own ruling and
// one trim curve was made to bound two faces at once. That shipped:
//
//   - 937 welded free edges at property quality and 160 at default (the corpus's largest leak by far,
//     88 % of all of it) on a case the scoreboard scores PASS;
//   - two self-crossing developed face loops, the larger pinching 84912.4 off the small wall face;
//   - the small wall face 5.85 % SHORT and the top plane and the big wall face each ~0.15-0.18 % LONG —
//     a compensating trio that kept the whole-body number at −0.0918 % and the case green.
//
// The split closes all three: 0 free edges at both qualities, 0 self-crossing loops, and all four
// rebuilt faces on their own closed forms (below), the body at −0.000194 % of DRAWEXE's 3.46388e8.
//
// ★ THE CLOSED FORMS ARE DERIVED FROM THE SOLID, NOT FROM THE ORACLE. Each integrand below is "where
// does this face's material actually stop", written from Q5's own dimensions; their sum plus the five
// exact planar faces is 346388024.88, which is the corpus record's own expectedArea 346388000 to
// 7.2e-8. That is what makes them a check on DRAWEXE rather than a restatement of it.
const (
	q5WallCx     = 6000.0  // the radius-3000 host wall's axis x
	q5WallR      = 3000.0  // the host wall's radius
	q5BandR      = 2500.0  // the fillet radius
	q5BandAxisY  = 2500.0  // the band's axis y — r in from the y = 0 host plane
	q5BandAxisZ  = 3500.0  // the band's axis z — r in from the z = 6000 host plane
	q5TopZ       = 6000.0  // the filleted edge's own plane, and the block's top
	q5BlockY     = 6000.0  // the block's y extent
	q5EndX       = 12000.0 // the end plane the band runs out to (its non-splitting end)
	q5SeamY      = 1500.0  // y of the ruling where the input splits the host wall into two faces
	q5PerFaceTol = 3e-4    // mesh quantization only; the defect gated against is −5.85 % / +0.18 %
)

// Face lineage keys on the shipped body.
const (
	q5NeighbourKey = "import:step#16:face#2" // the big wall piece the section crosses ONTO
	q5TopPlaneKey  = "import:step#16:face#4" // the z = 6000 host plane (the B host)
	q5StopFaceKey  = "import:step#16:face#5" // the small wall piece the section stops on
	q5BandKey      = "import:step#16:edge#15/fillet:cyl#0"
)

// TestQ5FarEndSplitIsAtomicAndHitsItsClosedForms is the one-ended split's guard: the split is applied to
// the band AND to every face its chain touches, or to none of them, and each rebuilt face lands on the
// area its own material actually covers.
func TestQ5FarEndSplitIsAtomicAndHitsItsClosedForms(t *testing.T) {
	t.Parallel()
	body := gridCaseBody(t, corpusRecord(t, "simple", "Q5"))
	assertQ5SurfacesMatchTheConstants(t, body)
	assertQ5JunctionIsSharedByAllThreeFaces(t, body)
	for _, tc := range []struct {
		lineage string
		want    float64
	}{
		{q5StopFaceKey, q5StopFaceArea()},
		{q5NeighbourKey, q5NeighbourArea()},
		{q5TopPlaneKey, q5TopPlaneArea()},
		{q5BandKey, q5BandArea()},
	} {
		got := ops.MeshArea(shippedFaceMesh(t, body, faceByLineage(t, body, tc.lineage)))
		if stdmath.Abs(got-tc.want)/tc.want > q5PerFaceTol {
			t.Errorf("simple/Q5 %s meshes %.10g, closed form %.10g (rel %+.5f%%, budget %.1g)",
				tc.lineage, got, tc.want, (got-tc.want)/tc.want*100, q5PerFaceTol)
		}
	}
	if bad := ops.SelfCrossingFaceLoops(body, ops.PropertyQuality()); len(bad) != 0 {
		t.Errorf("simple/Q5 self-crosses on %d face loop(s): %s", len(bad), describeSelfCrossing(bad))
	}
	assertQ5MeshIsWatertightAtEveryQuality(t, body)
}

// assertQ5MeshIsWatertightAtEveryQuality is what the un-split trim actually cost downstream: Q5's welded
// body mesh leaked 937 free edges at property quality and 160 at default — 88 % of the whole corpus's
// leakage — on a body every existing gate called healthy. It is now 0 at both, and folds were and stay 0.
//
// It measures through shippedMeshFreeEdges, the corpus-wide ratchet's own ruler
// (welded_mesh_leak_test.go), so Q5's required zero and that table's ceilings are read by ONE production
// function. Q5 is deliberately ABSENT from knownMeshLeaks now: its ceiling is this zero.
func assertQ5MeshIsWatertightAtEveryQuality(t *testing.T, body *topo.Body) {
	t.Helper()
	for _, gq := range gateQualities() {
		folds := 0
		for _, m := range ops.CalculateBodyFacets(body, gq.q).FaceMeshes {
			folds += ops.FoldEdgeCount(m)
		}
		if free := shippedMeshFreeEdges(body, gq.q); folds != 0 || free != 0 {
			t.Errorf("simple/Q5 body mesh at %s quality: %d fold edges, %d free edges; want 0 and 0",
				gq.name, folds, free)
		}
	}
}

// q5RimAngleAtSeam is the wall-rim angle of the ruling the input splits the host wall along, measured so
// that y = R·sin θ (θ = 0 at the wall's own x-extreme, x = q5WallCx + R). It is π/6.
func q5RimAngleAtSeam() float64 { return stdmath.Asin(q5SeamY / q5WallR) }

// q5RimAngleAtBandTangent is the rim angle where the band's tangent line on the top plane, y = 2500,
// meets the wall — i.e. where the band's terminal section reaches the top plane and the bite ends.
func q5RimAngleAtBandTangent() float64 { return stdmath.Asin(q5BandAxisY / q5WallR) }

// q5WallBiteTop is the height at which the band's own cylinder cuts the host wall at rim angle θ: the
// wall point is at y = R·sin θ, and the band's surface there stands at z = zAxis + √(r² − (y − yAxis)²).
// At θ = 0 that is exactly the band's own axis height 3500, and at the seam it is 3500 + √5250000.
func q5WallBiteTop(th float64) float64 {
	dy := q5WallR*stdmath.Sin(th) - q5BandAxisY
	return q5BandAxisZ + stdmath.Sqrt(stdmath.Max(q5BandR*q5BandR-dy*dy, 0))
}

// q5StopFaceArea is the SMALL wall piece — the face the band's terminal section stops on — from the
// block's underside up to the band∩wall curve, over its own quarter of rim from θ = 0 to the seam:
// R·∫₀^{π/6} zTop(θ) dθ = 8121170.18. The un-split trim shipped 7645850.16, −5.85 %.
func q5StopFaceArea() float64 {
	return q5WallR * simpson(q5WallBiteTop, 0, q5RimAngleAtSeam())
}

// q5NeighbourArea is the BIG wall piece the section crosses onto: bitten by the band from the seam to
// where the band's tangent line reaches the rim (θc = asin(2500/3000)), and full height z = 6000 from
// there round to the wall's far end at θ = π. R·[∫ zTop + (π − θc)·6000] = 47038978.00.
func q5NeighbourArea() float64 {
	thc := q5RimAngleAtBandTangent()
	return q5WallR * (simpson(q5WallBiteTop, q5RimAngleAtSeam(), thc) + (stdmath.Pi-thc)*q5TopZ)
}

// q5BandArea is the fillet band itself. Parameterise its section by φ ∈ [0, π/2], φ = 0 at the tangent
// line on the y = 0 host and φ = π/2 at the tangent line on the top, so the ruling at φ sits at
// y(φ) = 2500(1 − cos φ). That ruling runs from the end plane x = 12000 back to the host wall, which at
// that y stands at x = 6000 + √(3000² − y²): r·∫₀^{π/2} (12000 − 6000 − √(9e6 − y²)) dφ = 12837581.35.
//
// ★ The band is NOT the face the split repairs — it measures −0.00082 % of this form before and after.
// It is pinned here because it is the face an inherited attribution named as the 5.85 %-short one (see
// q5_perface_oracle_test.go), and a closed form is how that was settled.
func q5BandArea() float64 {
	return q5BandR * simpson(func(ph float64) float64 {
		y := q5BandAxisY * (1 - stdmath.Cos(ph))
		return q5EndX - q5WallCx - stdmath.Sqrt(stdmath.Max(q5WallR*q5WallR-y*y, 0))
	}, 0, stdmath.Pi/2)
}

// q5TopPlaneArea is the z = 6000 host plane, by Green's theorem (½∮(x dy − y dx)) on its own boundary:
// the end plane's edge at x = 12000, the block's far edge at y = 6000, the wall's rim arc from θ = π
// round to θc, and the band's tangent line at y = 2500. The contour integral is the honest form rather
// than a y-integral because the rim arc DOUBLES BACK in y — it reaches y = 3000 at its apex while the
// face's own tangent line sits at y = 2500 — so the region is not the area under a graph.
// = 49368722.08. The un-split trim shipped 49441243.96, +0.147 %.
func q5TopPlaneArea() float64 {
	thc := q5RimAngleAtBandTangent()
	arcEndX := q5WallCx + q5WallR*stdmath.Cos(thc)
	endPlaneEdge := 0.5 * q5EndX * (q5BlockY - q5BandAxisY)
	farEdge := 0.5 * q5BlockY * q5EndX
	// ∮ over the rim arc, θ from π down to θc, with x = cx + R cos θ and y = R sin θ:
	// x dy − y dx = (cx·R·cos θ + R² cos²θ + R² sin²θ) dθ = (cx·R·cos θ + R²) dθ.
	rimArc := 0.5 * (q5WallCx*q5WallR*stdmath.Sin(thc) + q5WallR*q5WallR*(thc-stdmath.Pi))
	tangentLine := -0.5 * q5BandAxisY * (q5EndX - arcEndX)
	return endPlaneEdge + farEdge + rimArc + tangentLine
}

// assertQ5SurfacesMatchTheConstants keeps the closed forms tied to the body they describe: if the fixture
// or the importer ever moves, they fail loud instead of silently measuring nothing.
func assertQ5SurfacesMatchTheConstants(t *testing.T, body *topo.Body) {
	t.Helper()
	wall := faceByLineage(t, body, q5StopFaceKey).Geometry().(geom.Cylinder)
	band := faceByLineage(t, body, q5BandKey).Geometry().(geom.Cylinder)
	for _, c := range []struct {
		name      string
		got, want float64
	}{
		{"host wall radius", wall.Radius, q5WallR},
		{"host wall axis x", float64(wall.Origin.X), q5WallCx},
		{"band radius", band.Radius, q5BandR},
		{"band axis y", float64(band.Origin.Y), q5BandAxisY},
		{"band axis z", float64(band.Origin.Z), q5BandAxisZ},
	} {
		if stdmath.Abs(c.got-c.want) > 1e-9 {
			t.Fatalf("simple/Q5 %s is %.12g, the closed forms assume %.12g", c.name, c.got, c.want)
		}
	}
}

// assertQ5JunctionIsSharedByAllThreeFaces is the ATOMICITY statement. The terminal trim crosses from the
// small wall piece to the big one at the triple point band ∩ wall ∩ seam-ruling — analytically
// (6000 + 1500√3, 1500, 3500 + √5250000). All three faces that meet there — both wall pieces and the
// band — must carry it as a boundary vertex; a split adopted by the hosts but not by the band (or the
// reverse) leaves the band's cap running somewhere else entirely, and the point is on two faces or none.
func assertQ5JunctionIsSharedByAllThreeFaces(t *testing.T, body *topo.Body) {
	t.Helper()
	th := q5RimAngleAtSeam()
	j := math.P3(
		math.Scalar(q5WallCx+q5WallR*stdmath.Cos(th)),
		math.Scalar(q5SeamY),
		math.Scalar(q5WallBiteTop(th)),
	)
	for _, lineage := range []string{q5StopFaceKey, q5NeighbourKey, q5BandKey} {
		if !faceHasVertexAt(faceByLineage(t, body, lineage), j) {
			t.Errorf("simple/Q5: face %s does not meet the wall-piece hand-off at %v — the far-end split "+
				"was applied to some of the faces sharing that trim but not to all of them", lineage, j)
		}
	}
}
