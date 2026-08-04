// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
)

// The T-N7.0 seam test bed. It builds the clean B3 octant through the REAL corner-solve path
// (b3CornerArms → solveCurvedCorner) so no test invents an impossible weld — the cautionary tale
// that motivated this re-scoped slice. Every extractCurvedCorner assertion drives this one fixture.

// b3CornerWeld solves the certified B3 octant (three equal-r arms: torus W∧K, cyl W∧N, planar-cyl K∧N)
// and returns the solved cornerWeld, the arms, the corner sphere, and the model resolution.
func b3CornerWeld(t *testing.T) (cornerWeld, []edgeFillet, geom.Sphere, Resolution) {
	t.Helper()
	sphere, arms := b3CornerArms(t)
	res := geom.ResolutionForSize(150)
	w, ok := solveCurvedCorner(sphere, arms, res)
	if !ok {
		t.Fatalf("solveCurvedCorner rejected the certified B3 corner (fixture broken)")
	}
	return w, arms, sphere, res
}

// TestExtractCurvedCorner_OctantIsThreeValentSphere is the seam's core gate: the clean B3 octant must
// extract as a CLOSED 3-valence RailLoop whose three concentric equal-r setback great-arcs the
// analytic-sphere tier wins (over tri3, by tier order), and the recognized sphere must equal the
// solved corner sphere. This proves the octant surface routes through the engine and stays a sphere.
func TestExtractCurvedCorner_OctantIsThreeValentSphere(t *testing.T) {
	w, arms, sphere, res := b3CornerWeld(t)
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok || loop.Valence() != 3 {
		t.Fatalf("extractCurvedCorner: want a closed 3-side octant loop; ok=%v valence=%d", ok, loop.Valence())
	}
	if !loop.Closed(res.Weld() * w.radius) {
		t.Fatalf("octant RailLoop is not closed")
	}
	patch, ok := resolveBlend(loop, res)
	if !ok || patch.Kind != BlendKindSphere {
		t.Fatalf("octant must resolve to the analytic sphere tier; ok=%v kind=%q", ok, patch.Kind)
	}
	assertSameSphere(t, patch.Surface, sphere, res.Weld()*w.radius)
}

// TestExtractCurvedCorner_SidesCarryArmAdjacents asserts each Side welds against its ARM surface (not
// the sphere): the ACL reads arms[i].armSurface for Adjacent (the choice that makes the G1 ribbons
// twist-compatible for the later coons4 tier), and every arm surface appears exactly once.
func TestExtractCurvedCorner_SidesCarryArmAdjacents(t *testing.T) {
	w, arms, _, res := b3CornerWeld(t)
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok {
		t.Fatalf("extractCurvedCorner declined the certified B3 octant")
	}
	for _, s := range loop.Sides {
		if s.Cont != G1 {
			t.Fatalf("Side.Cont = %v, want G1 (every arm weld is tangent)", s.Cont)
		}
		if s.Adjacent == nil {
			t.Fatalf("Side.Adjacent is nil — the arm surface was not attached")
		}
	}
	if got := countDistinctAdjacents(loop); got != len(arms) {
		t.Fatalf("distinct Side.Adjacent surfaces = %d, want %d (one per arm)", got, len(arms))
	}
}

// countDistinctAdjacents counts the distinct arm surfaces across the loop's Sides (each arm appears once).
func countDistinctAdjacents(loop RailLoop) int {
	seen := map[geom.Surface]bool{}
	for _, s := range loop.Sides {
		seen[s.Adjacent] = true
	}
	return len(seen)
}

// assertSameSphere asserts got is a geom.Sphere whose centre and radius match want within tol — pass
// res.Weld()·radius (the same model-relative weld epsilon extractCurvedCorner uses to close its
// RailLoop, not a bare literal), since the engine's circumcentre-recovered sphere carries only ~1e-12
// FP noise vs the solved exact-centre sphere and any model-scale-derived tol comfortably covers that.
func assertSameSphere(t *testing.T, got geom.Surface, want geom.Sphere, tol float64) {
	t.Helper()
	sph, ok := got.(geom.Sphere)
	if !ok {
		t.Fatalf("recognized surface is %T, want geom.Sphere", got)
	}
	if d := sph.Center.DistanceTo(want.Center); float64(d) > tol {
		t.Fatalf("recognized sphere centre %v off solved %v by %.3e (tol %.3e)", sph.Center, want.Center, d, tol)
	}
	if e := stdmath.Abs(sph.Radius - want.Radius); e > tol {
		t.Fatalf("recognized sphere radius %.9f, want solved %.9f (Δ=%.3e, tol %.3e)", sph.Radius, want.Radius, e, tol)
	}
}
