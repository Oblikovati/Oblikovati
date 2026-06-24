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

// Crossing-cylinder intersection through Boolean (M2 Phase 2, Oblikovati/Oblikovati#1335). A rod ∩ a
// fatter cylinder must Intersect to the exact analytic solid (rod band + two fat-wall lens caps), its
// volume matching the analytic intersection of two perpendicular cylinders, not triangle-soup CSG.

// crossingIntersectVolume is the exact volume of {y²+z² ≤ rRod²} ∩ {x²+y² ≤ rFat²} — a rod of radius rRod
// (axis x) crossing a cylinder of radius rFat (axis z) through the centre. Integrating x out first leaves
// ∫₀^{2π}∫₀^{rRod} 2√(rFat²−ρ²cos²φ)·ρ dρ dφ; the inner ρ-integral has the closed form below (its φ→π/2
// limit, where cos²φ→0, is rFat·rRod²).
func crossingIntersectVolume(rRod, rFat float64) float64 {
	const n = 20000
	sum := 0.0
	for i := 0; i < n; i++ {
		phi := 2 * stdmath.Pi * (float64(i) + 0.5) / n
		c := stdmath.Cos(phi) * stdmath.Cos(phi)
		if c < 1e-9 {
			sum += rFat * rRod * rRod
			continue
		}
		sum += (2.0 / (3 * c)) * (rFat*rFat*rFat - stdmath.Pow(rFat*rFat-rRod*rRod*c, 1.5))
	}
	return sum * 2 * stdmath.Pi / n
}

// TestBooleanIntersectCrossingCylinders intersects a rod (r=1.5, axis x) with a fat cylinder (R=3, axis z)
// and checks the result is the exact three-face analytic solid with the analytic intersection volume.
func TestBooleanIntersectCrossingCylinders(t *testing.T) {
	const rRod, rFat = 1.5, 3.0
	fat, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), rFat, 12)
	thin, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), rRod, 12)

	res, err := ops.Boolean(ops.Intersect, fat, thin)
	if err != nil {
		t.Fatalf("Boolean(Intersect): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("crossing-cylinder intersection is not a valid closed manifold solid: %+v", v)
	}
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); !ok {
			t.Errorf("face surface %T is not analytic (the exact path must run, not CSG)", f.Geometry())
		}
	}
	if n := len(res.Faces()); n != 3 {
		t.Errorf("result has %d faces, want 3 (rod band + two lens caps)", n)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := crossingIntersectVolume(rRod, rFat)
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("intersection volume %.4f, want %.4f (analytic) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestBooleanIntersectEqualRadiusDefersFromExactPath: two EQUAL-radius perpendicular cylinders are the
// Steinmetz case the imprint tracer cannot trace cleanly, so the exact path must decline (leaving the
// boolean to its fallback) rather than emit a wrong analytic solid.
func TestBooleanIntersectEqualRadiusDefersFromExactPath(t *testing.T) {
	a, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	b, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12) // equal radius

	if _, ok := brep.CrossingCylinderIntersect(a, b); ok {
		t.Error("equal-radius (Steinmetz) crossing should defer from the exact path (ok=false)")
	}
}
