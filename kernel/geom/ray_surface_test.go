// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// rayFrom builds a ray through origin along dir; the test dirs are already unit or axis-aligned.
func rayFrom(t *testing.T, ox, oy, oz, dx, dy, dz float64) Line {
	t.Helper()
	ray, err := NewLine(math.P3(ox, oy, oz), math.V3(dx, dy, dz))
	if err != nil {
		t.Fatalf("NewLine: %v", err)
	}
	return ray
}

// hitParams extracts just the ordered T values, the quantity every crossing-count classifier reads.
func hitParams(hits []RayHit) []float64 {
	ts := make([]float64, len(hits))
	for i, h := range hits {
		ts[i] = h.T
	}
	return ts
}

func wantParams(t *testing.T, hits []RayHit, want ...float64) {
	t.Helper()
	got := hitParams(hits)
	if len(got) != len(want) {
		t.Fatalf("hit parameters = %v, want %v", got, want)
	}
	for i := range want {
		if !near(got[i], want[i]) {
			t.Errorf("hit[%d] T = %g, want %g (all: %v)", i, got[i], want[i], got)
		}
	}
}

// A +Z ray pierces a horizontal plane once at its height; a ray parallel to the plane never
// crosses it.
func TestRayPlaneHitAndParallel(t *testing.T) {
	t.Parallel()
	pl := mustPlane(t, 0, 0, 5, 0, 0, 1)
	wantParams(t, RaySurfaceHits(pl, rayFrom(t, 0, 0, 0, 0, 0, 1), stdmath.Inf(1)), 5)
	if hits := RaySurfaceHits(pl, rayFrom(t, 0, 0, 0, 1, 0, 0), stdmath.Inf(1)); len(hits) != 0 {
		t.Errorf("ray parallel to plane = %v, want no crossing", hitParams(hits))
	}
	// The plane behind the ray origin is not a forward hit.
	if hits := RaySurfaceHits(pl, rayFrom(t, 0, 0, 0, 0, 0, -1), stdmath.Inf(1)); len(hits) != 0 {
		t.Errorf("plane behind the ray = %v, want no forward crossing", hitParams(hits))
	}
}

// A ray through a sphere enters and exits at the two exact radii; a clear ray misses; a tMax
// short of the far wall keeps only the near hit.
func TestRaySphereEnterExitAndClamp(t *testing.T) {
	t.Parallel()
	sp, _ := NewSphere(math.P3(0, 0, 3), 1)
	ray := rayFrom(t, 0, 0, 0, 0, 0, 1)
	wantParams(t, RaySurfaceHits(sp, ray, stdmath.Inf(1)), 2, 4)
	wantParams(t, RaySurfaceHits(sp, ray, 3), 2) // tMax between the two walls
	miss := rayFrom(t, 5, 0, 0, 0, 0, 1)
	if hits := RaySurfaceHits(sp, miss, stdmath.Inf(1)); len(hits) != 0 {
		t.Errorf("ray clear of the sphere = %v, want a miss", hitParams(hits))
	}
}

// A ray perpendicular to a cylinder axis crosses both walls; from the axis only the forward wall
// counts; a ray parallel to the axis grazes no radial band.
func TestRayCylinderWallsAndParallel(t *testing.T) {
	t.Parallel()
	cy, _ := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	wantParams(t, RaySurfaceHits(cy, rayFrom(t, -5, 0, 0, 1, 0, 0), stdmath.Inf(1)), 3, 7) // x=−2, x=+2
	wantParams(t, RaySurfaceHits(cy, rayFrom(t, 0, 0, 0, 1, 0, 0), stdmath.Inf(1)), 2)     // from the axis outward
	if hits := RaySurfaceHits(cy, rayFrom(t, 0.5, 0, 0, 0, 0, 1), stdmath.Inf(1)); len(hits) != 0 {
		t.Errorf("ray parallel to the cylinder axis = %v, want no crossing", hitParams(hits))
	}
}

// A ray across a cone at a fixed height meets the nappe at ±radius; a ray at the mirror height
// (below the apex) meets nothing, because the second algebraic root is on the rejected nappe.
func TestRayConeNappeSelection(t *testing.T) {
	t.Parallel()
	cone, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/4)                     // radius = height
	wantParams(t, RaySurfaceHits(cone, rayFrom(t, -5, 0, 3, 1, 0, 0), stdmath.Inf(1)), 2, 8) // x=−3, x=+3 at z=3
	if hits := RaySurfaceHits(cone, rayFrom(t, -5, 0, -3, 1, 0, 0), stdmath.Inf(1)); len(hits) != 0 {
		t.Errorf("ray at the mirror-nappe height = %v, want no crossing", hitParams(hits))
	}
}

// The torus has no closed-form ray quadric, so it exercises the general numeric path: a radial ray
// meets the tube's inner and outer walls at Major∓Minor.
func TestRayTorusNumericPath(t *testing.T) {
	t.Parallel()
	tor, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	wantParams(t, RaySurfaceHits(tor, rayFrom(t, 0, 0, 0, 1, 0, 0), 20), 3, 7) // inner 5−2, outer 5+2
}

// The numeric path needs a finite bound: an unbounded tMax gives no segment to sample, so it
// declines rather than guessing a range.
func TestRayNumericPathNeedsFiniteBound(t *testing.T) {
	t.Parallel()
	tor, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	if hits := RaySurfaceHits(tor, rayFrom(t, 0, 0, 0, 1, 0, 0), stdmath.Inf(1)); len(hits) != 0 {
		t.Errorf("unbounded numeric ray = %v, want none (no segment to sample)", hitParams(hits))
	}
}

// rayQuadraticRoots degrades correctly: a vanishing leading coefficient falls to the single
// linear root, an all-zero pair has none, and a negative discriminant has none.
func TestRayQuadraticDegenerate(t *testing.T) {
	t.Parallel()
	if r := rayQuadraticRoots(0, 2, -4); len(r) != 1 || !near(r[0], 2) {
		t.Errorf("linear 2x−4 roots = %v, want [2]", r)
	}
	if r := rayQuadraticRoots(0, 0, 5); len(r) != 0 {
		t.Errorf("degenerate constant roots = %v, want none", r)
	}
	if r := rayQuadraticRoots(1, 0, 1); len(r) != 0 {
		t.Errorf("x²+1 (no real root) = %v, want none", r)
	}
	if r := rayQuadraticRoots(1, 0, 0); len(r) != 1 || !near(r[0], 0) {
		t.Errorf("x² (tangent double root) = %v, want [0]", r)
	}
}

// Every hit records surface parameters that reproduce the pierce point through PointAt — the
// property the trim-loop test relies on.
func TestRayHitParamsRoundTrip(t *testing.T) {
	t.Parallel()
	sp, _ := NewSphere(math.P3(0, 0, 3), 1)
	for _, h := range RaySurfaceHits(sp, rayFrom(t, 0, 0, 0, 0, 0, 1), stdmath.Inf(1)) {
		back := sp.PointAt(h.U, h.V)
		if float64(back.DistanceTo(h.Point)) > 1e-9 {
			t.Errorf("PointAt(%g,%g)=%v does not reproduce hit point %v", h.U, h.V, back, h.Point)
		}
	}
}
