// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// notchedCylinderBody is a r=3 h=10 cylinder whose top rim was clipped by the oblique plane x+z=9.5 — a
// genuine already-cut side (one full bottom circle + a notched top boundary), the partial-rim target.
func notchedCylinderBody(t *testing.T) *topo.Body {
	t.Helper()
	bare, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("bare cylinder: %v", err)
	}
	pl, _ := geom.NewPlane(math.P3(1.5, 0, 8), math.V3(1, 0, 1))
	notched, err := brep.HalfSpaceCut(bare, pl)
	if err != nil {
		t.Fatalf("first cut (notch): %v", err)
	}
	return notched
}

func freeEdgeCount(b *topo.Body) int {
	n := 0
	for _, e := range b.Edges() {
		if len(e.Faces()) != 2 {
			n++
		}
	}
	return n
}

// TestPartialRimCutDisjointRodConservesRemoval is the headline #1732 certification: a second cut (a rod drilled
// through the still-full lower band, disjoint from the top notch) on an already-cut cylinder builds an EXACT
// analytic solid, not a CSG fallback. It is watertight, Euler-valid, and genus-1 (the through-tunnel). Shape is
// certified by CONSERVATION: because the rod is disjoint from the notch, it removes the identical plug from the
// notched target as from a bare cylinder — cross-checked against the certified RuledCrossingCutGeneral, so a
// right-volume-wrong-shape build (which the manifold/Euler checks miss) is caught.
func TestPartialRimCutDisjointRodConservesRemoval(t *testing.T) {
	t.Parallel()
	target := notchedCylinderBody(t)
	rod, _ := brep.SolidCylinder(math.P3(-6, 0, 3), math.V3(1, 0, 0), 1, 12) // through the lower band, clear of the notch
	res, ok := brep.PartialRimCutGeneral(target, rod, &diag.Recorder{})
	if !ok || res == nil {
		t.Fatal("partial-rim disjoint cut declined; want an exact analytic solid")
	}
	if r := ops.Validate(res); !r.Valid {
		t.Fatalf("partial-rim result is not a valid solid: %v", r.Issues)
	}
	if fe := freeEdgeCount(res); fe != 0 {
		t.Errorf("partial-rim result has %d free edges; want a watertight solid", fe)
	}
	if chi := res.EulerCharacteristic(); chi != 0 {
		t.Errorf("partial-rim result chi=%d; want 0 (genus-1 through-tunnel)", chi)
	}

	// Shape gate (mesh-independent, the #1724 right-volume-wrong-shape check): the result must enclose exactly
	// the analytic region {inside the notched target} ∧ {outside the rod}. A 96³ membership grid gives the
	// smooth analytic volume; the tessellated result sits below it by the faceted-cylinder + SSI-polyline
	// deficit — which is 2.81% for the ALREADY-CERTIFIED RuledCrossingCutGeneral against this same oracle, so
	// the 3.5% budget here is that established deficit, not slack. A right-Euler wrong-shape build diverges far more.
	analytic := analyticPartialRimVolume(96)
	if rel := stdmath.Abs(vol(res)-analytic) / analytic; rel > 0.035 {
		t.Errorf("result volume %.4f vs analytic {target ∧ ¬rod} %.4f (%.2f%%); want the exact cut region", vol(res), analytic, rel*100)
	}

	// Tight volume gate (mesh-consistent, faceting cancels): because the rod is disjoint from the notch it
	// removes the identical plug from the notched target as from a bare cylinder (the certified crossing cut). The
	// two removed volumes share the same faceted cylinder and tunnel tessellation, so they must agree to well
	// under 1% of the body — a far tighter bound than the smooth-oracle gate, catching any volume corruption the
	// partial-rim machinery might introduce.
	bare, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	rod2, _ := brep.SolidCylinder(math.P3(-6, 0, 3), math.V3(1, 0, 0), 1, 12)
	bareRes, okB := brep.RuledCrossingCutGeneral(bare, rod2, nil)
	if !okB {
		t.Fatal("bare crossing-cut oracle declined; cannot cross-check the removed plug")
	}
	removedNotched := vol(notchedCylinderBody(t)) - vol(res)
	removedBare := vol(mustBareCylinder(t)) - vol(bareRes)
	if rel := stdmath.Abs(removedNotched-removedBare) / vol(res); rel > 0.01 {
		t.Errorf("removed plug: notched=%.5f bare=%.5f, differ by %.2f%% of the body; a disjoint cut removes the same plug",
			removedNotched, removedBare, rel*100)
	}
}

func mustBareCylinder(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("bare cylinder: %v", err)
	}
	return b
}

// analyticPartialRimVolume integrates the exact partial-rim region — inside the r=3 h=10 cylinder, below the
// notch plane x+z<=9.5, and outside the r=1 rod on the x-axis at (y,z)=(0,3) — on an n³ grid over the target's
// bounding box. Mesh-independent, so it certifies the built solid's SHAPE, not just its Euler characteristic.
func analyticPartialRimVolume(n int) float64 {
	inTarget := func(x, y, z float64) bool { return x*x+y*y < 9 && z > 0 && z < 10 && x+z < 9.5 }
	inRod := func(x, y, z float64) bool { return y*y+(z-3)*(z-3) < 1 && x > -6 && x < 6 }
	count := 0
	for i := range n {
		x := -3 + 6*(float64(i)+0.5)/float64(n)
		for j := range n {
			y := -3 + 6*(float64(j)+0.5)/float64(n)
			for k := range n {
				z := 10 * (float64(k) + 0.5) / float64(n)
				if inTarget(x, y, z) && !inRod(x, y, z) {
					count++
				}
			}
		}
	}
	return float64(count) / float64(n*n*n) * (6 * 6 * 10)
}

// TestPartialRimCutInteractingCutDeclines: a second cut whose imprint enters the removed notch (a rod through
// the upper band where the front wall is gone) is OUTSIDE the disjoint sub-family, so PartialRimCutGeneral
// declines and kernel/ops keeps its observable CSG fallback — never a manifold-but-wrong analytic solid.
func TestPartialRimCutInteractingCutDeclines(t *testing.T) {
	t.Parallel()
	target := notchedCylinderBody(t)
	rod, _ := brep.SolidCylinder(math.P3(-6, 0, 7), math.V3(1, 0, 0), 1, 12) // z=7: front exit lands in the notch
	if _, ok := brep.PartialRimCutGeneral(target, rod, &diag.Recorder{}); ok {
		t.Fatal("an interacting second cut (imprint crossing the notch) must decline, not build a wrong solid")
	}
}
