// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"sort"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// The per-case gate for OCCT tests/blend/simple/N4 — the first case welded by the GENERAL corner-weld layer
// (kernel/ops/cornerweld_*.go). N4 is a 100³ box with a full r=20 × h=50 cylinder standing on its vertical
// corner (270° of it protrudes), filleted r=5 on three edges meeting at one trihedral vertex: the concave
// boss-wall ∧ box-wall ruling, the CONVEX boss cap-rim arc, and the concave band where the cap plane meets
// the box wall.
//
// DRAWEXE 8.0.0 receipt — `restore data/CFI_e5678fil.rle s ; tscale s 0 0 0 10 ; explode s e ;
// blend result s 5 s_4 5 s_13 5 s_2 ; nbshapes ; sprops ; vprops ; checkshape`:
// valid SOLID, 1 shell, 14 faces / 14 wires / 22 vertices / 34 edges, area 64287.2, volume 1.04694e6.
//
// The decisive thing this asserts beyond the area scoreboard is the RIM CONTINUATION. Only the 90° piece of
// the 270° cap rim is picked, and its far vertex is a G1 seam on the boss wall — so a weld that terminates
// the arm there would leave the rest of the rim sharp and produce a DIFFERENT solid. OCCT's blend runs the
// fillet over the whole tangent chain, emitting the band as two torus faces split at the wall-face seam
// (76.3° over the first wall face, EXACTLY 180° over the second). Both spans are checked below, so a
// regression that drops the continuation fails loud.
//
// ★ STANDARD FOR A NEW CORNER GREEN: reconcile the oracle PER FACE, not only on the whole-body scoreboard.
// A whole-body area inside `deps` (0.01) is NOT evidence of the right solid — the corner classes redistribute
// hundreds of units between neighbouring faces, so a geometrically wrong weld can land watertight, fold-free
// and in tolerance. This case is the worked proof: taking the WRONG half of the 180° contact circle in
// sweptContactRail yields a valid solid reading boss wall 3434.09 (oracle 2827.43) and top plane 283.35
// (oracle 456.56) for a net +0.68% — inside deps. Span assertions cannot see it either, because at exactly
// 180° BOTH halves span π and mesh to the same area. Only the per-face areas below discriminate, so every
// new corner green must pin the faces the oracle's own `explode result f ; sprops result_i` reports.

// TestN4CornerWeldLayerWatertight is the whole-body gate: watertight, fold-free, OCCT's face count and area.
func TestN4CornerWeldLayerWatertight(t *testing.T) {
	body := caseResultBody(t, "N4")
	assertWatertight(t, "N4", body, 14)
	assertWholeBodyFoldFree(t, "N4", body)
	assertWholeBodyArea(t, "N4", body, 64287.2)
	assertN4RimBandIsContinuedAndSplit(t, body)
	assertN4HostFacesMatchOraclePerFace(t, body)
}

// DRAWEXE 8.0.0 per-face receipt for N4, re-verified live (`explode result f` then `sprops result_i`):
// result_9 = the boss wall's 180° sector, result_14 = its 90° sector (two distinct r=20 cylinders, both
// receded from z=50 to z=45), result_11 = the boss top plane, the 270° pie receded to r=15. These three are
// the faces the rim rail's HALF moves; the torus band's own spans do not move at all (see the ★ note above).
const (
	n4OracleBossWall180 = 2827.43
	n4OracleBossWall90  = 1232.49
	n4OracleTopPlane    = 456.557
	// 0.002 relative. Our worst per-face deviation is the top plane's 4e-5; the wrong-half body is off by
	// 21% (wall) and 38% (top plane), so this separates them by two orders of magnitude on either side.
	n4PerFaceRelTol = 0.002
)

// assertN4HostFacesMatchOraclePerFace pins the three HOST faces whose areas actually discriminate which half
// of the contact circle the 180° rim rail runs along — the check the torus-span assertion is structurally
// blind to. See the ★ note on this file's header for the wrong-half evidence.
func assertN4HostFacesMatchOraclePerFace(t *testing.T, body *topo.Body) {
	t.Helper()
	walls := n4BossWallSectorAreas(body)
	if len(walls) != 2 {
		t.Fatalf("N4 has %d r=20 boss-wall cylinder faces, want 2 (the 180° and the 90° sector)", len(walls))
	}
	assertFaceAreaAgainstOracle(t, "N4 boss wall, 180° sector (oracle result_9)", walls[0], n4OracleBossWall180)
	assertFaceAreaAgainstOracle(t, "N4 boss wall, 90° sector (oracle result_14)", walls[1], n4OracleBossWall90)
	assertFaceAreaAgainstOracle(t, "N4 boss top plane, 270° pie receded to r=15 (oracle result_11)",
		n4BossTopPlaneArea(t, body), n4OracleTopPlane)
}

