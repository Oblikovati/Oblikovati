// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// K1 and Z1 (simple grid) are the re-survey's bore-lip rim pair, both FAIL(faulty) "concave rim fillet (a
// bore lip) is not yet supported" before fillet_rim_concave.go's rimWithCapOrientation: solveRim's raw
// pl.Normal() (fillet_rim.go) never accounted for capF.Reversed(), so its material-side probe silently
// assumed the cap's stored normal already pointed outward.
//
//   - K1 (cylinder radius 30, r=5) is a genuine bore lip: the plate material sits OUTSIDE the bore, so the
//     rolling ball is tangent to the wall from the material side at R+r=35 — the CONCAVE mirror.
//   - Z1 (cylinder radius 10, r=1) turned out NOT to be a bore lip at all: it is a plain CONVEX rim (like
//     I9) whose cap face happens to be stored bottom-up (a tower standing on its base, capF.Reversed()==
//     true) — the SAME cap-orientation fix resolves it at R−r=9, not R+r. Both symptoms — a probe that
//     fails because the radius sign is wrong, and one that fails because the offset direction is wrong —
//     surface through the identical "probe landed outside the body" reject, so resolveRim tries the
//     orientation-corrected R−r geometry before concluding a rim is genuinely concave (fillet_rim.go).
const (
	k1WholeArea = 72801.3 // OCCT tests/blend/simple/K1 whole-result area
	z1WholeArea = 2485.86 // OCCT tests/blend/simple/Z1 whole-result area
	k1z1Deps    = 0.01    // the K1/Z1 corpus records' deps tolerance
)

// TestK1ConcaveBoreLipWatertight asserts the bore-lip mirror builds a watertight manifold solid — every
// edge 2-incident, valid + closed + holes-contained + IsSolid — with the oracle's 8 faces, and that its
// torus band's major radius is R+r=35 (cylinder radius 30, r=5), the CONCAVE mirror of solveRim's R−r.
func TestK1ConcaveBoreLipWatertight(t *testing.T) {
	t.Parallel()
	body := caseResultBody(t, "K1")
	assertWatertight(t, "K1", body, 8)
	assertRimTorusMajor(t, "K1", body, 35)
}

// TestZ1ConvexReversedCapWatertight asserts the cap-orientation-corrected CONVEX retry builds a
// watertight manifold solid with the oracle's 6 faces, and that its torus band's major radius is
// R−r=9 (cylinder radius 10, r=1) — Z1 is not actually a bore lip, so this must NOT be R+r=11.
func TestZ1ConvexReversedCapWatertight(t *testing.T) {
	t.Parallel()
	body := caseResultBody(t, "Z1")
	assertWatertight(t, "Z1", body, 6)
	assertRimTorusMajor(t, "Z1", body, 9)
}

// assertRimTorusMajor fails unless the body carries exactly one torus face with the expected major radius.
func assertRimTorusMajor(t *testing.T, name string, body *topo.Body, wantMajor float64) {
	t.Helper()
	tor, count := geom.Torus{}, 0
	for _, f := range body.Faces() {
		if g, ok := f.Geometry().(geom.Torus); ok {
			tor, count = g, count+1
		}
	}
	if count != 1 {
		t.Fatalf("%s has %d torus faces, want exactly 1 (the rim band)", name, count)
	}
	if tor.MajorRadius != wantMajor {
		t.Errorf("%s torus major radius = %g, want %g", name, tor.MajorRadius, wantMajor)
	}
}

// TestK1Z1TessellationFoldGate meshes every K1/Z1 face, asserts each is fold-free with a finite positive
// area, and that the summed mesh area equals OCCT's whole-result area within the corpus deps — mirroring
// TestI9Arc3dRimTessellationFoldGate / TestJ1ClosedRimTessellationFoldGate for the same rim-band assembly.
func TestK1Z1TessellationFoldGate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		want float64
	}{{"K1", k1WholeArea}, {"Z1", z1WholeArea}} {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			meshTotal := 0.0
			for _, f := range body.Faces() {
				m := tessellate.TessellateFace(f, ops.PropertyQuality())
				area := ops.MeshArea(m)
				meshTotal += area
				if area <= 0 || stdmath.IsInf(area, 0) || stdmath.IsNaN(area) {
					t.Fatalf("%s %T face meshed to %.4f, want a finite positive area", tc.name, f.Geometry(), area)
				}
				assertFaceFoldFreeAtEveryQuality(t, tc.name, f, m)
			}
			if rel := stdmath.Abs(meshTotal-tc.want) / tc.want; rel > k1z1Deps {
				t.Fatalf("%s total mesh area %.4f, want OCCT %.2f within deps %.2f (rel %.6f)", tc.name, meshTotal, tc.want, k1z1Deps, rel)
			}
			if vol := query.BodyGeometryProperties(body, ops.PropertyQuality()).Volume; vol <= 0 {
				t.Fatalf("%s result volume %.4f, want positive", tc.name, vol)
			}
		})
	}
}
