// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"sort"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
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
//
// A SECOND live false-green was then found by the same reasoning, in the CORNER FILL: with the fill built by
// projecting a straight chord onto each host, the corner patch measured 59.273 against result_5's 80.733
// (−27%, Hausdorff to OCCT's own patch 1.03 ≈ 21% of r) while the vertical plane it rails on over-read by
// +29.71 — and the two nearly cancelled into +0.008% whole-body, again inside deps. Proven by mutation:
// reverting the fill to chord projection turns this test RED naming the 59.273 patch. Both faces are pinned
// below, which is why the gate now covers five faces rather than three.
//
// The patch is pinned by its INTEGRATED SURFACE area, not its mesh area: DRAWEXE's `sprops` is itself a
// surface quadrature, and a mesh figure would spend two thirds of the tolerance budget on the tessellator's
// own discretisation error, leaving 1.13× headroom instead of 3.6× (n4MeshSanityRelTol documents the
// measurements). The mesh is still checked, loosely, so the full-domain-trim premise stays honest.

// TestN4CornerWeldLayerWatertight is the whole-body gate: watertight, fold-free, OCCT's face count and area.
func TestN4CornerWeldLayerWatertight(t *testing.T) {
	t.Parallel()
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
//
// result_5 (the CORNER PATCH) and result_1 (the vertical plane the corner's ball rolls on) are the two the
// CORNER FILL moves, and they were the two the earlier gate did not pin — which is exactly how a corner
// patch 27% short of the oracle survived as a green: the patch read 59.273 and the vplane 8704.54 (+29.71),
// and those two errors mostly CANCELLED into a whole-body +0.008%, inside deps. Pinning them is what makes
// the green mean the right corner surface rather than the right total.
const (
	n4OracleBossWall180 = 2827.43
	n4OracleBossWall90  = 1232.49
	n4OracleTopPlane    = 456.557
	n4OracleCornerPatch = 80.7328
	n4OracleVPlane      = 8674.84
	// 0.002 relative. Our worst per-face deviation is the corner patch's 5.6e-4 (the rolling-ball canal's
	// between-station interpolation residual against OCCT's own degree-8 approximation of the same surface);
	// every other face is under 5e-5. The wrong-half body is off by 21% (wall) and 38% (top plane) and the
	// superseded chord-projected fill by 27% (patch), so this separates them by an order of magnitude or more.
	n4PerFaceRelTol = 0.002
	// The corner patch is pinned by its INTEGRATED SURFACE area, not by its mesh area, because the mesh
	// carries the TESSELLATOR's discretisation error stacked on top of the surface's own. Measured: surface
	// 80.7781 (rel 5.61e-4 — 3.6× headroom under n4PerFaceRelTol) versus mesh 80.8753 (rel 1.77e-3 — only
	// 1.13× headroom), so ~⅔ of the mesh figure's budget is spent on mesh density, an actively-tuned
	// heuristic in this repo. Pinning the mesh at 0.002 would therefore (a) turn N4 spuriously RED on any
	// NURBS-tessellator density change and (b) leave no room to catch a real 0.2% SURFACE regression. The
	// mesh area is still asserted, at this deliberately looser tolerance, as a coarse sanity check AND as
	// the guard that the patch face is still trimmed to its whole parameter domain — which is the premise
	// that makes the full-domain integral the FACE's area rather than the surface's.
	n4MeshSanityRelTol = 0.01
)

// assertN4HostFacesMatchOraclePerFace pins the five faces the two live false-green mechanisms move: the
// three HOST faces that discriminate which half of the contact circle the 180° rim rail runs along (the
// check the torus-span assertion is structurally blind to), plus the CORNER PATCH and the vertical plane
// its rail rides, which discriminate the corner fill's SHAPE. See the ★ note on this file's header.
func assertN4HostFacesMatchOraclePerFace(t *testing.T, body *topo.Body) {
	t.Helper()
	walls := n4BossWallSectorAreas(body)
	if len(walls) != 2 {
		t.Fatalf("N4 has %d r=20 boss-wall cylinder faces, want 2 (the 180° and the 90° sector)", len(walls))
	}
	assertFaceAreaAgainstOracle(t, "N4 boss wall, 180° sector (oracle result_9)", walls[0], n4OracleBossWall180, n4PerFaceRelTol)
	assertFaceAreaAgainstOracle(t, "N4 boss wall, 90° sector (oracle result_14)", walls[1], n4OracleBossWall90, n4PerFaceRelTol)
	assertFaceAreaAgainstOracle(t, "N4 boss top plane, 270° pie receded to r=15 (oracle result_11)",
		n4BossTopPlaneArea(t, body), n4OracleTopPlane, n4PerFaceRelTol)
	// ★ THE LOAD-BEARING corner-fill assertion. DRAWEXE's own `sprops result_5 1.e-9` is a Gauss-quadrature
	// SURFACE integral (it reports relative error 0), so integrating our patch the same way compares like
	// with like. It separates the superseded chord-projected fill by 133× — that fill's 59.2728 is 21.46
	// absolute off a ±0.1615 band — where the vplane assertion below separates it by only 1.7×.
	patch := n4CornerPatchFace(t, body)
	assertFaceAreaAgainstOracle(t, "N4 corner patch SURFACE integral, the rolling-ball canal (oracle result_5)",
		bsplinePatchSurfaceArea(patch), n4OracleCornerPatch, n4PerFaceRelTol)
	assertFaceAreaAgainstOracle(t, "N4 corner patch MESH (loose sanity + full-domain-trim guard only)",
		faceMeshArea2(patch), n4OracleCornerPatch, n4MeshSanityRelTol)
	// Corroborating, NOT load-bearing: the chord-projected fill put +29.70 on this plane, which is only 1.7×
	// the ±17.35 tolerance band, so on its own it is a thin guard. It is kept because it witnesses the OTHER
	// half of the redistribution the two errors used to cancel across.
	assertFaceAreaAgainstOracle(t, "N4 vertical plane the corner ball rolls on (oracle result_1)",
		n4CornerVPlaneArea(t, body), n4OracleVPlane, n4PerFaceRelTol)
}

// n4CornerPatchFace returns the corner patch — the result's only BSpline face (every other N4 face is an
// analytic plane, cylinder or torus, per the DRAWEXE face inventory).
func n4CornerPatchFace(t *testing.T, body *topo.Body) *topo.Face {
	t.Helper()
	var found []*topo.Face
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.BSplineSurface); ok {
			found = append(found, f)
		}
	}
	if len(found) != 1 {
		t.Fatalf("N4 carries %d BSpline faces, want exactly 1 (the corner patch)", len(found))
	}
	return found[0]
}

