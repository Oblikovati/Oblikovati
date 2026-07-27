// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestL1L7CornerSetbackWatertight is the per-case whole-body gate for the three OCCT tests/blend/simple
// cases the P1 dihedral corner-setback pass touches: L1 (100³ box + 40³ boss, 4 orthogonal concave
// dihedral miters) and L7 (a 10°-rotated boss whose miter corners join a rotated-horizontal base edge
// with a vertical edge — still θ=90° in 3D, so the same orthogonal setback closes them), both RED→GREEN;
// plus N5 (a rotated boss with two orthogonal concave dihedral miters) which was GREEN by tolerance at
// base and the pass legitimately RE-WELDS toward OCCT (area rel 0.3337%→0.1043%). It asserts, WITHOUT
// relying on the area-only TestOCCTBlendSimple, that each set-back result is a watertight fold-free solid
// with OCCT's face count AND (L1) the derivation's per-face host-plane areas — the crux the pass fixes
// (the pre-P1 reflected-seam body over-kept +3200 of host material: top 9100 vs 7500, walls 1800 vs
// 1400). A regression that drops the setback, or mis-sides the seam, fails one of these loud.
func TestL1L7CornerSetbackWatertight(t *testing.T) {
	for _, tc := range []struct {
		name     string
		faces    int
		area     float64 // OCCT checkprops whole-body reference area
		topPlane float64 // re-trimmed box-top plane area (L1 only; 0 = skip — L7/N5 hosts are rotated)
		wall     float64 // each re-trimmed boss-wall plane area (L1 only)
		walls    int     // number of boss-wall planes expected at `wall`
	}{
		{"L1", 15, 66070.8, 7500, 1400, 4},
		{"L7", 15, 65635.1, 0, 0, 0},
		{"N5", 11, 64525.7, 0, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			assertWatertight(t, tc.name, body, tc.faces)
			assertWholeBodyFoldFree(t, tc.name, body)
			assertWholeBodyArea(t, tc.name, body, tc.area)
			if tc.topPlane > 0 {
				assertL1HostPlanes(t, body, tc.topPlane, tc.wall, tc.walls)
			}
		})
	}
}

// assertWholeBodyFoldFree checks the body's Property-quality WHOLE-BODY tessellation has no fold edge (an
// interior mesh edge shared by more than two triangles), the invariant that it bounds a well-defined
// volume — a set-back seam that mis-welds two bands would fold here.
func assertWholeBodyFoldFree(t *testing.T, name string, body *topo.Body) {
	t.Helper()
	for _, gq := range gateQualities() {
		mesh, _ := ops.TessellateBody(body, gq.q)
		if f := ops.FoldEdgeCount(mesh); f != 0 {
			t.Fatalf("%s tessellation has %d fold edges at %s quality, want 0 (a set-back seam that mis-welds folds here)",
				name, f, gq.name)
		}
	}
}

// assertL1HostPlanes checks L1's re-trimmed host planes match the first-principles derivation: the box
// top z=100 recedes to a 50×50 hole (area 7500, not the reflected-seam 9100) and each of the four boss
// walls shrinks to z∈[105,140] (area 1400, not the reflected-seam 1800). Planar faces tessellate exactly
// from their trimmed boundary (holes subtracted), so a mis-set-back host fails to the digit.
func assertL1HostPlanes(t *testing.T, body *topo.Body, topPlane, wall float64, walls int) {
	t.Helper()
	sawTop := false
	gotWalls := 0
	for _, f := range body.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok {
			continue
		}
		a := faceMeshArea2(f)
		if isBoxTopPlane(pl) {
			sawTop = true
			if stdmath.Abs(a-topPlane) > 0.01 {
				t.Fatalf("L1 box-top plane area %.4f, want %.1f (a dropped setback over-keeps 9100)", a, topPlane)
			}
		}
		if isBossWallPlane(pl) && stdmath.Abs(a-wall) < 0.01 {
			gotWalls++
		}
	}
	if !sawTop {
		t.Fatalf("L1 result carries no box-top plane at z=100")
	}
	if gotWalls != walls {
		t.Fatalf("L1 has %d boss-wall planes at area %.0f, want %d (a dropped setback leaves them at 1800)", gotWalls, wall, walls)
	}
}

// isBoxTopPlane reports whether pl is the box top face at z=100 with an upward normal.
func isBoxTopPlane(pl geom.Plane) bool {
	n := pl.Normal()
	return stdmath.Abs(n.Z) > 0.99 && stdmath.Abs(pl.Origin.Z-100) < 0.01
}

// isBossWallPlane reports whether pl is a vertical face (horizontal normal). A vertical plane's Origin.Z
// is not fixed by the plane, so it is not gated on z here; the area≈1400 filter at the call site
// distinguishes the four boss walls (40×35) from the 10000 box side walls.
func isBossWallPlane(pl geom.Plane) bool {
	return stdmath.Abs(pl.Normal().Z) < 0.01
}
