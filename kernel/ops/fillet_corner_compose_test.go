// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestComposeMixedTorusAndDihedral is the composition PROOF the unified corner-setback pass exists for:
// a NATIVE box+boss filleted so ONE corner is a mixedTorus AND a DIFFERENT corner is a dihedralMiter —
// two distinct setback corner-TYPES on one body. The OLD four-pass engine fired the mixedTorus pass for
// corner A, EARLY-RETURNED, and swallowed corner B's dihedral setback (leaving an over-keep tab); the
// unified pass classifies EVERY corner and composes BOTH. Fixture: box 100³ + boss [40,80]²×h40 on the
// top; fillet r=5 on the boss's two base edges at corner A=(80,40,100) + the boss vertical there (→ A is
// mixedTorus: 2 concave + 1 convex) + one more base edge at x=40 (→ corner B=(40,40,100) is a 2-edge
// concave-orthogonal dihedralMiter, its vertical unfilleted).
func TestComposeMixedTorusAndDihedral(t *testing.T) {
	body := boxWithBoss(t)
	keys := composeMixedAndDihedralKeys(body)
	if len(keys) != 4 {
		t.Fatalf("compose fixture selected %d edges, want 4 (2 base + 1 vertical at A, + 1 base at B)", len(keys))
	}
	res, err := FilletEdgesCorner(body, filletPicksFor(keys, 5), CornerMiter, FillConcaveOutward)
	if err != nil {
		t.Fatalf("fillet compose box+boss: %v", err)
	}
	assertComposedWatertight(t, res)
	assertSingleMixedTorus(t, res)
	// Corner B set back — DISCRIMINATING on the corner-B SEAM VERTEX, not the hole's bounding box.
	// The dihedral setback re-samples B's miter seam so its receded corner vertex lands at exactly
	// (35,35,100) — the two shared-face rails EXTENDED +r past the raw (40,40) endpoint MEET there.
	// If corner B's dihedral is DROPPED (the old early-return), B stays the reflected-cylinder over-keep
	// miter whose corner vertex is (45,45,100) and (35,35,100) is ABSENT from the loop. The bounding-box
	// lo is (35,35) EITHER WAY (the two bands' independent r-retraction supplies min-x/min-y via the
	// (75,35) and (35,80) rail ends), so only the PRESENCE of the (35,35,100) seam vertex — and the
	// ABSENCE of the (45,45,100) over-keep vertex — proves B's setback composed. (Verified: dropping the
	// dihedral treatment for B flips the loop vertex (35,35,100)→(45,45,100) and fails this test.)
	eps := ResolutionForBody(res).Weld()
	verts := boxTopHoleLoopVertices(t, res, eps)
	if !holeLoopHasPoint(verts, math.P3(35, 35, 100), eps) {
		t.Fatalf("box-top hole loop %v lacks the receded miter vertex (35,35,100): corner B's dihedral setback was dropped", verts)
	}
	if holeLoopHasPoint(verts, math.P3(45, 45, 100), eps) {
		t.Fatalf("box-top hole loop %v carries the over-keep seam vertex (45,45,100): corner B kept the un-set-back reflected miter", verts)
	}
}

// boxTopHoleLoopVertices returns the ordered vertices of the box-top (z=100) face's single hole loop.
func boxTopHoleLoopVertices(t *testing.T, b *topo.Body, eps float64) []math.Point3 {
	t.Helper()
	for _, f := range b.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok || stdmath.Abs(pl.Normal().Z) < 0.99 || stdmath.Abs(pl.Origin.Z-100) > eps {
			continue
		}
		for _, l := range f.Loops() {
			if l.IsOuter() {
				continue
			}
			var out []math.Point3
			for _, u := range l.EdgeUses() {
				out = append(out, u.Edge().StartVertex().Point())
			}
			return out
		}
	}
	t.Fatal("no box-top (z=100) plane face with a hole loop")
	return nil
}

// holeLoopHasPoint reports whether any vertex in verts coincides with want within eps.
func holeLoopHasPoint(verts []math.Point3, want math.Point3, eps float64) bool {
	for _, v := range verts {
		if v.DistanceTo(want) < eps {
			return true
		}
	}
	return false
}

// TestComposeConcaveSphereAndDihedral guards the Phase-B re-solve / Phase-C railWrite ORDER: a box −
// blind pocket whose ONE trihedral corner (20,20,60) joins three concave fillets (→ concaveSphere, a
// void-sphere blend replacement forcing a re-solve) AND a separate concave dihedral edge-pair at
// (50,20,60) (→ dihedralMiter, a Phase-C railWrite). The re-solve regenerates fils from the flipped
// blends BEFORE the dihedral railWrite lands, so the dihedral setback must survive it. Assert: the
// void-side octant sphere at (25,25,65) is present (re-solve worked) AND the pocket-floor rail at x=50
// receded to 45 (=50−r: the dihedral railWrite was NOT clobbered by the re-solve) AND watertight.
func TestComposeConcaveSphereAndDihedral(t *testing.T) {
	body := boxWithPocket(t)
	keys := composeSphereAndDihedralKeys(t, body)
	if len(keys) != 4 {
		t.Fatalf("compose fixture selected %d edges, want 4 (3 at the pocket corner + 1 floor edge at x=50)", len(keys))
	}
	res, err := FilletEdgesCorner(body, filletPicksFor(keys, 5), CornerMiter, FillConcaveOutward)
	if err != nil {
		t.Fatalf("fillet compose box−pocket: %v", err)
	}
	assertComposedWatertight(t, res)
	eps := ResolutionForBody(res).Weld()
	assertVoidSphereCentre(t, res, math.P3(25, 25, 65), eps) // Phase-B re-solve produced the void octant
	if hiX := pocketFloorHoleMaxX(t, res, eps); stdmath.Abs(hiX-45) > eps {
		t.Fatalf("pocket-floor hole reaches x=%.4f, want 45 (=50−r): the dihedral railWrite was clobbered by the re-solve", hiX)
	}
}