// n4PatchAreaCells is the Gauss cell grid the corner patch's surface integral uses per parameter direction.
// Converged on N4: 4×4 → 80.778102, 8×8 → 80.778102, 16×16 → 80.778102 (a 5-point rule is exact to degree
// 9, and the patch is one Bézier span in u × a cubic-lofted v, so the rule resolves the integrand outright).
const n4PatchAreaCells = 8

// gauss5Nodes / gauss5Weights are the 5-point Gauss-Legendre rule on [−1,1].
var (
	gauss5Nodes   = [5]float64{-0.906179845938664, -0.5384693101056831, 0, 0.5384693101056831, 0.906179845938664}
	gauss5Weights = [5]float64{0.2369268850561891, 0.4786286704993665, 0.5688888888888889, 0.4786286704993665, 0.2369268850561891}
)

// bsplinePatchSurfaceArea is the TRUE surface area of a BSpline face — ∫∫|S_u × S_v| du dv over the whole
// parameter domain by a tensor Gauss-Legendre rule — carrying none of the tessellator's discretisation
// error (see n4MeshSanityRelTol for why that distinction is the point). It integrates the FULL domain, which
// is the face only while the face is trimmed to exactly its own boundary isoparms (the corner patch is, via
// canalPatchLoops); the companion mesh-area assertion is what keeps that premise honest.
func bsplinePatchSurfaceArea(f *topo.Face) float64 {
	surf := f.Geometry().(geom.BSplineSurface)
	u0, u1 := surf.UDomain()
	v0, v1 := surf.VDomain()
	hu, hv := (u1-u0)/n4PatchAreaCells, (v1-v0)/n4PatchAreaCells
	area := 0.0
	for i := 0; i < n4PatchAreaCells; i++ {
		for j := 0; j < n4PatchAreaCells; j++ {
			area += gaussCellArea(surf, u0+(float64(i)+0.5)*hu, v0+(float64(j)+0.5)*hv, hu, hv)
		}
	}
	return area
}

// gaussCellArea is one cell's share of the area integral: the 5×5 Gauss-Legendre sum of the area element
// |S_u × S_v| over the cell centred at (cu, cv) with sides (hu, hv).
func gaussCellArea(surf geom.BSplineSurface, cu, cv, hu, hv float64) float64 {
	sum := 0.0
	for a, ua := range gauss5Nodes {
		for b, vb := range gauss5Nodes {
			su, sv := surf.DerivativesAt(cu+ua*hu/2, cv+vb*hv/2)
			sum += gauss5Weights[a] * gauss5Weights[b] * float64(su.Cross(sv).Length())
		}
	}
	return sum * hu * hv / 4
}

// n4CornerVPlaneArea returns the mesh area of the box wall the corner ball rolls on: the wall the boss axis
// lies in, identified geometrically as the vertical plane whose in-plane offset along its own normal is the
// box's 100 (its opposite wall is at 0, and the other pair's normal points the other way).
func n4CornerVPlaneArea(t *testing.T, body *topo.Body) float64 {
	t.Helper()
	want := math.V3(0.984807753012208, -0.17364817766693, 0)
	for _, f := range body.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok || stdmath.Abs(float64(pl.Normal().Dot(want))) < 1-1e-9 {
			continue
		}
		if stdmath.Abs(float64(pl.Origin.VectorTo(math.P3(0, 0, 0)).Dot(want))+100) < 1e-6 {
			return faceMeshArea2(f)
		}
	}
	t.Fatalf("N4 carries no corner vertical plane (the box wall at offset 100 along (0.9848,-0.1736,0))")
	return 0
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

// assertFaceAreaAgainstOracle fails unless got matches the DRAWEXE per-face value within relTol. relTol is
// per-assertion, not global, because the surface-integral and mesh-area measures of the SAME face carry
// different error stacks (n4PerFaceRelTol vs n4MeshSanityRelTol).
func assertFaceAreaAgainstOracle(t *testing.T, what string, got, want, relTol float64) {
	t.Helper()
	if rel := stdmath.Abs(got-want) / want; rel > relTol {
		t.Fatalf("%s measured %.4f, want DRAWEXE %.4f within %.3f relative (rel %.6f)",
			what, got, want, relTol, rel)
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
