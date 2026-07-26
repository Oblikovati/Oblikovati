// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The per-case gate for OCCT tests/blend/simple/O1 — the SECOND case welded by the general corner-weld layer
// (kernel/ops/cornerweld_*.go) and the layer's own falsification test: O1 cost one plan CONFIGURATION
// (kernel/ops/cornerweld_class_o1.go) plus its corner solve, with no new executor stage.
//
// The shape: a r=50 × h=130 cylinder fused to a 30 × 50 × 70 box that protrudes out of it (the box's two
// vertical edges at (50,0) and (80,10) sit exactly ON the cylinder), filleted r=5 on the three edges meeting
// at (80,10,90) — the CONCAVE box-wall ∧ cylinder ruling, the CONCAVE arc where the box top meets the
// cylinder (a cove torus, major R+r = 55), and the CONVEX box-top ∧ box-wall edge. Roll-sense regime R2 for
// both concave arms.
//
// DRAWEXE 8.0.0 receipt, re-verified live for this slice (`source test-utilities/occt-blend/oracle/drawenv.sh`;
// `restore test-utilities/occt-blend/data/CFI_f5678fin.rle s ; tscale s 0 0 0 10 ; explode s e ;
// blend result s 5 s_7 5 s_6 5 s_14 ; nbshapes ; sprops ; vprops ; checkshape`):
// "This shape seems to be valid", 1 SOLID / 1 SHELL, 12 FACE / 13 WIRE / 18 VERTEX / 27 EDGE,
// sprops Mass = 65104.9, vprops Mass = 1.11166e+06. Thirteen wires for twelve faces because the boss wall
// carries the corner's own INNER loop (the window the box cuts in it) — the input already has 8 faces / 9
// wires, and the blend adds exactly the three arm bands plus the corner patch.
//
// ★ WHY THIS FILE PINS MORE THAN AREAS. Whole-body area is provably blind for corner cases (an audit found
// nine cases 9–19% of radius wrong in shape while inside the 1% area gate; S7 was 18.5% wrong and 0.03%
// right in area), so per-face reconciliation is the project standard. O1 adds a second lesson: the ORACLE
// itself is approximate here. OCCT's corner patch is a rational approximation of the rolling-ball envelope
// whose interior sits up to 1.16% of r off it, and whose two boundary loci — which are also the boss wall's
// and the band arm's boundaries — sit up to 3.3% of r off the exact contact loci. So the three faces around
// the corner CANNOT be pinned tightly against DRAWEXE. What is pinned tightly instead is the property that
// makes ours the exact surface and OCCT's the approximation: every cross-section of our patch is a circle of
// radius exactly r whose centre rides the boss wall at ρ = R + r and rolls on the band's tube at 2r
// (TestO1CornerPatchIsTheExactRollingBallEnvelope). OCCT's own implied ball misses that by 0.174.

// TestO1CornerWeldLayerWatertight is the whole-body gate: watertight, fold-free, OCCT's face count and area.
func TestO1CornerWeldLayerWatertight(t *testing.T) {
	body := caseResultBody(t, "O1")
	assertWatertight(t, "O1", body, 12)
	assertWholeBodyFoldFree(t, "O1", body)
	assertWholeBodyArea(t, "O1", body, o1OracleTotalArea)
	assertO1BossWallCarriesInnerLoop(t, body)
	assertO1FacesMatchOraclePerFace(t, body)
}

