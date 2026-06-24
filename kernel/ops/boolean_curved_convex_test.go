// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// The user-facing ops.Boolean(Intersect, curvedSolid, box) must now take the EXACT curved path (compose
// HalfSpaceCut over the box planes), not the triangle-soup CSG — analytic faces preserved, analytic
// volume (M2 Phase 1, Oblikovati/Oblikovati#1334, acceptance #1).

// TestBooleanIntersectSphereBoxExact intersects a sphere with a box that trims its +x and lower-z sides.
// The box's other four planes clear the sphere (kept whole); only x=2 and z=0 cut.
func TestBooleanIntersectSphereBoxExact(t *testing.T) {
	const R = 5.0
	sphere, _ := brep.SolidSphere(math.P3(0, 0, 0), R, "s")
	box, _ := brep.SolidBlock(math.P3(-10, -10, -10), math.P3(2, 10, 0), "box") // keeps x ≤ 2, z ≤ 0

	res, err := ops.Boolean(ops.Intersect, sphere, box)
	if err != nil {
		t.Fatalf("Boolean(Intersect): %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !r.Closed || !r.Manifold || !res.IsSolid() {
		t.Fatalf("sphere∩box is not a valid closed manifold solid: %+v", r)
	}
	// EXACT path keeps analytic surfaces — a CSG result would be all tessellated planar triangles.
	spheres := 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Sphere:
			spheres++
		case geom.Plane:
		default:
			t.Errorf("face surface %T is not analytic (the exact path must be taken, not CSG)", f.Geometry())
		}
	}
	if spheres != 1 {
		t.Errorf("result has %d sphere faces, want 1 (the curved cap survived as exact geometry)", spheres)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	capV := stdmath.Pi * 9 * (3*R - 3) / 3 // x>2 spherical cap (height 3); its z<0 half is removed
	want := (2.0/3.0)*stdmath.Pi*R*R*R - capV/2
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("sphere∩box volume %.4f, want %.4f (analytic) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestBooleanIntersectSphereContainedBoxUnaffected: a box fully inside the sphere intersects to the box
// (handled by classify before the curved path) — a sanity check that the wiring did not disturb it.
func TestBooleanIntersectSphereContainedBoxUnaffected(t *testing.T) {
	sphere, _ := brep.SolidSphere(math.P3(0, 0, 0), 5, "s")
	box, _ := brep.SolidBlock(math.P3(-1, -1, -1), math.P3(1, 1, 1), "box") // corner dist √3 < 5: inside
	res, err := ops.Boolean(ops.Intersect, sphere, box)
	if err != nil {
		t.Fatalf("Boolean(Intersect): %v", err)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	if stdmath.Abs(got-8) > 1e-6 { // the 2×2×2 box
		t.Errorf("sphere ∩ contained box volume %.6f, want 8", got)
	}
}
