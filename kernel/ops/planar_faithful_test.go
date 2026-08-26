// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

func ngon2D(cx, cy, r float64, n int) []math.Point2 {
	var p []math.Point2
	for i := range n {
		a := 2 * stdmath.Pi * float64(i) / float64(n)
		p = append(p, math.P2(cx+r*stdmath.Cos(a), cy+r*stdmath.Sin(a)))
	}
	return p
}

func planarTrisArea(tris [][3]int, outer []math.Point2, holes [][]math.Point2) float64 {
	verts := append([]math.Point2(nil), outer...)
	for _, h := range holes {
		verts = append(verts, h...)
	}
	var a float64
	for _, t := range tris {
		a += triArea(verts[t[0]], verts[t[1]], verts[t[2]])
	}
	return a
}

// TestPlanarTrisHoledCoversArea: a holed planar face triangulates to exactly its area (outer − holes),
// whichever path planarTris takes — the watertightness precondition (full, non-overlapping coverage).
func TestPlanarTrisHoledCoversArea(t *testing.T) {
	outer := ngon2D(0, 0, 25, 24)
	holes := [][]math.Point2{ngon2D(8, 5, 1.75, 32), ngon2D(-9, -6, 1.75, 32)}
	want := stdmath.Abs(signedArea(outer))
	for _, h := range holes {
		want -= stdmath.Abs(signedArea(h))
	}
	got := planarTrisArea(planarTris(outer, holes), outer, holes)
	if stdmath.Abs(got-want) > 1e-6*want {
		t.Errorf("planarTris area = %g, want %g", got, want)
	}
}

// TestPlanarAreaMatches: accepts a correct triangulation (incl. a zero-area collinear-merge variant)
// and rejects one with an overlap (the earcut-defect signature that triggers the CDT fallback).
func TestPlanarAreaMatches(t *testing.T) {
	// a unit square split with a redundant collinear midpoint on one edge: edge 0-1-2 collinear,
	// triangulating as the two triangles {0,2,3} (the merge) covers the exact area.
	sq := []math.Point2{math.P2(0, 0), math.P2(1, 0), math.P2(2, 0), math.P2(2, 1), math.P2(0, 1)}
	good := [][3]int{{0, 2, 3}, {0, 3, 4}}
	if !planarAreaMatches(good, sq, nil) {
		t.Error("collinear-merge triangulation (exact area) should match")
	}
	overlap := append(append([][3]int{}, good...), good[0]) // duplicate a triangle -> area too big
	if planarAreaMatches(overlap, sq, nil) {
		t.Error("overlapping triangulation (area too big) should NOT match")
	}
	if planarAreaMatches(nil, sq, nil) {
		t.Error("empty triangulation should never match")
	}
}

// TestPlanarTrisCollinearSlotBottomsCoverExactly pins the #39 finding: earcut fans the material below
// three in-line rectangular slots with ONE long edge spanning all three slot bottoms, so each
// slot-bottom loop segment is SUBSUMED (collinear), not reproduced as its own triangle edge — yet the
// coverage/area is exact. This is why the guard checks AREA, not literal per-segment reproduction: a
// per-segment check would flag this trivially-clean face as defective and route it to the CDT. The
// collinear subsumption is watertight (the surface is flat along the shared line).
func TestPlanarTrisCollinearSlotBottomsCoverExactly(t *testing.T) {
	outer := rectHole(0, 0, 10, 10)
	holes := [][]math.Point2{rectHole(1, 1, 3, 9), rectHole(4, 1, 6, 9), rectHole(7, 1, 9, 9)}
	want := stdmath.Abs(signedArea(outer))
	for _, h := range holes {
		want -= stdmath.Abs(signedArea(h))
	}
	tris := planarTris(outer, holes)
	if got := planarTrisArea(tris, outer, holes); stdmath.Abs(got-want) > 1e-6*want {
		t.Errorf("three in-line slots: planarTris area = %g, want %g", got, want)
	}
	// The guard must ACCEPT earcut here (keep the cheap path), not route to the CDT: earcut's own area
	// already matches despite subsuming the slot bottoms.
	if !planarAreaMatches(earcut(outer, holes), outer, holes) {
		t.Error("earcut on three collinear slots is area-exact and must be accepted, not sent to the CDT")
	}
}

// TestPlanarCDTLargeFaceAreaExact pins the justification for retiring the 256-vertex CDT
// budget (#1610): with #1409's corridor-walk segment insertion the CDT re-meshes a
// 600-boundary-vertex annulus area-exact well inside the per-recompute tessellation
// budget — so an area-wrong earcut on a big face routes to it instead of shipping.
func TestPlanarCDTLargeFaceAreaExact(t *testing.T) {
	outer := ngon2D(0, 0, 50, 400)
	hole := ngon2D(0, 0, 20, 200)
	tris := planarCDT(outer, [][]math.Point2{hole})
	if !planarAreaMatches(tris, outer, [][]math.Point2{hole}) {
		t.Errorf("large-annulus CDT area mismatch: %d tris cover %g", len(tris),
			planarTrisArea(tris, outer, [][]math.Point2{hole}))
	}
}

// TestPlanarTrisLargeCleanFaceKeepsEarcut: a large but CLEAN face must still take the
// cheap path — the CDT stays reserved for the genuine area defect.
func TestPlanarTrisLargeCleanFaceKeepsEarcut(t *testing.T) {
	outer := ngon2D(0, 0, 50, 400)
	got := planarTris(outer, nil)
	want := bestSingleLoopTriangulation(outer)
	if len(got) != len(want) {
		t.Errorf("clean large face: planarTris returned %d tris, want earcut's %d", len(got), len(want))
	}
}
