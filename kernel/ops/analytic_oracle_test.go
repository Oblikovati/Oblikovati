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
