// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
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
	for i := range n {
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
	t.Parallel()
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
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := crossingIntersectVolume(rRod, rFat)
	// 4%: the two fat-wall lens caps inscribe their curvature, so the DefaultQuality meshed volume runs a
	// little under the analytic intersection. The B-rep is exact — at a 10× finer chord tolerance the
	// volume converges to within 0.04% — so this bounds the property-mesh inscribing (~2.5%, and sensitive
	// to where the traced saddle loop's vertices fall about the high-curvature pinch), as for the Steinmetz
	// and partial-penetration siblings below.
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("intersection volume %.4f, want %.4f (analytic) — rel %.4f > 4%%", got, want, rel)
	}
}

// TestBooleanCutDrillCrossingCylinder drills a fat cylinder (R=3, axis z) with a crossing rod (r=1.5, axis
// x): Cut must give the exact four-face analytic solid (two caps, the holed side wall, the tunnel) whose
// volume is the fat cylinder minus the crossing intersection (the tunnel), not triangle-soup CSG.
func TestBooleanCutDrillCrossingCylinder(t *testing.T) {
	t.Parallel()
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
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
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
	t.Parallel()
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
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi*rFat*rFat*hFat + stdmath.Pi*rRod*rRod*hRod - crossingIntersectVolume(rRod, rFat)
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("joined volume %.4f, want %.4f (fat + rod − intersection) — rel %.4f > 4%%", got, want, rel)
	}
}

