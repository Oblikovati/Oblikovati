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

// TestBooleanCutDrillCrossingCylinder drills a fat cylinder (R=3, axis z) with a crossing rod (r=1.5, axis
// x): Cut must give the exact four-face analytic solid (two caps, the holed side wall, the tunnel) whose
// volume is the fat cylinder minus the crossing intersection (the tunnel), not triangle-soup CSG.
func TestBooleanCutDrillCrossingCylinder(t *testing.T) {
	const rRod, rFat, hFat = 1.5, 3.0, 12.0
	fat, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), rFat, hFat)
	thin, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), rRod, 12)

	res, err := ops.Boolean(ops.Cut, fat, thin)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("drilled cylinder is not a valid closed manifold solid: %+v", v)
	}
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder, geom.Plane:
		default:
			t.Errorf("face surface %T is not analytic (the exact path must run, not CSG)", f.Geometry())
		}
	}
	if n := len(res.Faces()); n != 4 {
		t.Errorf("drilled cylinder has %d faces, want 4 (two caps, holed wall, tunnel)", n)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi*rFat*rFat*hFat - crossingIntersectVolume(rRod, rFat)
	// 4%: the concave tunnel and the drilled wall inscribe their curvature, so the meshed volume runs a
	// little under the analytic fat − tunnel (the B-rep is exact; this bounds the property-mesh error).
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("drilled volume %.4f, want %.4f (fat − tunnel) — rel %.4f > 4%%", got, want, rel)
	}
}

// TestBooleanJoinCrossingCylinders joins a fat cylinder (R=3, axis z) with a crossing rod (r=1.5, axis x):
// Join must give the connected analytic solid (fat caps, holed fat wall, a rod stub each side capped by the
// rod's end disc) whose volume is fat + rod − the crossing intersection, not triangle-soup CSG.
func TestBooleanJoinCrossingCylinders(t *testing.T) {
	const rRod, hRod, rFat, hFat = 1.5, 12.0, 3.0, 12.0
	fat, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), rFat, hFat)
	thin, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), rRod, hRod)

	res, err := ops.Boolean(ops.Join, fat, thin)
	if err != nil {
		t.Fatalf("Boolean(Join): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("joined cylinders are not a valid closed manifold solid: %+v", v)
	}
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder, geom.Plane:
		default:
			t.Errorf("face surface %T is not analytic (the exact path must run, not CSG)", f.Geometry())
		}
	}
	if n := len(res.Faces()); n != 7 {
		t.Errorf("joined cylinders have %d faces, want 7 (two fat caps, holed wall, two stubs, two rod caps)", n)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi*rFat*rFat*hFat + stdmath.Pi*rRod*rRod*hRod - crossingIntersectVolume(rRod, rFat)
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("joined volume %.4f, want %.4f (fat + rod − intersection) — rel %.4f > 4%%", got, want, rel)
	}
}

// TestBooleanCutRodMinusFatStubs subtracts a fat cylinder (R=3, axis z) from a crossing rod (r=1.5, axis x):
// Cut must give the two disconnected rod stubs (a two-shell solid) whose total volume is the rod minus the
// crossing intersection, not triangle-soup CSG.
func TestBooleanCutRodMinusFatStubs(t *testing.T) {
	const rRod, hRod, rFat = 1.5, 12.0, 3.0
	fat, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), rFat, 12)
	thin, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), rRod, hRod)

	res, err := ops.Boolean(ops.Cut, thin, fat) // rod − fat
	if err != nil {
		t.Fatalf("Boolean(Cut rod−fat): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("rod − fat is not a valid closed manifold solid: %+v", v)
	}
	if n := len(res.Shells()); n != 2 {
		t.Errorf("rod − fat has %d shells, want 2 (a disconnected stub each side)", n)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi*rRod*rRod*hRod - crossingIntersectVolume(rRod, rFat)
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("rod − fat volume %.4f, want %.4f (rod − intersection) — rel %.4f > 4%%", got, want, rel)
	}
}