// n4BossWallSectorAreas returns the mesh areas of the boss wall's cylinder faces (the only radius-20
// cylinders in the result; the fillet's own arm cylinders are radius 5), largest first.
func n4BossWallSectorAreas(body *topo.Body) []float64 {
	var areas []float64
	for _, f := range body.Faces() {
		if cyl, ok := f.Geometry().(geom.Cylinder); ok && stdmath.Abs(cyl.Radius-20) < 1e-6 {
			areas = append(areas, faceMeshArea2(f))
		}
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(areas)))
	return areas
}

// n4BossTopPlaneArea returns the mesh area of the boss's top plane — the unique z-normal plane at z=50 (the
// box's own z-normal planes sit at z=0 and z=100).
func n4BossTopPlaneArea(t *testing.T, body *topo.Body) float64 {
	t.Helper()
	for _, f := range body.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if ok && stdmath.Abs(float64(pl.Normal().Z)) > 1-1e-9 && stdmath.Abs(float64(pl.Origin.Z)-50) < 1e-6 {
			return faceMeshArea2(f)
		}
	}
	t.Fatalf("N4 carries no boss top plane (a z-normal plane at z=50)")
	return 0
}

// assertFaceAreaAgainstOracle fails unless got matches the DRAWEXE per-face value within n4PerFaceRelTol.
func assertFaceAreaAgainstOracle(t *testing.T, what string, got, want float64) {
	t.Helper()
	if rel := stdmath.Abs(got-want) / want; rel > n4PerFaceRelTol {
		t.Fatalf("%s meshed to %.4f, want DRAWEXE %.4f within %.3f relative (rel %.6f)",
			what, got, want, n4PerFaceRelTol, rel)
	}
}

// assertN4RimBandIsContinuedAndSplit checks the convex cap-rim fillet is present as TWO torus faces of the
// same tube (major R−r = 15, minor r = 5) whose spans are the oracle's — 76.3° over the first boss-wall face
// and exactly 180° over the second. The 180° face is the load-bearing one for the CONTINUATION: it is the
// span the arm only reaches by running through the G1 seam. It says NOTHING about which half of the contact
// circle the rail took — at exactly 180° both halves have the same endpoints and the same span, so the two
// torus faces come out bit-identical either way; assertN4HostFacesMatchOraclePerFace is what discriminates.
func assertN4RimBandIsContinuedAndSplit(t *testing.T, body *topo.Body) {
	t.Helper()
	var spans []float64
	for _, f := range body.Faces() {
		tor, ok := f.Geometry().(geom.Torus)
		if !ok || stdmath.Abs(tor.MajorRadius-15) > 1e-3 || stdmath.Abs(tor.MinorRadius-5) > 1e-3 {
			continue
		}
		spans = append(spans, faceMeshArea2(f)/(tor.MinorRadius*(tor.MajorRadius*stdmath.Pi/2+tor.MinorRadius)))
	}
	if len(spans) != 2 {
		t.Fatalf("N4 has %d R−r=15 cap-rim torus faces, want 2 (the rim continuation split at the boss-wall seam)", len(spans))
	}
	half := stdmath.Max(spans[0], spans[1])
	if stdmath.Abs(half-stdmath.Pi) > 0.01 {
		t.Fatalf("N4's continued rim band spans %.4f rad, want π (the whole second boss-wall face)", half)
	}
	// The oracle's own total: (190.242+448.65)/(r·(R·π/2+r)) = 4.4737 rad = 256.3° — the 270° rim less the
	// slice the corner patch consumes. Ours reads 4.4519 (−0.5%, the corner-patch redistribution of §3).
	if total := spans[0] + spans[1]; total < 4.42 || total > 4.53 {
		t.Fatalf("N4's rim band spans %.4f rad in total, want ≈4.4737 (the oracle's 270° rim less the corner patch)", total)
	}
}