// TestBooleanCutRodMinusFatStubs subtracts a fat cylinder (R=3, axis z) from a crossing rod (r=1.5, axis x):
// Cut must give the two disconnected rod stubs (a two-shell solid) whose total volume is the rod minus the
// crossing intersection, not triangle-soup CSG.
func TestBooleanCutRodMinusFatStubs(t *testing.T) {
	t.Parallel()
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
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
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
	t.Parallel()
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
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := crossingIntersectVolume(rRod, rFat) / 2 // the plug is half the full crossing intersection
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("plug volume %.4f, want %.4f (half the crossing intersection) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestBooleanCutPartialPenetrationBlindHole cuts a thin rod (r=1.5, axis x) that ENDS at the fat centre out
// of a fat cylinder (R=3, axis z): Cut must give the exact blind pocket (two caps, the holed wall, the rod
// tunnel, the blind bottom) whose volume is the fat minus the plug (half the full crossing intersection).
func TestBooleanCutPartialPenetrationBlindHole(t *testing.T) {
	t.Parallel()
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
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi*rFat*rFat*hFat - crossingIntersectVolume(rRod, rFat)/2 // fat − the plug
	// 4%: the curved tunnel and holed wall inscribe their curvature, so the meshed volume runs a little
	// under the analytic fat − plug (the B-rep is exact; this bounds the property-mesh error, as for the
	// full drill).
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("blind hole volume %.4f, want %.4f (fat − plug) — rel %.4f > 4%%", got, want, rel)
	}
}

// TestBooleanCutPartialPenetrationRodMinusFat subtracts a fat cylinder (R=3, axis z) from a thin rod
// (r=1.5, axis x) that ends at the fat centre: Cut must give the single rod stub outside the fat (a
// one-shell solid) whose volume is the rod minus the plug (half the full crossing intersection).
func TestBooleanCutPartialPenetrationRodMinusFat(t *testing.T) {
	t.Parallel()
	const rRod, hRod, rFat = 1.5, 6.0, 3.0
	fat, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), rFat, 12)
	stub, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), rRod, hRod) // ends at x=0, inside the fat

	res, err := ops.Boolean(ops.Cut, stub, fat) // rod − fat
	if err != nil {
		t.Fatalf("Boolean(Cut rod−fat partial): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("rod − fat is not a valid closed manifold solid: %+v", v)
	}
	if n := len(res.Shells()); n != 1 {
		t.Errorf("rod − fat has %d shells, want 1 (the rod breaches only one wall)", n)
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi*rRod*rRod*hRod - crossingIntersectVolume(rRod, rFat)/2 // rod − the plug
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("rod − fat volume %.4f, want %.4f (rod − plug) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestBooleanJoinPartialPenetration joins a thin rod (r=1.5, axis x) ending at the fat centre with a fat
// cylinder (R=3, axis z): Join must give the fat with one rod stub out the entry side, its volume the fat
// plus the rod minus the plug (the doubly-counted overlap).
func TestBooleanJoinPartialPenetration(t *testing.T) {
	t.Parallel()
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
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
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
	t.Parallel()
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
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := 16.0 / 3.0 * r * r * r // the Steinmetz bicylinder volume
	// 4%: the four lobes inscribe their curvature and pinch to a sharp corner at each pinch vertex, so the
	// meshed volume runs a little under the analytic 16/3·R³ (the B-rep is exact; this bounds the
	// property-mesh error, ~2.5% at DefaultQuality).
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("Steinmetz volume %.4f, want %.4f (16/3·R³) — rel %.4f > 4%%", got, want, rel)
	}
}

// TestBooleanCutEqualRadiusSteinmetz subtracts two EQUAL-radius perpendicular cylinders (R=3): Cut must give
// the exact bitten solid (the target with the tool's saddle bite) whose volume is the cylinder minus the
// Steinmetz bicylinder (π·R²·h − 16/3·R³), not triangle-soup CSG.
func TestBooleanCutEqualRadiusSteinmetz(t *testing.T) {
	t.Parallel()
	const r, h = 3.0, 12.0
	cx, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), r, h)
	cz, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), r, h)

	res, err := ops.Boolean(ops.Cut, cx, cz)
	if err != nil {
		t.Fatalf("Boolean(Cut equal-radius): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("Steinmetz cut is not a valid closed manifold solid: %+v", v)
	}
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder, geom.Plane:
		default:
			t.Errorf("face surface %T is not analytic (the exact path must run, not CSG)", f.Geometry())
		}
	}
	if n := len(res.Faces()); n != 6 {
		t.Errorf("Steinmetz cut has %d faces, want 6 (two bands, two lobes, two caps)", n)
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi*r*r*h - 16.0/3.0*r*r*r // the cylinder minus the bicylinder
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("Steinmetz cut volume %.4f, want %.4f (cyl − bicylinder) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestBooleanJoinEqualRadiusSteinmetz unites two EQUAL-radius perpendicular cylinders (R=3): Join must give
// the exact union (each cylinder's outside, meeting along the intersection ellipses) whose volume is two
// cylinders minus the doubly-counted bicylinder (2·π·R²·h − 16/3·R³), not triangle-soup CSG.
func TestBooleanJoinEqualRadiusSteinmetz(t *testing.T) {
	t.Parallel()
	const r, h = 3.0, 12.0
	cx, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), r, h)
	cz, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), r, h)

	res, err := ops.Boolean(ops.Join, cx, cz)
	if err != nil {
		t.Fatalf("Boolean(Join equal-radius): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("Steinmetz join is not a valid closed manifold solid: %+v", v)
	}
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder, geom.Plane:
		default:
			t.Errorf("face surface %T is not analytic (the exact path must run, not CSG)", f.Geometry())
		}
	}
	if n := len(res.Faces()); n != 8 {
		t.Errorf("Steinmetz join has %d faces, want 8 (two bands + two caps per cylinder)", n)
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := 2*stdmath.Pi*r*r*h - 16.0/3.0*r*r*r // two cylinders minus the bicylinder
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("Steinmetz join volume %.4f, want %.4f (2·cyl − bicylinder) — rel %.4f > 2%%", got, want, rel)
	}
}

