// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/math"
)

// The kernel's mass-properties and box oracles read the analytic B-rep (M48/C3 #3421/#3453/#3454/
// #3482). These tests assert what a tessellated integral CANNOT deliver: exact closed-form values.

// analyticQuadRelTol is what "exact" means for these assertions: the adaptive quadrature converges
// to its own relative tolerance (quadRelTol), so the analytic values agree with the closed forms to
// ~1e-11 relative. The tessellated sums they replaced were wrong in the THIRD digit.
const analyticQuadRelTol = 1e-9

// analyticRelDiff is |got-want|/|want|, the scale-free comparison these closed-form checks need.
func analyticRelDiff(got, want float64) float64 {
	if want == 0 {
		return stdmath.Abs(got)
	}
	return stdmath.Abs(got-want) / stdmath.Abs(want)
}

// TestBodyGeometryPropertiesIsExactForACylinder: πr²h and 2πr(r+h) to the last digits, where the
// tessellated sum was low by the chord deficit of its inscribed polygon.
func TestBodyGeometryPropertiesIsExactForACylinder(t *testing.T) {
	const r, h = 3.0, 7.0
	body, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	gp := BodyGeometryProperties(body, DefaultQuality())
	if want := stdmath.Pi * r * r * h; analyticRelDiff(gp.Volume, want) > analyticQuadRelTol {
		t.Errorf("volume = %.12f, want πr²h = %.12f", gp.Volume, want)
	}
	if want := 2 * stdmath.Pi * r * (r + h); analyticRelDiff(gp.Area, want) > analyticQuadRelTol {
		t.Errorf("area = %.12f, want 2πr(r+h) = %.12f", gp.Area, want)
	}
	if d := float64(gp.Centroid.DistanceTo(math.P3(0, 0, h/2))); d > 1e-9 {
		t.Errorf("centroid is %g off the axis midpoint", d)
	}
}

// TestBodyInertiaIsExactForACylinder: Izz = ½Vr² about the axis, from the analytic integral.
func TestBodyInertiaIsExactForACylinder(t *testing.T) {
	const r, h = 3.0, 7.0
	body, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	got := BodyInertia(body, DefaultQuality()).Izz
	want := 0.5 * (stdmath.Pi * r * r * h) * r * r
	if analyticRelDiff(got, want) > analyticQuadRelTol {
		t.Errorf("Izz = %.12f, want ½Vr² = %.12f", got, want)
	}
}

// TestAnalyticShellVolumeSignsTheCavity: a void shell's material-outward normals point INTO the
// cavity, so its signed volume is negative — and now exactly −8, not −8±0.05 as the merged face
// mesh reported.
func TestAnalyticShellVolumeSignsTheCavity(t *testing.T) {
	body := cavityBody(t)
	var outer, void float64
	for _, sh := range body.Shells() {
		v, ok := AnalyticShellVolume(sh)
		if !ok {
			t.Fatalf("shell %d declined analytic integration", sh.Index())
		}
		if v < 0 {
			void = v
		} else {
			outer = v
		}
	}
	if analyticRelDiff(outer, 64) > analyticQuadRelTol {
		t.Errorf("outer shell volume = %.12f, want 64", outer)
	}
	if analyticRelDiff(void, -8) > analyticQuadRelTol {
		t.Errorf("void shell volume = %.12f, want -8", void)
	}
}

