// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// TestM8MixedCurvedCornerWatertight is the per-case whole-body gate for OCCT tests/blend/simple/M8 — the
// mixed-sense CURVED-host trihedral corner (fillet_curved_mixed_weld.go). M8 is a box + boss (R=25) whose
// single trihedral vertex joins ONE convex Cyl∧Plane fillet + a concave cove torus arm (major R+r=30) + a
// planar Plane∧Plane arm. No single ball is tangent to the boss wall at both R−r and R+r, so the corner is
// NOT a sphere: OCCT builds an analytic 2r-TORUS patch (major R=2r=10, minor r=5). This asserts — beyond
// the area-only scoreboard — that the result is a watertight 14-face solid carrying exactly that 2r-torus
// corner (a regression to the sphere path, or a cracked weld, fails loud here) plus OCCT's whole-body area.
func TestM8MixedCurvedCornerWatertight(t *testing.T) {
	body := caseResultBody(t, "M8")
	assertWatertight(t, "M8", body, 14)
	assertWholeBodyFoldFree(t, "M8", body)
	assertWholeBodyArea(t, "M8", body, 67959.5)
	assertM8CornerTorus(t, body)
}

// assertM8CornerTorus checks the body carries the derived 2r-torus corner patch — major R=2r=10, minor
// r=5 — distinct from the cove ARM torus (major R+r=30), and that its mesh area matches the analytic
// quarter-tube U·r·(R·π/2−r) (U = the 104.5° top-contact major span), not the mesher alone. A re-forced
// sphere leaves zero 2r-tori; a mis-sized/mis-wound patch fails on the radii or the area.
func assertM8CornerTorus(t *testing.T, body *topo.Body) {
	t.Helper()
	found := 0
	for _, f := range body.Faces() {
		tor, ok := f.Geometry().(geom.Torus)
		if !ok || stdmath.Abs(tor.MajorRadius-10) > 1e-3 {
			continue // skip the cove arm torus (major 30) and every non-torus face
		}
		found++
		if stdmath.Abs(tor.MinorRadius-5) > 1e-3 {
			t.Fatalf("M8 corner torus minor=%.6f, want r=5", tor.MinorRadius)
		}
		u := 1.8237740235 // arc(d) top-contact major span (104.5°), from the DRAWEXE-verified solve
		want := u * tor.MinorRadius * (tor.MajorRadius*stdmath.Pi/2 - tor.MinorRadius)
		if a := faceMeshArea2(f); stdmath.Abs(a-want) > 0.01*want {
			t.Fatalf("M8 corner torus area %.4f, want ≈%.4f =U·r·(R·π/2−r) (a sphere/complement reads far off)", a, want)
		}
	}
	if found != 1 {
		t.Fatalf("M8 has %d R=2r corner tori, want exactly 1 mixed-sense 2r-torus patch (a re-forced sphere reads 0)", found)
	}
}
