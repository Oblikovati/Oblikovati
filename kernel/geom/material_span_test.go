// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// planeFacing builds a boundary surface on the plane through origin whose OUTWARD normal is n.
func planeFacing(t *testing.T, origin math.Point3, n math.Vector3) BoundarySurface {
	t.Helper()
	p, err := NewPlane(origin, n)
	if err != nil {
		t.Fatalf("plane at %v facing %v: %v", origin, n, err)
	}
	return BoundarySurface{Surface: p}
}

// The two faces of one slab must land on the SAME canonical direction, whichever way each is turned,
// or a caller cannot group them — and the grouping is what makes the measurement rotation-invariant.
func TestOppositeFacesOfASlabShareOneDirection(t *testing.T) {
	t.Parallel()
	bottom, ok := AsMaterialSlab(planeFacing(t, math.P3(0, 0, 0), math.V3(0, 0, -1)))
	if !ok {
		t.Fatal("a plane must report a slab")
	}
	top, ok := AsMaterialSlab(planeFacing(t, math.P3(0, 0, 4), math.V3(0, 0, 1)))
	if !ok {
		t.Fatal("a plane must report a slab")
	}
	if bottom.Dir != top.Dir {
		t.Errorf("the two faces of a slab report %v and %v; they must canonicalise to one direction",
			bottom.Dir, top.Dir)
	}
	if !bottom.MaterialAbove || top.MaterialAbove {
		t.Errorf("material lies ABOVE the bottom face and BELOW the top; got %v and %v",
			bottom.MaterialAbove, top.MaterialAbove)
	}
	if bottom.Offset != 0 || top.Offset != 4 {
		t.Errorf("offsets are %v and %v; want 0 and 4", bottom.Offset, top.Offset)
	}
}

// Canonicalisation must be EXACT for two exactly opposite normals, in every octant: a direction and
// its negation normalise by the same length, and the flip is a sign change, so the two agree bit for
// bit rather than to within a tolerance.
func TestCanonicalDirectionIsExactForOppositeNormals(t *testing.T) {
	t.Parallel()
	for _, v := range []math.Vector3{
		math.V3(0, 0, 1), math.V3(1, 1, 1), math.V3(-3, 7, -0.5), math.V3(0, -2, 5),
	} {
		a, _, err := canonicalDirection(v)
		if err != nil {
			t.Fatalf("canonicalDirection(%v): %v", v, err)
		}
		b, _, err := canonicalDirection(v.Scale(-1))
		if err != nil {
			t.Fatalf("canonicalDirection(-%v): %v", v, err)
		}
		if a != b {
			t.Errorf("%v canonicalises to %v and its negation to %v; they must be identical", v, a, b)
		}
	}
}

// A non-planar surface is not a slab, and a plane is not an enclosure: each function answers only
// where it has a closed form, and says so where it has none.
func TestTheSpanFunctionsDeclineWhereTheyHaveNoClosedForm(t *testing.T) {
	t.Parallel()
	cyl, err := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	if _, ok := AsMaterialSlab(BoundarySurface{Surface: cyl}); ok {
		t.Error("a cylinder is not a planar slab")
	}
	if _, ok := EnclosedSpan(planeFacing(t, math.P3(0, 0, 0), math.V3(0, 0, 1))); ok {
		t.Error("a plane encloses nothing")
	}
	cone, err := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.5)
	if err != nil {
		t.Fatalf("cone: %v", err)
	}
	if _, ok := EnclosedSpan(BoundarySurface{Surface: cone}); ok {
		t.Error("a cone's enclosed material runs to a point at the apex and has no surface-only span")
	}
}

// A closed curved surface encloses material only when its normal points OUT of it. Facing in, the same
// cylinder is a BORE, and what it wraps is a void.
func TestEnclosedSpanReadsTheFacingDirection(t *testing.T) {
	t.Parallel()
	cyl, err := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	got, ok := EnclosedSpan(BoundarySurface{Surface: cyl})
	if !ok || got != 4 {
		t.Errorf("EnclosedSpan(rod r=2) = %v, %v; want 4", got, ok)
	}
	if _, ok := EnclosedSpan(BoundarySurface{Surface: cyl, Reversed: true}); ok {
		t.Error("a bore wraps a void, not material")
	}
	sph, err := NewSphere(math.P3(1, 2, 3), 1.5)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	if got, ok := EnclosedSpan(BoundarySurface{Surface: sph}); !ok || got != 3 {
		t.Errorf("EnclosedSpan(ball r=1.5) = %v, %v; want 3", got, ok)
	}
	tor, err := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.25)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	if got, ok := EnclosedSpan(BoundarySurface{Surface: tor}); !ok || got != 2.5 {
		t.Errorf("EnclosedSpan(torus tube r=1.25) = %v, %v; want 2.5", got, ok)
	}
}

// A tube WALL is the span between the outer cylinder and the bore. The pair must be coaxial, and the
// outer one must face out while the inner faces in — any other arrangement has a void between it.
func TestOpposedSpanIsATubeWall(t *testing.T) {
	t.Parallel()
	outer, err := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 10)
	if err != nil {
		t.Fatalf("outer: %v", err)
	}
	inner, err := NewCylinder(math.P3(0, 0, 3), math.V3(0, 0, 1), 9.75)
	if err != nil {
		t.Fatalf("inner: %v", err)
	}
	got, ok := OpposedSpan(BoundarySurface{Surface: outer}, BoundarySurface{Surface: inner, Reversed: true})
	if !ok || stdmath.Abs(got-0.25) > 1e-12 { // tol:numeric — an exact radius difference
		t.Errorf("OpposedSpan(tube 10/9.75) = %v, %v; want 0.25", got, ok)
	}
	if _, ok := OpposedSpan(BoundarySurface{Surface: outer}, BoundarySurface{Surface: inner}); ok {
		t.Error("two out-facing cylinders hold a void between them, not material")
	}
	offAxis, err := NewCylinder(math.P3(3, 0, 0), math.V3(0, 0, 1), 9.75)
	if err != nil {
		t.Fatalf("off-axis: %v", err)
	}
	if _, ok := OpposedSpan(BoundarySurface{Surface: outer}, BoundarySurface{Surface: offAxis, Reversed: true}); ok {
		t.Error("cylinders that do not share an axis line have no constant wall")
	}
}

// Concentric spheres are the same shell in the other family; a pair that is not concentric is not one.
func TestOpposedSpanIsASphericalShell(t *testing.T) {
	t.Parallel()
	outer, err := NewSphere(math.P3(1, 1, 1), 5)
	if err != nil {
		t.Fatalf("outer: %v", err)
	}
	inner, err := NewSphere(math.P3(1, 1, 1), 4.5)
	if err != nil {
		t.Fatalf("inner: %v", err)
	}
	got, ok := OpposedSpan(BoundarySurface{Surface: outer}, BoundarySurface{Surface: inner, Reversed: true})
	if !ok || stdmath.Abs(got-0.5) > 1e-12 { // tol:numeric — an exact radius difference
		t.Errorf("OpposedSpan(shell 5/4.5) = %v, %v; want 0.5", got, ok)
	}
	moved, err := NewSphere(math.P3(2, 1, 1), 4.5)
	if err != nil {
		t.Fatalf("moved: %v", err)
	}
	if _, ok := OpposedSpan(BoundarySurface{Surface: outer}, BoundarySurface{Surface: moved, Reversed: true}); ok {
		t.Error("spheres that are not concentric have no constant shell")
	}
}
