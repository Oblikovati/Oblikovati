// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
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

// coneFrustumVolume is the volume of a frustum of axial length h between end radii r0 and r1:
// π·h/3·(r0² + r0·r1 + r1²).
func coneFrustumVolume(r0, r1, h float64) float64 {
	return stdmath.Pi * h / 3 * (r0*r0 + r0*r1 + r1*r1)
}

// TestBooleanCutConeCylinderDrillsFat drills a radius-3 cylinder (axis z) with a crossing frustum (radius
// 1→2.5, axis x): Cut must give the exact analytic solid (two fat caps, the holed fat wall, the cone tunnel)
// whose volume is the fat minus the cone∩cylinder, not triangle-soup CSG.
func TestBooleanCutConeCylinderDrillsFat(t *testing.T) {
	const rFat, hFat = 3.0, 12.0
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), rFat, hFat)
	cone, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")

	res, err := ops.Boolean(ops.Cut, cyl, cone)
	if err != nil {
		t.Fatalf("Boolean(Cut fat−cone): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("drilled cylinder is not a valid closed manifold solid: %+v", v)
	}
	assertConeCylinderAnalytic(t, res)
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi*rFat*rFat*hFat - coneCylinderIntersectVolume(rFat)
	// 4%: the tapered tunnel and the drilled wall inscribe their curvature, so the meshed volume runs a
	// little under the analytic fat − tunnel (the B-rep is exact; this bounds the property-mesh error).
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("drilled volume %.4f, want %.4f (fat − cone∩cyl) — rel %.4f > 4%%", got, want, rel)
	}
}

// TestBooleanCutConeMinusCylinderStubs subtracts a fat cylinder from a crossing frustum: Cut must give the
// two disconnected tapered stubs (a two-shell solid) whose total volume is the cone minus the cone∩cylinder.
func TestBooleanCutConeMinusCylinderStubs(t *testing.T) {
	const rFat = 3.0
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), rFat, 12)
	cone, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")

	res, err := ops.Boolean(ops.Cut, cone, cyl) // cone − fat
	if err != nil {
		t.Fatalf("Boolean(Cut cone−fat): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("cone − fat is not a valid closed manifold solid: %+v", v)
	}
	if n := len(res.Shells()); n != 2 {
		t.Errorf("cone − fat has %d shells, want 2 (a disconnected stub each side)", n)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := coneFrustumVolume(1, 2.5, 12) - coneCylinderIntersectVolume(rFat)
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("cone − fat volume %.4f, want %.4f (cone − cone∩cyl) — rel %.4f > 4%%", got, want, rel)
	}
}

// TestBooleanJoinConeCylinder joins a radius-3 cylinder (axis z) with a crossing frustum (radius 1→2.5, axis
// x): Join must give the connected analytic solid (fat caps, holed fat wall, a tapered stub each side) whose
// volume is fat + cone − the cone∩cylinder, not triangle-soup CSG.
func TestBooleanJoinConeCylinder(t *testing.T) {
	const rFat, hFat = 3.0, 12.0
	cyl, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), rFat, hFat)
	cone, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")

	res, err := ops.Boolean(ops.Join, cyl, cone)
	if err != nil {
		t.Fatalf("Boolean(Join cyl∪cone): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("joined cone∪cylinder is not a valid closed manifold solid: %+v", v)
	}
	assertConeCylinderAnalytic(t, res)
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi*rFat*rFat*hFat + coneFrustumVolume(1, 2.5, 12) - coneCylinderIntersectVolume(rFat)
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("joined volume %.4f, want %.4f (fat + cone − cone∩cyl) — rel %.4f > 4%%", got, want, rel)
	}
}

// assertConeCylinderAnalytic fails the test on any face whose surface is not a cone, cylinder, or plane (the
// exact analytic path must run, not the triangle-soup CSG fallback).
func assertConeCylinderAnalytic(t *testing.T, res *topo.Body) {
	t.Helper()
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cone, geom.Cylinder, geom.Plane:
		default:
			t.Errorf("face surface %T is not analytic (the exact path must run, not CSG)", f.Geometry())
		}
	}
}

// conePartialPlugVolume is the volume of the partial-penetration frustum (apex at x=−14, half-angle atan
// 0.125; the frustum runs x∈[−6,0], r 1→1.75) ∩ the cylinder x²+y²≤rFat² (axis z). The frustum ends at the
// fat axis (x=0, inside the fat), so only x∈[−rFat,0] lies within the cylinder: integrate the clipped-disk
// area there. Mirrors coneCylinderIntersectVolume but with the upper limit at the cone's blind end (x=0).
func conePartialPlugVolume(rFat float64) float64 {
	const n = 200000
	const apexX, tanHalf = -14.0, 0.125
	sum, lo, hi := 0.0, -rFat, 0.0
	for i := 0; i < n; i++ {
		x := lo + (hi-lo)*(float64(i)+0.5)/n
		r := (x - apexX) * tanHalf
		h := stdmath.Sqrt(rFat*rFat - x*x)
		sum += clippedDiskArea(r, h)
	}
	return sum * (hi - lo) / n
}

