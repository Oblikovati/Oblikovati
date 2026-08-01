// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// farEndVertexTol is the budget for "the band's far end sits where OCCT puts it", relative to the body's
// bounding diagonal (scale-invariant, ADR-0042). The trim solves the band∩wall crossing in closed form and
// Newton-polishes it, so every pinned vertex lands within ~3e-16 relative — this 1e-9 leaves ~6 decades of
// headroom while still being 6 decades TIGHTER than the run-on it replaces (0.0076–0.072 relative).
const farEndVertexTol = 1e-9

// farEndOnWallTol is the budget for "every boundary edge of these bodies lies on the face it bounds",
// relative to the bounding diagonal. It is the same 1e-6 the corpus-wide ratchet uses, and these cases hold
// it with NO debt entry: measured worst 5.3e-7 (C7's cubic interpolation of its cone trim curve), the rest
// ≤2.6e-9. Squaring the far end off at the flank's parametric extreme instead puts it at 0.0076 (B5) to
// 0.072 (C7) — four to five decades outside.
const farEndOnWallTol = 1e-6

// farEndWallCase is one case whose fillet band runs into a CURVED host wall, with the vertices DRAWEXE
// 8.0.0 reports there (wantVerts) and the ones the un-trimmed construction shipped instead (bannedVerts —
// the flank's parametric extreme). Coordinates are transcribed from `explode result V` + `dump`.
type farEndWallCase struct {
	name        string
	wantVerts   []math.Point3
	bannedVerts []math.Point3
	wall        string // the stop face's surface family, for the failure message
}

// TestFilletBandFarEndLandsOnTheWall is the oracle gate for the far-end wall trim (fillet_farend_trim.go).
//
// A fillet band must END where the solid does. Before this trim the terminal cross-section was squared off
// in the plane through the filleted edge's end VERTEX perpendicular to the edge axis, which is only correct
// when the stop face is a plane with the axis for its normal; against a curved wall the band ran ON past the
// wall and the wall's own loop carried a curve that was not on it. These six cases are the corpus's
// cylinder / cone / sphere walls, and every far-end vertex below is DRAWEXE's own — e.g. B5's cap band ends
// at x = √(50²−10²) = 48.9897948556636, not at the flank extreme x = 50.
func TestFilletBandFarEndLandsOnTheWall(t *testing.T) {
	t.Parallel()
	for _, tc := range farEndWallCases() {
		t.Run(tc.name, func(t *testing.T) {
			body := pinnedBody(t, "simple", tc.name)
			diag := boundingDiag(body)
			for _, w := range tc.wantVerts {
				assertBodyHasVertex(t, body, w, diag, tc)
			}
			for _, b := range tc.bannedVerts {
				assertBodyLacksVertex(t, body, b, diag, tc)
			}
			assertLoopSegmentsOnFaces(t, Record{Grid: "simple", Case: tc.name}, body, farEndOnWallTol)
		})
	}
}

// farEndWallCases is the pinned population: the cylinder (B5/B1), cone (C4/B9/C7) and sphere (D7) walls a
// planar-edge band stops against in the corpus. C7 is the extreme case — a `pcone s 50 0 120 90` APEX cone
// whose single band ran all the way to the apex at z=120, shipping +17.67% body area; OCCT ends it at z=96,
// where the cone's own radius has shrunk to the fillet radius.
func farEndWallCases() []farEndWallCase {
	return []farEndWallCase{
		{name: "B5", wall: "Cylinder r=50",
			wantVerts:   []math.Point3{math.P3(48.9897948556636, 10, 100), math.P3(-10, -48.9897948556636, 100)},
			bannedVerts: []math.Point3{math.P3(50, 10, 100), math.P3(-10, -50, 100)}},
		{name: "B1", wall: "Cylinder r=50",
			wantVerts:   []math.Point3{math.P3(-10, 48.9897948556636, 100), math.P3(-48.9897948556636, 10, 100)},
			bannedVerts: []math.Point3{math.P3(-10, 50, 100), math.P3(-50, 10, 100)}},
		{name: "C4", wall: "Cone 90→40",
			wantVerts:   []math.Point3{math.P3(38.7298334620742, 10, 150), math.P3(-10, -38.7298334620741, 150)},
			bannedVerts: []math.Point3{math.P3(40, 10, 150), math.P3(-10, -40, 150)}},
		{name: "B9", wall: "Cone 90→40",
			wantVerts:   []math.Point3{math.P3(-10, 38.7298334620742, 150), math.P3(-38.7298334620742, 10, 150)},
			bannedVerts: []math.Point3{math.P3(-10, 40, 150), math.P3(-40, 10, 150)}},
		{name: "C7", wall: "Cone 50→apex",
			wantVerts:   []math.Point3{math.P3(-10, 0, 96), math.P3(0, 10, 96)},
			bannedVerts: []math.Point3{math.P3(-10, 0, 120), math.P3(0, 10, 120)}},
		{name: "D7", wall: "Sphere r=150",
			wantVerts: []math.Point3{
				math.P3(74.3303437365925, 10, 129.903810567666), math.P3(-10, -74.3303437365925, 129.903810567666),
				math.P3(90.1281099954577, 0, 119.903810567666), math.P3(0, -90.1281099954577, 119.903810567666)},
			bannedVerts: []math.Point3{math.P3(-10, -75, 129.903810567666), math.P3(0, -75, 119.903810567666)}},
	}
}

// assertBodyHasVertex fails unless the body carries a vertex at DRAWEXE's coordinate.
func assertBodyHasVertex(t *testing.T, b *topo.Body, want math.Point3, diag float64, tc farEndWallCase) {
	t.Helper()
	if d := nearestVertexDistance(b, want); d/diag > farEndVertexTol {
		t.Errorf("%s: no vertex at DRAWEXE's far end (%.10g,%.10g,%.10g) on the %s wall — nearest is %.6g away (rel %.4g, tol %.1g)",
			tc.name, want.X, want.Y, want.Z, tc.wall, d, d/diag, farEndVertexTol)
	}
}

// assertBodyLacksVertex fails when the body still carries the UN-trimmed far end: the flank's parametric
// extreme, which lies off the stop wall. This is the direct anti-regression on the run-on itself.
func assertBodyLacksVertex(t *testing.T, b *topo.Body, banned math.Point3, diag float64, tc farEndWallCase) {
	t.Helper()
	if d := nearestVertexDistance(b, banned); d/diag <= farEndVertexTol {
		t.Errorf("%s: band far end is still squared off at the flank extreme (%.10g,%.10g,%.10g), which is off the %s wall",
			tc.name, banned.X, banned.Y, banned.Z, tc.wall)
	}
}

// nearestVertexDistance is the distance from p to the body's closest vertex.
func nearestVertexDistance(b *topo.Body, p math.Point3) float64 {
	best := stdmath.Inf(1)
	for _, v := range b.Vertices() {
		if d := v.Point().DistanceTo(p); d < best {
			best = d
		}
	}
	return best
}
