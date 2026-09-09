// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A two-colouring says which faces must differ, never which colour is which — flipping a whole
// component is always admissible. That freedom is not harmless. Reversing a face's loops leaves the
// region it denotes alone on an OPEN surface, where the region is the loops' interior either way; on a
// CLOSED one a loop and its reverse bound COMPLEMENTARY regions.
//
// So the sphere patch of a sphere∩box corner came out wound against the solid it bounds, and every
// point of it classified inverted — the south pole outside the face, the north pole inside — while the
// body stayed manifold and closed, because traversal consistency cannot see the difference (ADR-0062).
func TestClosedSurfaceFaceKeepsItsWindingThroughTheReorient(t *testing.T) {
	t.Parallel()
	sphere, err := SolidSphere(math.P3(0, 0, 0), 5, "s")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	// Both cuts as ordinary differences against a block, which is what a half-space cut IS
	// (ADR-0062): keep z ≤ 0, then x ≤ 2.
	overZ, err := SolidBlock(math.P3(-9, -9, 0), math.P3(9, 9, 9), "overz")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	hemi, err := Boolean(Difference, sphere, overZ)
	if err != nil {
		t.Fatalf("Difference (hemisphere): %v", err)
	}
	pastX, err := SolidBlock(math.P3(2, -9, -9), math.P3(9, 9, 9), "pastx")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	corner, err := Boolean(Difference, hemi, pastX)
	if err != nil {
		t.Fatalf("Difference: %v", err)
	}
	face := sphereFaceOfBody(t, corner)
	for _, c := range []struct {
		p    math.Point3
		want bool
		why  string
	}{
		{math.P3(0, 0, -5), true, "the south pole is in the kept patch"},
		{math.P3(-4.9, 0, -1), true, "so is the back of it, below the equator"},
		{math.P3(0, 4.9, -1), true, "and its +y side"},
		{math.P3(4.9, 0, -1), false, "x > 2 was cut away"},
		{math.P3(0, 0, 5), false, "the north pole is the other hemisphere"},
		{math.P3(-4.9, 0, 1), false, "so is the back of it, above the equator"},
	} {
		if got := PointInFaceTrim(face, c.p); got != c.want {
			t.Errorf("PointInFaceTrim(%v) = %v, want %v — %s", c.p, got, c.want, c.why)
		}
	}
}

// sphereFaceOfBody returns the body's single spherical face.
func sphereFaceOfBody(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Sphere); ok {
			return f
		}
	}
	t.Fatal("the body has no spherical face")
	return nil
}