// conePartialFrustum builds the partial-penetration frustum (x=−6 r=1 → x=0 r=1.75); conePartialFat the
// radius-3 cylinder it ends inside.
func conePartialFrustum() *topo.Body {
	cone, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(0, 0, 0), 1, 1.75, "cone")
	return cone
}

func conePartialFat() *topo.Body {
	fat, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	return fat
}

// TestBooleanIntersectConePartialPlug intersects the radius-3 cylinder with a frustum ending at its axis: the
// result is the exact three-face plug (cone band + lens cap + blind cap) with the analytic cone∩cylinder
// volume.
func TestBooleanIntersectConePartialPlug(t *testing.T) {
	const rFat = 3.0
	res, err := ops.Boolean(ops.Intersect, conePartialFat(), conePartialFrustum())
	if err != nil {
		t.Fatalf("Boolean(Intersect cone-plug): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("cone plug is not a valid closed manifold solid: %+v", v)
	}
	assertConeCylinderAnalytic(t, res)
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := conePartialPlugVolume(rFat)
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("cone plug volume %.4f, want %.4f (analytic) — rel %.4f > 3%%", got, want, rel)
	}
}

// TestBooleanCutConePartialBlindHole subtracts the frustum from the fat (fat − cone): a blind tapered pocket
// whose volume is the fat minus the plug.
func TestBooleanCutConePartialBlindHole(t *testing.T) {
	const rFat, hFat = 3.0, 12.0
	res, err := ops.Boolean(ops.Cut, conePartialFat(), conePartialFrustum())
	if err != nil {
		t.Fatalf("Boolean(Cut fat−cone partial): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("cone blind hole is not a valid closed manifold solid: %+v", v)
	}
	assertConeCylinderAnalytic(t, res)
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi*rFat*rFat*hFat - conePartialPlugVolume(rFat)
	// 4%: the faceted fat wall and the inscribed tapered pocket run the meshed volume a little under the
	// analytic fat − plug (the B-rep is exact; this bounds the property-mesh error).
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("cone blind hole volume %.4f, want %.4f (fat − plug) — rel %.4f > 4%%", got, want, rel)
	}
}

// TestBooleanCutConePartialStub subtracts the fat from the frustum (cone − fat): the single tapered stub
// sticking out the entry side (one shell) whose volume is the frustum minus the plug.
func TestBooleanCutConePartialStub(t *testing.T) {
	const rFat = 3.0
	res, err := ops.Boolean(ops.Cut, conePartialFrustum(), conePartialFat())
	if err != nil {
		t.Fatalf("Boolean(Cut cone−fat partial): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("cone − fat (partial) is not a valid closed manifold solid: %+v", v)
	}
	if n := len(res.Shells()); n != 1 {
		t.Errorf("cone − fat (partial) has %d shells, want 1 (a single one-sided stub)", n)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := coneFrustumVolume(1, 1.75, 6) - conePartialPlugVolume(rFat)
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("cone − fat (partial) volume %.4f, want %.4f (frustum − plug) — rel %.4f > 3%%", got, want, rel)
	}
}

// TestBooleanJoinConePartial joins the fat and the partially-penetrating frustum (fat ∪ cone): one connected
// solid whose volume is fat + frustum − the plug.
func TestBooleanJoinConePartial(t *testing.T) {
	const rFat, hFat = 3.0, 12.0
	res, err := ops.Boolean(ops.Join, conePartialFat(), conePartialFrustum())
	if err != nil {
		t.Fatalf("Boolean(Join fat∪cone partial): %v", err)
	}
	if v := ops.Validate(res); !v.Valid || !v.Closed || !v.Manifold || !res.IsSolid() {
		t.Fatalf("fat ∪ cone (partial) is not a valid closed manifold solid: %+v", v)
	}
	assertConeCylinderAnalytic(t, res)
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	want := stdmath.Pi*rFat*rFat*hFat + coneFrustumVolume(1, 1.75, 6) - conePartialPlugVolume(rFat)
	if rel := stdmath.Abs(got-want) / want; rel > 0.04 {
		t.Errorf("fat ∪ cone (partial) volume %.4f, want %.4f (fat + frustum − plug) — rel %.4f > 4%%", got, want, rel)
	}
}