// DRAWEXE 8.0.0 per-face receipt for O1 (`explode result f` then `sprops result_i 1.e-9`), identified by area
// + centroid against the known feature geometry:
//
//	result_1  38279.6   the boss wall (2 wires — it carries the corner window)
//	result_2   1296.44  the box bottom z=20, GROWN by the cylinder arm's cap
//	result_3    286.332 the CONCAVE cylinder arm (r5 cylinder at (85, 7.5736, z), z 20..85)
//	result_4   2805.37  the box wall x=50, GROWN by the cove arm's cap (+ r² − πr²/4 = 5.365 exactly)
//	result_5     72.1806 the CORNER PATCH
//	result_6    192.033 the CONCAVE cove torus arm (major 55, minor 5, 27.04° of the 36.87° picked arc)
//	result_7   7853.98  the cylinder's top disc (untouched, π·50²)
//	result_8   7853.98  the cylinder's bottom disc (untouched)
//	result_9   3092.28  the box wall x=80, receded by the convex band and clipped by the cylinder arm
//	result_10  2094.63  the box wall y=−40, RECEDED by the convex band's cap (− 5.365 exactly)
//	result_11   923.937 the box top z=90, clipped by the cove arm's r=55 contact circle and the band
//	result_12   354.112 the CONVEX planar band arm (r5 cylinder, axis ∥ y at (75, ·, 85))
const (
	o1OracleTotalArea = 65104.9
	o1OracleBossWall  = 38279.6
	o1OracleBoxBottom = 1296.44
	o1OracleCylArm    = 286.332
	o1OracleWallX50   = 2805.37
	o1OracleCornerPad = 72.1806
	o1OracleCoveArm   = 192.033
	o1OracleDisc      = 7853.98
	o1OracleWallX80   = 3092.28
	o1OracleWallY40   = 2094.63
	o1OracleBoxTop    = 923.937
	o1OracleBandArm   = 354.112

	// o1PerFaceRelTol covers the TEN faces OCCT's corner-patch approximation barely moves. Measured worst: the
	// boss wall at 1.29e-4 (its window's mid-rail IS one of the two boundaries the approximation displaces, by
	// up to 0.143 over a ~15-long rail — hence 4.94 of its 38280); every other one of the ten is under 3.2e-5.
	// 3.9× headroom. A wrong station or a wrong contact rail on any arm shows up here at percent scale, so this
	// is the tier that actually discriminates.
	o1PerFaceRelTol = 5e-4
	// o1BandArmRelTol covers the CONVEX BAND arm, whose near boundary IS the patch's u=1 locus. Ours is the
	// exact contact locus; OCCT's is displaced up to 0.162 (3.2% of r) along the band's own tube, which costs
	// 0.915 of the face's 354 — measured 2.58e-3, so 2.3× headroom. It cannot be tightened without asserting
	// OCCT's approximation error as truth.
	o1BandArmRelTol = 6e-3
	// o1PatchVsOracleRelTol covers the CORNER PATCH against DRAWEXE. Our exact envelope integrates to 74.0910
	// against OCCT's 72.1806 (+2.64%), which is OCCT's approximation, not ours (see this file's ★ note and
	// TestO1CornerPatchIsTheExactRollingBallEnvelope, which is the real gate on this face's shape). Kept as a
	// recorded RELATION rather than a tight pin; it still separates the class of defect that mattered on N4 —
	// a chord-projected corner fill, which came out 27% short — by more than 5×.
	o1PatchVsOracleRelTol = 0.05
	// o1PatchExactAreaMM2 is the CONVERGED surface integral of the exact rolling-ball envelope over this
	// corner, computed independently of the kernel by closed-form quadrature of
	// X(α,θ) = C(α) + r(cos θ·ê₁ + sin θ·ê₂) with C(α) the ρ-cylinder ∩ 2r-tube intersection curve
	// (74.0890 at 100², 74.09086 at 400², 74.09095 at 800²). It is the value our patch's own surface integral
	// must reproduce, and it is what makes o1PatchVsOracleRelTol's looseness safe: the tight gate on this face
	// is against the exact envelope, not against OCCT's approximation of it.
	o1PatchExactAreaMM2    = 74.0910
	o1PatchExactAreaRelTol = 2e-4
	// o1MeshSanityRelTol checks the patch's MESH against the same exact value, and does double duty. Its first
	// job is the one n4MeshSanityRelTol has: keep the surface-integral premise honest, i.e. that the face really
	// is trimmed to its whole parameter domain. Its second is to guard the FOLD (o1CanalRailPieces): a folded
	// tessellation double-counts its overlap, and the four-fold mesh this case used to produce read 74.7494
	// against the surface's 74.0910 (+0.89%). Fold-free it reads 74.0880 (−4e-5), so 2e-3 leaves 50× headroom
	// for tessellator-density drift while still failing 4.4× before a re-introduced fold.
	o1MeshSanityRelTol = 2e-3
)