// assertComposedWatertight fails unless res is a valid, closed, hole-contained solid whose body
// tessellation is fold-free — the composed body must be watertight, never a tab-carrying degenerate.
func assertComposedWatertight(t *testing.T, res *topo.Body) {
	t.Helper()
	rep := Validate(res)
	if !rep.Valid || !rep.Closed || !rep.HolesContained || !res.IsSolid() {
		t.Fatalf("composed body not watertight: valid=%v closed=%v holes=%v solid=%v",
			rep.Valid, rep.Closed, rep.HolesContained, res.IsSolid())
	}
	if mesh, _ := TessellateBody(res, PropertyQuality()); FoldEdgeCount(mesh) != 0 {
		t.Fatalf("composed body mesh has %d fold edges, want 0 (non-manifold weld)", FoldEdgeCount(mesh))
	}
}

// assertSingleMixedTorus fails unless res carries EXACTLY ONE corner torus of the derived mixed-sense
// R=2r=10, minor r=5 patch (mesh area ≈84.10) and zero corner spheres — proof corner A composed as a
// torus, not the oversized sphere a dropped mixedTorus treatment would leave (274.35).
func assertSingleMixedTorus(t *testing.T, res *topo.Body) {
	t.Helper()
	eps := ResolutionForBody(res).Weld()
	tori, spheres := 0, 0
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Sphere); ok {
			spheres++
		}
		tor, ok := f.Geometry().(geom.Torus)
		if !ok {
			continue
		}
		tori++
		if stdmath.Abs(tor.MajorRadius-10) > eps || stdmath.Abs(tor.MinorRadius-5) > eps {
			t.Fatalf("corner torus major=%.6f minor=%.6f, want R=2r=10 minor=r=5", tor.MajorRadius, tor.MinorRadius)
		}
		if a := faceTriArea(f); stdmath.Abs(a-84.10) > 0.02*84.10 {
			t.Fatalf("corner torus area %.4f, want ≈84.10 (a forced sphere reads 274.35)", a)
		}
	}
	if tori != 1 || spheres != 0 {
		t.Fatalf("composed body has %d corner tori + %d spheres, want exactly 1 torus and 0 spheres", tori, spheres)
	}
}

// composeMixedAndDihedralKeys selects the box+boss compose picks: the three edges at corner A=(80,40,100)
// (its two concave base edges + the convex boss vertical) plus the base edge running along x=40 (whose
// far end at B=(40,40,100) pairs with A's shared base edge into a concave-orthogonal dihedral miter).
func composeMixedAndDihedralKeys(body *topo.Body) [][]byte {
	keys := append([][]byte(nil), edgeKeysAtVertex(body, math.P3(80, 40, 100))...)
	for _, e := range body.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		onX40 := stdmath.Abs(a.X-40) < 1e-6 && stdmath.Abs(c.X-40) < 1e-6
		atZ100 := stdmath.Abs(a.Z-100) < 1e-6 && stdmath.Abs(c.Z-100) < 1e-6
		if onX40 && atZ100 && ClassifyEdgeConvexity(e) == EdgeConcave {
			keys = append(keys, e.ReferenceKey())
		}
	}
	return keys
}

// composeSphereAndDihedralKeys selects the box−pocket compose picks: the three concave edges at the
// pocket corner (20,20,60) (→ concaveSphere) plus the one floor edge at x=50 (→ with (20,20)'s y=20 floor
// edge it forms a concave-orthogonal dihedral miter at the shared corner (50,20,60)).
func composeSphereAndDihedralKeys(t *testing.T, body *topo.Body) [][]byte {
	keys := append([][]byte(nil), pocketCornerEdgeKeys(t, body, math.P3(20, 20, 60))...)
	for _, e := range body.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		atZ60 := stdmath.Abs(a.Z-60) < 1e-6 && stdmath.Abs(c.Z-60) < 1e-6
		atX50 := stdmath.Abs(a.X-50) < 1e-6 && stdmath.Abs(c.X-50) < 1e-6
		if atZ60 && atX50 && ClassifyEdgeConvexity(e) == EdgeConcave {
			keys = append(keys, e.ReferenceKey())
		}
	}
	return keys
}

// pocketFloorHoleMaxX returns the maximum vertex x of the pocket-floor (z=60) face's boundary loop — the
// x=50 dihedral corner recedes to 45 (=50−r) when its setback fires, staying at 50 if it was dropped.
func pocketFloorHoleMaxX(t *testing.T, b *topo.Body, eps float64) float64 {
	t.Helper()
	for _, f := range b.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok || stdmath.Abs(pl.Normal().Z) < 0.99 || stdmath.Abs(pl.Origin.Z-60) > eps {
			continue
		}
		_, hi := loopXYExtent(f.Loops()[0])
		return hi.X
	}
	t.Fatal("no pocket-floor (z=60) plane face")
	return 0
}
