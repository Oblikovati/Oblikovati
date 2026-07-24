// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

// TestAsymmetricMiterGreens is the whole-body gate on the mixed-radius planar miter greened by
// fillet_miter_asymmetric.go. P9/V9 fillet two box-top edges that SHARE the top face but carry DIFFERENT
// radii (r1 on the x=5 arm, r0.5 on the y=0 arm). The equal-radius miter guard-rejected them ("edges
// meeting at a shared corner must use one radius") because its seam is the shortcut cyl1 ∩ (nF1−nF2 mirror
// plane) — valid only when the two cylinders coincide on that plane, which unequal radii break. The
// asymmetric path samples the TRUE cyl0 ∩ cyl1 intersection, so every seam point lies on BOTH cylinders
// and the two arm faces weld watertight. This asserts, on the REAL STEP bodies, the oracle's exact 8-face
// count AND whole-body area within the corpus deps — the regression a wrong seam (non-shared points, a
// non-manifold weld, or a mis-run-out terminus) would fail loud. Grounded in DRAWEXE 8.0.0:
//
//	box s 0 0 0 5 5 5 ; explode s e ; blend result s 1 s_6 0.5 s_10 ; nbshapes ; sprops
//	→ FACE 8, valid solid, Mass 146.393.
func TestAsymmetricMiterGreens(t *testing.T) {
	for _, tc := range []struct {
		name     string
		faces    int
		occtArea float64
	}{
		{"P9", 8, 146.393},
		{"V9", 8, 146.393},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			assertWatertight(t, tc.name, body, tc.faces)
			assertWholeBodyArea(t, tc.name, body, tc.occtArea)
		})
	}
}
