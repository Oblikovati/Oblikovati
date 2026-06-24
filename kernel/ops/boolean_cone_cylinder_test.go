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

// Cone–cylinder intersection through Boolean (M2 Phase 2, Oblikovati/Oblikovati#1335). A cone (a tapered
// rod) crossing a fatter cylinder must Intersect to the exact analytic solid — the cone band plus two
// cylinder-wall lens caps — its volume matching the analytic cone∩cylinder, not triangle-soup CSG.

// coneCylinderIntersectVolume is the volume of the frustum (apex at x=−14, half-angle atan 0.125, so radius
// 1→2.5 over x∈[−6,6]) ∩ the cylinder x²+y²≤R² (axis z). At each x the cone is a disk of radius r(x) clipped
// by the cylinder slab |y|≤√(R²−x²); only |x|≤R lies inside the cylinder. Integrating the clipped-disk area.
func coneCylinderIntersectVolume(rFat float64) float64 {
	const n = 200000
	const apexX, tanHalf = -14.0, 0.125
	sum, lo, hi := 0.0, -rFat, rFat
	for i := 0; i < n; i++ {
		x := lo + (hi-lo)*(float64(i)+0.5)/n
		r := (x - apexX) * tanHalf
		h := stdmath.Sqrt(rFat*rFat - x*x)
		sum += clippedDiskArea(r, h)
	}
	return sum * (hi - lo) / n
}

// clippedDiskArea is the area of {y²+z²≤r²} ∩ {|y|≤h} — a disk of radius r clipped to a slab of half-width h.
func clippedDiskArea(r, h float64) float64 {
	if h >= r {
		return stdmath.Pi * r * r
	}
	if h <= 0 {
		return 0
	}
	seg := r*r*stdmath.Acos(h/r) - h*stdmath.Sqrt(r*r-h*h) // one circular segment beyond |y|=h
	return stdmath.Pi*r*r - 2*seg
}

// TestBooleanIntersectConeCylinder crosses a frustum (radius 1→2.5, axis x) through a radius-3 cylinder
// (axis z) and checks the result is the exact three-face analytic solid (cone band + two lens caps) with the
// analytic cone∩cylinder volume.
func TestBooleanIntersectConeCylinder(t *testing.T) {
	const rFat = 3.0
	cone, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), rFat, 12)

	res, err := ops.Boolean(ops.Intersect, cone, cyl)
	if err != nil {
		t.Fatalf("Boolean(Intersect cone∩cyl): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("cone∩cylinder is not a valid closed manifold solid: %+v", v)
	}
	cones, cyls := 0, 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cone:
			cones++
		case geom.Cylinder:
			cyls++
		default:
			t.Errorf("face surface %T is not analytic (the exact path must run, not CSG)", f.Geometry())
		}
	}
	if cones != 1 || cyls != 2 {
		t.Errorf("got %d cone + %d cylinder faces, want 1 (band) + 2 (lens caps)", cones, cyls)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := coneCylinderIntersectVolume(rFat)
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("cone∩cylinder volume %.4f, want %.4f (analytic) — rel %.4f > 3%%", got, want, rel)
	}
}
