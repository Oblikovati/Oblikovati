// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// The user-facing ops.Boolean(Intersect, curvedSolid, box) must now take the EXACT curved path (compose
// HalfSpaceCut over the box planes), not the triangle-soup CSG — analytic faces preserved, analytic
// volume (M2 Phase 1, Oblikovati/Oblikovati#1334, acceptance #1).

// TestBooleanIntersectSphereBoxExact intersects a sphere with a box that trims its +x and lower-z sides.
// The box's other four planes clear the sphere (kept whole); only x=2 and z=0 cut.
func TestBooleanIntersectSphereBoxExact(t *testing.T) {
	t.Parallel()
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
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	capV := stdmath.Pi * 9 * (3*R - 3) / 3 // x>2 spherical cap (height 3); its z<0 half is removed
	want := (2.0/3.0)*stdmath.Pi*R*R*R - capV/2
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("sphere∩box volume %.4f, want %.4f (analytic) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestBooleanIntersectCylinderBoxExact intersects an axis-aligned cylinder with a box narrower than its
// diameter: all four box walls clip chords off the cross-section, leaving four exact cylinder arc faces
// at the corners (a "squircle" prism). The box top/bottom clear the cylinder height. The exact path must
// preserve the cylinder surfaces and match the analytic cross-section (disk minus four segments) × height.
func TestBooleanIntersectCylinderBoxExact(t *testing.T) {
	t.Parallel()
	const r, h, a = 3.0, 10.0, 2.5
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	box, _ := brep.SolidBlock(math.P3(-a, -a, -5), math.P3(a, a, 15), "box")

	res, err := ops.Boolean(ops.Intersect, cyl, box)
	if err != nil {
		t.Fatalf("Boolean(Intersect): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("cylinder∩box is not a valid closed manifold solid: %+v", v)
	}
	cylFaces := 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder:
			cylFaces++
		case geom.Plane:
		default:
			t.Errorf("face surface %T is not analytic (the exact path must be taken, not CSG)", f.Geometry())
		}
	}
	if cylFaces != 4 {
		t.Errorf("result has %d cylinder faces, want 4 (one exact arc per clipped corner)", cylFaces)
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	segs := r*r*stdmath.Acos(a/r) - a*stdmath.Sqrt(r*r-a*a) // one minor segment beyond a wall
	want := (stdmath.Pi*r*r - 4*segs) * h
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("cylinder∩box volume %.4f, want %.4f (analytic) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestBooleanIntersectSphereContainedBoxUnaffected: a box fully inside the sphere intersects to the box
// (handled by classify before the curved path) — a sanity check that the wiring did not disturb it.
func TestBooleanIntersectSphereContainedBoxUnaffected(t *testing.T) {
	t.Parallel()
	sphere, _ := brep.SolidSphere(math.P3(0, 0, 0), 5, "s")
	box, _ := brep.SolidBlock(math.P3(-1, -1, -1), math.P3(1, 1, 1), "box") // corner dist √3 < 5: inside
	res, err := ops.Boolean(ops.Intersect, sphere, box)
	if err != nil {
		t.Fatalf("Boolean(Intersect): %v", err)
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	if stdmath.Abs(got-8) > 1e-6 { // the 2×2×2 box
		t.Errorf("sphere ∩ contained box volume %.6f, want 8", got)
	}
}
