// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// The interior coordinate extrema of a surface (#3421): only a sphere and a torus have any, and a
// face's box needs them because its boundary curves alone miss a bulge between them.

// TestRuledSurfacesHaveNoInteriorExtrema: a plane's normal is constant and a cylinder's or cone's
// depends on u alone, so their stationary sets are rulings whose extremes sit on the face boundary —
// nothing for a box to add.
func TestRuledSurfacesHaveNoInteriorExtrema(t *testing.T) {
	pl, err := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	cyl, err := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	cone, err := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.5)
	if err != nil {
		t.Fatalf("NewCone: %v", err)
	}
	for _, s := range []Surface{pl, cyl, cone} {
		pts, ok := SurfaceAxisCriticalPoints(s)
		if !ok || len(pts) != 0 {
			t.Errorf("%T reports %d interior extrema (ok=%v), want none", s, len(pts), ok)
		}
	}
}

// TestSphereCriticalPointsAreTheAxisPoles: the six points where the radial normal lines up with a
// world axis, which is exactly where a spherical face bulges past its boundary circle.
func TestSphereCriticalPointsAreTheAxisPoles(t *testing.T) {
	sph, err := NewSphere(math.P3(1, 2, 3), 5)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	pts, ok := SurfaceAxisCriticalPoints(sph)
	if !ok || len(pts) != 6 {
		t.Fatalf("sphere reports %d points (ok=%v), want 6", len(pts), ok)
	}
	box := math.BoxFromPoints(pts...)
	if stdmath.Abs(float64(box.Min.X)-(-4)) > 1e-12 || stdmath.Abs(float64(box.Max.Z)-8) > 1e-12 {
		t.Errorf("pole box = %v, want the centre ±5 on every axis", box)
	}
}

// TestTorusCriticalPointsSitOnTheSurface: every returned point must be ON the torus and have a
// normal parallel to some world axis — the defining property, checked rather than assumed.
func TestTorusCriticalPointsSitOnTheSurface(t *testing.T) {
	tor, err := NewTorus(math.P3(0, 0, 0), math.V3(1, 2, 3), 6, 2)
	if err != nil {
		t.Fatalf("NewTorus: %v", err)
	}
	pts, ok := SurfaceAxisCriticalPoints(tor)
	if !ok || len(pts) != 12 {
		t.Fatalf("torus reports %d points (ok=%v), want 12", len(pts), ok)
	}
	for i, p := range pts {
		u, v := tor.ParamAt(p)
		if d := float64(tor.PointAt(u, v).DistanceTo(p)); d > 1e-9 {
			t.Errorf("point %d is %g off the torus", i, d)
		}
		if !normalAlignsWithAnAxis(tor.NormalAt(u, v)) {
			t.Errorf("point %d has normal %v, parallel to no world axis", i, tor.NormalAt(u, v))
		}
	}
}

// TestAxisAlignedTorusDeclines: with the tube axis along a world axis the stationary set for that
// axis is a whole latitude circle, not isolated points, so enumerating it would mean sampling.
func TestAxisAlignedTorusDeclines(t *testing.T) {
	tor, err := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 6, 2)
	if err != nil {
		t.Fatalf("NewTorus: %v", err)
	}
	if _, ok := SurfaceAxisCriticalPoints(tor); ok {
		t.Error("a torus about +z must decline rather than report isolated extrema")
	}
}

// normalAlignsWithAnAxis reports whether n is parallel to x, y or z.
func normalAlignsWithAnAxis(n math.Vector3) bool {
	for axis := range 3 {
		if stdmath.Abs(stdmath.Abs(float64(n.Dot(worldAxis(axis))))-float64(n.Length())) < 1e-9 {
			return true
		}
	}
	return false
}
