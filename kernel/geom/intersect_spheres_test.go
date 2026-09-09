// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestSphereSphereSectionIsTheRadicalCircle: two overlapping spheres cross in one circle, in the plane
// perpendicular to their line of centres. Every point of it must lie on BOTH spheres.
func TestSphereSphereSectionIsTheRadicalCircle(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(8)
	a, _ := NewSphere(math.P3(0, 0, 0), 2)
	b, _ := NewSphere(math.P3(2, 0, 0), 2)
	curves, handled := IntersectSurfacesAnalytic(a, b, res)
	if !handled {
		t.Fatal("two overlapping spheres were not claimed by the analytic intersector")
	}
	if len(curves) != 1 {
		t.Fatalf("two spheres crossed in %d curves, want 1 (the radical circle)", len(curves))
	}
	c, isCircle := curves[0].(Circle)
	if !isCircle {
		t.Fatalf("the section is a %T, want a geom.Circle", curves[0])
	}
	// Centres 2 apart, radii 2: the circle sits midway, of radius √3.
	if stdmath.Abs(float64(c.Center.X)-1) > 1e-12 || stdmath.Abs(c.Radius-stdmath.Sqrt(3)) > 1e-12 {
		t.Errorf("radical circle centre %v radius %v; want x=1, r=%v", c.Center, c.Radius, stdmath.Sqrt(3))
	}
	for i := 0; i <= 16; i++ {
		p := c.PointAt(float64(i) / 16)
		if da := stdmath.Abs(float64(a.Center.DistanceTo(p)) - a.Radius); da > 1e-12 {
			t.Fatalf("a section point is %v off sphere a", da)
		}
		if db := stdmath.Abs(float64(b.Center.DistanceTo(p)) - b.Radius); db > 1e-12 {
			t.Fatalf("a section point is %v off sphere b", db)
		}
	}
}

// TestSphereSphereSectionDecidesTheNonCrossings: spheres apart, or one strictly inside the other, are
// DECIDED with no curves; a tangent touch and a concentric pair are declined, because a point and a
// coincident surface are not crossings a section curve can carry.
func TestSphereSphereSectionDecidesTheNonCrossings(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(20)
	at := func(x, r float64) Sphere {
		s, _ := NewSphere(math.P3(math.Scalar(x), 0, 0), r)
		return s
	}
	for _, tc := range []struct {
		name            string
		a, b            Sphere
		handled, curves bool
	}{
		{"apart", at(0, 2), at(10, 2), true, false},
		{"one inside the other", at(0, 5), at(0.5, 2), true, false},
		{"tangent outside", at(0, 2), at(4, 2), false, false},
		{"concentric, equal radii", at(0, 2), at(0, 2), false, false},
		{"crossing", at(0, 2), at(2, 2), true, true},
	} {
		curves, handled := IntersectSurfacesAnalytic(tc.a, tc.b, res)
		if handled != tc.handled {
			t.Errorf("%s: handled = %v, want %v", tc.name, handled, tc.handled)
		}
		if (len(curves) > 0) != tc.curves {
			t.Errorf("%s: %d curves, want any = %v", tc.name, len(curves), tc.curves)
		}
	}
}