// assertO1FacesMatchOraclePerFace reconciles all TWELVE faces against the DRAWEXE receipt above, at the three
// documented tolerance tiers. Nine faces are pinned tightly; the three the oracle's own patch approximation
// moves (the patch, the band arm, and the boss wall whose window the patch rail bounds) carry their measured
// bands, and the patch's real gate is the exactness test below.
func assertO1FacesMatchOraclePerFace(t *testing.T, body *topo.Body) {
	t.Helper()
	tight := []struct {
		what string
		got  float64
		want float64
	}{
		{"O1 boss wall, r=50 (oracle result_1)", faceMeshArea2(o1BossWallFace(t, body)), o1OracleBossWall},
		{"O1 concave cylinder arm (oracle result_3)", faceMeshArea2(o1ArmFace(t, body, math.V3(0, 0, 1))), o1OracleCylArm},
		{"O1 cove torus arm, major R+r=55 (oracle result_6)", faceMeshArea2(o1CoveArmFace(t, body)), o1OracleCoveArm},
		{"O1 box bottom z=20, grown (oracle result_2)", o1PlaneAreaAt(t, body, math.V3(0, 0, 1), 20), o1OracleBoxBottom},
		{"O1 box top z=90, clipped (oracle result_11)", o1PlaneAreaAt(t, body, math.V3(0, 0, 1), 90), o1OracleBoxTop},
		{"O1 cylinder top disc z=130 (oracle result_7)", o1PlaneAreaAt(t, body, math.V3(0, 0, 1), 130), o1OracleDisc},
		{"O1 cylinder bottom disc z=0 (oracle result_8)", o1PlaneAreaAt(t, body, math.V3(0, 0, 1), 0), o1OracleDisc},
		{"O1 box wall x=50, grown (oracle result_4)", o1PlaneAreaAt(t, body, math.V3(1, 0, 0), 50), o1OracleWallX50},
		{"O1 box wall x=80, receded (oracle result_9)", o1PlaneAreaAt(t, body, math.V3(1, 0, 0), 80), o1OracleWallX80},
		{"O1 box wall y=−40, receded (oracle result_10)", o1PlaneAreaAt(t, body, math.V3(0, 1, 0), -40), o1OracleWallY40},
	}
	for _, c := range tight {
		assertFaceAreaAgainstOracle(t, c.what, c.got, c.want, o1PerFaceRelTol)
	}
	assertFaceAreaAgainstOracle(t, "O1 convex planar band arm (oracle result_12)",
		faceMeshArea2(o1ArmFace(t, body, math.V3(0, 1, 0))), o1OracleBandArm, o1BandArmRelTol)
	patch := o1CornerPatchFace(t, body)
	assertFaceAreaAgainstOracle(t, "O1 corner patch SURFACE integral vs the EXACT rolling-ball envelope",
		bsplinePatchSurfaceArea(patch), o1PatchExactAreaMM2, o1PatchExactAreaRelTol)
	assertFaceAreaAgainstOracle(t, "O1 corner patch SURFACE integral vs DRAWEXE result_5 (recorded relation)",
		bsplinePatchSurfaceArea(patch), o1OracleCornerPad, o1PatchVsOracleRelTol)
	assertFaceAreaAgainstOracle(t, "O1 corner patch MESH (loose sanity + full-domain-trim guard only)",
		faceMeshArea2(patch), o1PatchExactAreaMM2, o1MeshSanityRelTol)
}

// assertO1BossWallCarriesInnerLoop pins the topology detail DRAWEXE reports as 13 wires for 12 faces: the
// boss wall is the ONE face with two loops, because the protruding box cuts a window in it and the corner
// weld re-clips that window rather than the wall's outer boundary. A weld that spliced the wrong loop would
// still be watertight, so this is checked explicitly.
func assertO1BossWallCarriesInnerLoop(t *testing.T, body *topo.Body) {
	t.Helper()
	wires := 0
	for _, f := range body.Faces() {
		wires += len(f.Loops())
	}
	if wires != 13 {
		t.Fatalf("O1 result carries %d loops across 12 faces, want the oracle's 13 (the boss wall's inner window)", wires)
	}
	if got := len(o1BossWallFace(t, body).Loops()); got != 2 {
		t.Fatalf("O1 boss wall has %d loops, want 2 (its outer cylinder boundary + the corner window)", got)
	}
}