// TestBooleanIntersectPartialPenetration intersects a fat cylinder (R=3, axis z) with a thin rod (r=1.5,
// axis x) that ENDS at the fat centre: Intersect must give the exact three-face plug (fat-wall lens, rod
// stub band, blind end cap). Because the rod stops at the centre, the plug is exactly half the full-crossing
// intersection, so its volume is crossingIntersectVolume(r,R)/2, not triangle-soup CSG.
func TestBooleanIntersectPartialPenetration(t *testing.T) {
	const rRod, rFat = 1.5, 3.0
	fat, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), rFat, 12)
	stub, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), rRod, 6) // ends at x=0, inside the fat

	res, err := ops.Boolean(ops.Intersect, fat, stub)
	if err != nil {
		t.Fatalf("Boolean(Intersect partial): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("rod plug is not a valid closed manifold solid: %+v", v)
	}
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder, geom.Plane:
		default:
			t.Errorf("face surface %T is not analytic (the exact path must run, not CSG)", f.Geometry())
		}
	}
	if n := len(res.Faces()); n != 3 {
		t.Errorf("rod plug has %d faces, want 3 (lens, stub band, blind end cap)", n)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := crossingIntersectVolume(rRod, rFat) / 2 // the plug is half the full crossing intersection
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("plug volume %.4f, want %.4f (half the crossing intersection) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestBooleanCutPartialPenetrationBlindHole cuts a thin rod (r=1.5, axis x) that ENDS at the fat centre out
// of a fat cylinder (R=3, axis z): Cut must give the exact blind pocket (two caps, the holed wall, the rod
// tunnel, the blind bottom) whose volume is the fat minus the plug (half the full crossing intersection).
func TestBooleanCutPartialPenetrationBlindHole(t *testing.T) {
	const rRod, rFat, hFat = 1.5, 3.0, 12.0
	fat, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), rFat, hFat)
	stub, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), rRod, 6) // ends at x=0, inside the fat

	res, err := ops.Boolean(ops.Cut, fat, stub)
	if err != nil {
		t.Fatalf("Boolean(Cut blind hole): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("blind hole is not a valid closed manifold solid: %+v", v)
	}
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder, geom.Plane:
		default:
			t.Errorf("face surface %T is not analytic (the exact path must run, not CSG)", f.Geometry())
		}
	}
	if n := len(res.Faces()); n != 5 {
		t.Errorf("blind hole has %d faces, want 5 (two caps, holed wall, tunnel, blind bottom)", n)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi*rFat*rFat*hFat - crossingIntersectVolume(rRod, rFat)/2 // fat − the plug
	// 4%: the curved tunnel and holed wall inscribe their curvature, so the meshed volume runs a little
	// under the analytic fat − plug (the B-rep is exact; this bounds the property-mesh error, as for the
	// full drill).
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("blind hole volume %.4f, want %.4f (fat − plug) — rel %.4f > 4%%", got, want, rel)
	}
}

// TestBooleanJoinPartialPenetration joins a thin rod (r=1.5, axis x) ending at the fat centre with a fat
// cylinder (R=3, axis z): Join must give the fat with one rod stub out the entry side, its volume the fat
// plus the rod minus the plug (the doubly-counted overlap).
func TestBooleanJoinPartialPenetration(t *testing.T) {
	const rRod, hRod, rFat, hFat = 1.5, 6.0, 3.0, 12.0
	fat, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), rFat, hFat)
	stub, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), rRod, hRod) // ends at x=0, inside the fat

	res, err := ops.Boolean(ops.Join, fat, stub)
	if err != nil {
		t.Fatalf("Boolean(Join partial): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("partial join is not a valid closed manifold solid: %+v", v)
	}
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder, geom.Plane:
		default:
			t.Errorf("face surface %T is not analytic (the exact path must run, not CSG)", f.Geometry())
		}
	}
	if n := len(res.Faces()); n != 5 {
		t.Errorf("partial join has %d faces, want 5 (two caps, holed wall, stub band, entry cap)", n)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi*rFat*rFat*hFat + stdmath.Pi*rRod*rRod*hRod - crossingIntersectVolume(rRod, rFat)/2
	// 4%: the curved holed wall and stub inscribe their curvature, so the meshed volume runs a little under
	// the analytic fat + rod − plug (the B-rep is exact; this bounds the property-mesh error).
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("partial join volume %.4f, want %.4f (fat + rod − plug) — rel %.4f > 4%%", got, want, rel)
	}
}

// TestBooleanIntersectEqualRadiusSteinmetz intersects two EQUAL-radius perpendicular cylinders (axes x and
// z, R=3): Intersect must give the exact Steinmetz bicylinder — four analytic cylinder faces — whose volume
// is the closed form 16/3·R³, not triangle-soup CSG (the SSI tracer pinches on this case, so it is fitted
// analytically as two crossing ellipses).
func TestBooleanIntersectEqualRadiusSteinmetz(t *testing.T) {
	const r = 3.0
	cx, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), r, 12)
	cz, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), r, 12)

	res, err := ops.Boolean(ops.Intersect, cx, cz)
	if err != nil {
		t.Fatalf("Boolean(Intersect equal-radius): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("Steinmetz bicylinder is not a valid closed manifold solid: %+v", v)
	}
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); !ok {
			t.Errorf("face surface %T is not analytic (the exact path must run, not CSG)", f.Geometry())
		}
	}
	if n := len(res.Faces()); n != 4 {
		t.Errorf("Steinmetz has %d faces, want 4 (two lobes per cylinder)", n)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := 16.0 / 3.0 * r * r * r // the Steinmetz bicylinder volume
	// 4%: the four lobes inscribe their curvature and pinch to a sharp corner at each pinch vertex, so the
	// meshed volume runs a little under the analytic 16/3·R³ (the B-rep is exact; this bounds the
	// property-mesh error, ~2.5% at DefaultQuality).
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("Steinmetz volume %.4f, want %.4f (16/3·R³) — rel %.4f > 4%%", got, want, rel)
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