// TestBooleanIntersectNearPinchContinuity sweeps the z-cylinder's radius across the near-pinch snap ceiling
// (#1780) and pins two things. (1) BELOW the ceiling the snap produces a clean watertight analytic bicylinder
// — the four-face, manifold, 16/3·R³ solid — where before #1780 the same input fell to the faceted route and
// came out NON-manifold; the snap is a watertightness win, not just smoothness. (2) Volume is CONTINUOUS
// across the ceiling: the snapped exact bicylinder (below) and the deterministic faceted fallback (above)
// both track the analytic crossing-intersection volume to the faceting budget, so crossing the ceiling is no
// volume jump — the true B-rep step is O(Δr·R²), well under 1e-3 here. This is the guard the old silent
// fallback lacked (it would show as a VOLUME step). The residual band above the ceiling is still the faceted,
// non-manifold route — folding it onto the analytic path is #1780 Direction 2 — so watertightness is asserted
// only where the snap owns the result.
func TestBooleanIntersectNearPinchContinuity(t *testing.T) {
	t.Parallel()
	const r = 3.0
	baseX, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), r, 12)
	baseZ, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), r, 12)
	ceil := geom.ResolutionForBox(baseX.RangeBox().Union(baseZ.RangeBox())).Stitch()

	// δ straddling the ceiling: 0, 0.4·ceil, 0.9·ceil snap (exact bicylinder); 2·ceil, 6·ceil fall to the
	// faceted route (still ≪ 2.5e-4·r, so the near-pinch band, not a clean crossing).
	type sample struct {
		d       float64
		snapped bool // within the ceiling: the exact analytic bicylinder the snap owns
	}
	samples := []sample{{0, true}, {0.4 * ceil, true}, {0.9 * ceil, true}, {2 * ceil, false}, {6 * ceil, false}}
	vols := make([]float64, len(samples))
	for i, s := range samples {
		cx, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), r, 12)
		cz, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), r+s.d, 12)
		res, err := ops.Boolean(ops.Intersect, cx, cz)
		if err != nil {
			t.Fatalf("δ=%.3g: Boolean(Intersect): %v", s.d, err)
		}
		if s.snapped {
			if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
				t.Fatalf("δ=%.3g (snap band): must be a valid closed manifold solid, got %+v", s.d, v)
			}
			if n := len(res.Faces()); n != 4 {
				t.Errorf("δ=%.3g (snap band): %d faces, want the four-lobe bicylinder", s.d, n)
			}
		}
		got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
		want := crossingIntersectVolume(r, r+s.d)
		if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
			t.Errorf("δ=%.3g: volume %.4f, want %.4f (analytic) — rel %.4f > 4%%", s.d, got, want, rel)
		}
		vols[i] = got
	}
	// Volume continuity: no sample departs from the equal-radius baseline by more than the faceting budget, so
	// the ceiling is not a volume discontinuity (the ~1.6% snap→faceted change is meshing noise — inscribed
	// four-lobe vs CSG facets — not a geometry step; both B-reps are ~16/3·R³).
	for i, v := range vols {
		if rel := stdmath.Abs(v-vols[0]) / vols[0]; rel > 0.04 {
			t.Errorf("δ=%.3g volume %.4f jumped %.4f from the equal-radius baseline %.4f — a step across the snap ceiling", samples[i].d, v, rel, vols[0])
		}
	}
}

// TestBooleanIntersectEqualRadiusDefersFromExactPath: two EQUAL-radius perpendicular cylinders are the
// Steinmetz case — its bicylinder pinches into four lobes, which the general crossing-intersect path cannot
// emit as a clean watertight solid. The path must therefore NOT be adopted (validBooleanSolid rejects it),
// so the boolean falls to the dedicated analytic Steinmetz handler instead of emitting a wrong solid.
func TestBooleanIntersectEqualRadiusDefersFromExactPath(t *testing.T) {
	t.Parallel()
	a, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	b, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 3, 12) // equal radius

	if res, ok := brep.RuledCrossingIntersectGeneral(a, b, nil); ok && ops.Validate(res).Valid {
		t.Error("equal-radius (Steinmetz) crossing should not be adopted by the general intersect path")
	}
}
