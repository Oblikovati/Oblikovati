// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// testBox5 builds the same 5^3 native box the box-corner tests use, package-internal so these
// tests can reach the unexported partial-corner helpers directly.
func testBox5(t *testing.T) *topo.Body {
	t.Helper()
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(5, 5, 5), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	return box
}

// boxCornerPicks returns filletPicks for the two top edges sharing vertex (5,0,5) — the SAME
// corpus P8/V8 corner fillet_box_corner_test.go uses natively, radius r.
func boxCornerPicks(t *testing.T, b *topo.Body, r float64) []filletPick {
	t.Helper()
	var ps []filletPick
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if boxEdgeMatches(a, c, math.P3(5, 0, 5), math.P3(5, 5, 5)) ||
			boxEdgeMatches(a, c, math.P3(0, 0, 5), math.P3(5, 0, 5)) {
			ps = append(ps, filletPick{edge: e, r0: r, r1: r})
		}
	}
	if len(ps) != 2 {
		t.Fatalf("expected 2 corner-sharing top edges, found %d", len(ps))
	}
	return ps
}

// boxEdgeMatches is boxCornerEdge (fillet_box_corner_test.go), duplicated locally since that one
// lives in the external ops_test package and is not visible from here.
func boxEdgeMatches(a, c, p, q math.Point3) bool {
	return (a.DistanceTo(p) < 1e-6 && c.DistanceTo(q) < 1e-6) ||
		(a.DistanceTo(q) < 1e-6 && c.DistanceTo(p) < 1e-6)
}

// TestTouchedFacesOfPicksBoxCorner: the box's shared-corner picks (2 edges meeting at a trihedral
// vertex, sharing ONE face) touch exactly 3 distinct faces — the shared one plus each edge's outer
// face — matching the ordinary trihedral solveBlend's own face count, even though this box corner
// is never actually routed through solvePartialCorner (its vertex valence is 3, not >3).
func TestTouchedFacesOfPicksBoxCorner(t *testing.T) {
	t.Parallel()
	ps := boxCornerPicks(t, testBox5(t), 1)
	touched := touchedFacesOfPicks(ps)
	if len(touched) != 3 {
		t.Fatalf("touched faces = %d, want 3", len(touched))
	}
}

// TestPartialCornerLoopGapDetectsOpenFan: the box's 2-of-3-edge corner is exactly the "open fan"
// shape a genuine partial corner has — the two OUTER touched faces are each shared by only ONE of
// the two picks (only the shared middle face has degree 2) — so partialCornerLoopGap must report
// them. This is the structural condition that makes chainArcs panic (see the file doc comment);
// asserting it here, independent of solveCorner's dispatch, pins the pure detector's own logic.
func TestPartialCornerLoopGapDetectsOpenFan(t *testing.T) {
	t.Parallel()
	ps := boxCornerPicks(t, testBox5(t), 1)
	touched := touchedFacesOfPicks(ps)
	gap := partialCornerLoopGap(touched, ps)
	if len(gap) != 2 {
		t.Fatalf("loop gap faces = %d, want 2 (the two faces touched by only one pick)", len(gap))
	}
}

// TestPartialCornerLoopGapNilOnFullCycle: feeding the SAME pick twice (simulating a face touched by
// two arms, degree 2) must NOT be reported as a gap — the detector keys strictly on face id, so a
// face reached by two independent picks (or, degenerately, one pick counted twice) satisfies
// closure. This regresses the degree-counting arithmetic itself, decoupled from any real topology.
func TestPartialCornerLoopGapNilOnFullCycle(t *testing.T) {
	t.Parallel()
	ps := boxCornerPicks(t, testBox5(t), 1)
	touched := touchedFacesOfPicks(ps)
	doubled := append(append([]filletPick{}, ps...), ps...) // every touched face now has degree 2 or 4
	gap := partialCornerLoopGap(touched, doubled)
	if gap != nil {
		t.Fatalf("loop gap = %v, want nil once every touched face is reached by >=2 picks", gap)
	}
}

// TestPartialCornerArmsSupportedRejectsVaryingRadius: a varying pick (r0 != r1) cannot share a
// partial planar corner (its cone has no seam with a cylinder) — same precondition the full-round
// K-gon (fullRoundArmsSupported) and the ordinary miter both enforce.
func TestPartialCornerArmsSupportedRejectsVaryingRadius(t *testing.T) {
	t.Parallel()
	ps := boxCornerPicks(t, testBox5(t), 1)
	ps[0].r1 = 2 // varying: r0=1, r1=2
	if err := partialCornerArmsSupported(ps); err == nil {
		t.Fatal("expected an error for a varying-radius pick, got nil")
	}
}

// sharedCornerVertexID returns the vertex id common to both picks' edges (the trihedral corner
// they meet at), failing the test if the two edges do not share exactly one endpoint.
func sharedCornerVertexID(t *testing.T, ps []filletPick) uint64 {
	t.Helper()
	a := map[uint64]bool{ps[0].edge.StartVertex().ID(): true, ps[0].edge.EndVertex().ID(): true}
	for _, id := range []uint64{ps[1].edge.StartVertex().ID(), ps[1].edge.EndVertex().ID()} {
		if a[id] {
			return id
		}
	}
	t.Fatal("picks share no vertex")
	return 0
}

// TestArmSpinePerpDistanceZeroAtTrihedralSphereCentre cross-checks armSpinePerpDistance against
// the ALREADY-TRUSTED solveBlend sphere: the box's ordinary trihedral corner sphere (3 faces, both
// picks' arm spines) must measure distance ~0 from itself, since armCornerCentre's own spine-
// concurrence gate (fillet.go) adopts that exact sphere for every arm of every shipped trihedral
// corner. This is the numeric identity the partial-corner decline path leans on (see
// solvePartialCorner) — pinned here against known-good geometry instead of only exercised
// indirectly through the corpus fixtures that currently never reach it.
func TestArmSpinePerpDistanceZeroAtTrihedralSphereCentre(t *testing.T) {
	t.Parallel()
	box := testBox5(t)
	ps := boxCornerPicks(t, box, 1)
	v := vertexByID(edgesOf(ps), sharedCornerVertexID(t, ps))
	faces := facesAtVertex(v)
	if len(faces) != 3 {
		t.Fatalf("box corner valence = %d, want 3", len(faces))
	}
	cb, err := solveBlend(box, v, faces, 1)
	if err != nil {
		t.Fatalf("solveBlend: %v", err)
	}
	for i, p := range ps {
		if d := armSpinePerpDistance(v, p, cb.center, 1); d > 1e-9 {
			t.Fatalf("pick %d: spine distance %.3g, want ~0 (the box corner's own sphere)", i, d)
		}
	}
}