// o1BossWallFace returns the r=50 boss wall — the result's only cylinder of that radius (the three fillet
// arms are r=5).
func o1BossWallFace(t *testing.T, body *topo.Body) *topo.Face {
	t.Helper()
	return o1UniqueFace(t, body, "boss wall (r=50 cylinder)", func(f *topo.Face) bool {
		cyl, ok := f.Geometry().(geom.Cylinder)
		return ok && stdmath.Abs(cyl.Radius-50) < 1e-6
	})
}

// o1ArmFace returns the r=5 cylinder arm whose axis runs along dir — (0,0,1) is the concave wall∧cylinder
// arm, (0,1,0) the convex planar band.
func o1ArmFace(t *testing.T, body *topo.Body, dir math.Vector3) *topo.Face {
	t.Helper()
	return o1UniqueFace(t, body, o1AxisLabel(dir)+" cylinder arm", func(f *topo.Face) bool {
		cyl, ok := f.Geometry().(geom.Cylinder)
		if !ok || stdmath.Abs(cyl.Radius-5) > 1e-6 {
			return false
		}
		return stdmath.Abs(float64(cyl.AxisDir.AsVector().Dot(dir))) > 1-1e-9
	})
}

// o1CoveArmFace returns the cove torus arm: major R+r = 55, minor r = 5.
func o1CoveArmFace(t *testing.T, body *topo.Body) *topo.Face {
	t.Helper()
	return o1UniqueFace(t, body, "cove torus arm (major 55, minor 5)", func(f *topo.Face) bool {
		tor, ok := f.Geometry().(geom.Torus)
		return ok && stdmath.Abs(tor.MajorRadius-55) < 1e-6 && stdmath.Abs(tor.MinorRadius-5) < 1e-6
	})
}

// o1CornerPatchFace returns the corner patch — the result's only BSpline face (every other O1 face is an
// analytic plane, cylinder or torus, per the DRAWEXE face inventory).
func o1CornerPatchFace(t *testing.T, body *topo.Body) *topo.Face {
	t.Helper()
	return o1UniqueFace(t, body, "corner patch (the only BSpline face)", func(f *topo.Face) bool {
		_, ok := f.Geometry().(geom.BSplineSurface)
		return ok
	})
}

// o1PlaneAreaAt returns the mesh area of the plane whose normal is ±n and whose signed offset along n is
// `offset`. That pair identifies every one of O1's seven planar faces uniquely.
func o1PlaneAreaAt(t *testing.T, body *topo.Body, n math.Vector3, offset float64) float64 {
	t.Helper()
	unitN := n.Scale(math.Scalar(1 / float64(n.Length())))
	f := o1UniqueFace(t, body, "plane ⊥ "+o1AxisLabel(n), func(f *topo.Face) bool {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok || stdmath.Abs(float64(pl.Normal().Dot(unitN))) < 1-1e-9 {
			return false
		}
		return stdmath.Abs(float64(pl.Origin.VectorTo(math.P3(0, 0, 0)).Dot(unitN))+offset) < 1e-6
	})
	return faceMeshArea2(f)
}

// o1AxisLabel names an axis direction for a decline message (the exception-message rule wants the offending
// axis, and math.Vector3 has no String()).
func o1AxisLabel(v math.Vector3) string {
	return fmt.Sprintf("axis (%g,%g,%g)", float64(v.X), float64(v.Y), float64(v.Z))
}

// o1UniqueFace returns the single face matching pick, failing when zero or several do — so a face-count or
// face-identity regression names the ambiguity instead of silently measuring the wrong face.
func o1UniqueFace(t *testing.T, body *topo.Body, what string, pick func(*topo.Face) bool) *topo.Face {
	t.Helper()
	var found []*topo.Face
	for _, f := range body.Faces() {
		if pick(f) {
			found = append(found, f)
		}
	}
	if len(found) != 1 {
		t.Fatalf("O1 result carries %d faces matching %s, want exactly 1", len(found), what)
	}
	return found[0]
}