// TestPreciseRangeBoxSpansTheFullSphere: the equator bulges past every boundary curve, so a box
// read off facet chords is short by the sagitta on all six faces. The analytic box is not.
func TestPreciseRangeBoxSpansTheFullSphere(t *testing.T) {
	const r = 5.0
	body, err := brep.SolidSphere(math.P3(1, 2, 3), r, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	box := PreciseRangeBox(body, DefaultQuality())
	if stdmath.Abs(float64(box.Min.X)-(1-r)) > 1e-9 || stdmath.Abs(float64(box.Max.Z)-(3+r)) > 1e-9 {
		t.Errorf("sphere box = %v, want the centre ±%g on every axis", box, r)
	}
}

// TestPreciseRangeBoxSpansACylinderRadius: the side face's rim circles carry the bulge, and their
// closed-form extrema put the box on the true radius rather than on an inscribed polygon's apothem.
func TestPreciseRangeBoxSpansACylinderRadius(t *testing.T) {
	const r, h = 3.0, 7.0
	body, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	box := PreciseRangeBox(body, DefaultQuality())
	if stdmath.Abs(float64(box.Max.X)-r) > 1e-9 || stdmath.Abs(float64(box.Min.Y)+r) > 1e-9 {
		t.Errorf("cylinder box = %v, want ±%g in x and y", box, r)
	}
	if stdmath.Abs(float64(box.Min.Z)) > 1e-9 || stdmath.Abs(float64(box.Max.Z)-h) > 1e-9 {
		t.Errorf("cylinder box z = [%g, %g], want [0, %g]", box.Min.Z, box.Max.Z, h)
	}
}

// TestFaceInteriorPointLandsInsideEveryTrim: the probe a per-face gate classifies must really be on
// the face it came from, on every face of a body with curved trims.
func TestFaceInteriorPointLandsInsideEveryTrim(t *testing.T) {
	ball, err := brep.SolidSphere(math.P3(0, 0, 0), 5, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	rod, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 1, 0), 3, 4.5)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	res, err := Boolean(Cut, rod, ball)
	if err != nil {
		t.Fatalf("Boolean: %v", err)
	}
	for i, f := range res.Faces() {
		p, ok := FaceInteriorPoint(f)
		if !ok {
			t.Errorf("face %d (%T) yielded no interior point", i, f.Geometry())
			continue
		}
		if !brep.PointInFaceTrim(f, p) {
			t.Errorf("face %d interior point %v is outside its own trim", i, p)
		}
	}
}

// TestDrilledPlateIntegratesAnalytically is the periodic-band regression (#3453): a plate with a
// hole in it is the most ordinary body in CAD, and its bore wall is a band whose loop crosses the
// parameter seam. Green's u-form cannot close such a loop, so the whole body used to fall back to
// the tessellation; the conjugate v-form integrates it exactly.
func TestDrilledPlateIntegratesAnalytically(t *testing.T) {
	plate, err := brep.SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	drill, err := brep.SolidCylinder(math.P3(0, 0, -1), math.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	res, err := Boolean(Cut, plate, drill)
	if err != nil {
		t.Fatalf("Boolean: %v", err)
	}
	gp, ok := AnalyticGeometryProperties(res)
	if !ok {
		t.Fatal("the drilled plate declined analytic integration; its bore wall is a periodic band")
	}
	want := 10*10*2 - stdmath.Pi*2*2*2
	if analyticRelDiff(gp.Volume, want) > analyticQuadRelTol {
		t.Errorf("drilled plate volume = %.9f, want %.9f", gp.Volume, want)
	}
}

// TestVectorAreaClosureRejectsAResidual: the outward vector area of a closed surface is exactly
// zero, so a residual means some face was integrated over the wrong region or with a flipped
// orientation. The check must reject that rather than let a wrong number through.
func TestVectorAreaClosureRejectsAResidual(t *testing.T) {
	closed := massTerms{area: 100}
	if !vectorAreaCloses(closed) {
		t.Error("a body whose vector area vanishes must pass the closure check")
	}
	if vectorAreaCloses(massTerms{area: 100, ax: 1}) {
		t.Error("a body with a 1%% vector-area residual must be declined, not integrated")
	}
	if vectorAreaCloses(massTerms{}) {
		t.Error("a body with no area at all cannot be certified closed")
	}
}

// TestGreenFormFollowsTheClosingAxis: the reduction is chosen by which parameter the loops return
// to, and a loop that closes in neither is refused rather than integrated over a region that is not
// bounded in the covering space.
func TestGreenFormFollowsTheClosingAxis(t *testing.T) {
	cases := []struct {
		name     string
		loop     faceLoop
		wantDV   bool
		wantOKay bool
	}{
		{"closes both ways", faceLoop{}, true, true},
		{"wraps u, closes v", faceLoop{netU: 4 * stdmath.Pi}, false, true},
		{"wraps v, closes u", faceLoop{netV: 2 * stdmath.Pi}, true, true},
		{"wraps both", faceLoop{netU: 2 * stdmath.Pi, netV: 2 * stdmath.Pi}, false, false},
	}
	for _, c := range cases {
		form, ok := greenFormFor([]faceLoop{c.loop})
		if ok != c.wantOKay || (ok && form.dv != c.wantDV) {
			t.Errorf("%s: form.dv=%v ok=%v, want dv=%v ok=%v", c.name, form.dv, ok, c.wantDV, c.wantOKay)
		}
	}
}
