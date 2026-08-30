// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	m "oblikovati.org/math"
)

const massRelTol = 1e-6 // analytic integration must reach the closed form to ~1e-6 relative

func relErrMP(got, want float64) float64 {
	if want == 0 {
		return stdmath.Abs(got)
	}
	return stdmath.Abs(got-want) / stdmath.Abs(want)
}

// A box is six planar faces; the analytic (exact polygon) integration must reproduce the closed
// forms for volume, area, centroid and per-unit-density inertia to machine precision.
func TestAnalyticBoxMatchesClosedForm(t *testing.T) {
	a, b, c := 4.0, 2.0, 1.0
	body, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(m.Scalar(a), m.Scalar(b), m.Scalar(c)), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	gp, ok := AnalyticGeometryProperties(body)
	if !ok {
		t.Fatal("box not analytically integrable")
	}
	wantVol := a * b * c
	wantArea := 2 * (a*b + b*c + c*a)
	if e := relErrMP(gp.Volume, wantVol); e > 1e-10 {
		t.Errorf("Volume = %v, want %v (rel %g)", gp.Volume, wantVol, e)
	}
	if e := relErrMP(gp.Area, wantArea); e > 1e-10 {
		t.Errorf("Area = %v, want %v (rel %g)", gp.Area, wantArea, e)
	}
	for _, d := range []struct {
		got, want float64
		name      string
	}{{float64(gp.Centroid.X), a / 2, "cx"}, {float64(gp.Centroid.Y), b / 2, "cy"}, {float64(gp.Centroid.Z), c / 2, "cz"}} {
		if stdmath.Abs(d.got-d.want) > 1e-9 {
			t.Errorf("Centroid.%s = %v, want %v", d.name, d.got, d.want)
		}
	}
	it, _ := AnalyticInertia(body)
	// Per-unit-density inertia about the centroid: Ixx = V(b²+c²)/12, etc.
	wantIxx := wantVol * (b*b + c*c) / 12
	wantIyy := wantVol * (a*a + c*c) / 12
	wantIzz := wantVol * (a*a + b*b) / 12
	if e := relErrMP(it.Ixx, wantIxx); e > 1e-9 {
		t.Errorf("Ixx = %v, want %v (rel %g)", it.Ixx, wantIxx, e)
	}
	if e := relErrMP(it.Iyy, wantIyy); e > 1e-9 {
		t.Errorf("Iyy = %v, want %v (rel %g)", it.Iyy, wantIyy, e)
	}
	if e := relErrMP(it.Izz, wantIzz); e > 1e-9 {
		t.Errorf("Izz = %v, want %v (rel %g)", it.Izz, wantIzz, e)
	}
	if stdmath.Abs(it.Ixy) > 1e-9*wantIzz || stdmath.Abs(it.Iyz) > 1e-9*wantIzz || stdmath.Abs(it.Izx) > 1e-9*wantIzz {
		t.Errorf("products of inertia not ~0: %+v", it)
	}
}

// A sphere is one boundary-less analytic face — the full-domain quadrature must recover
// V=4/3πr³, A=4πr² and I=2/5·V·r² about the centre.
func TestAnalyticSphereMatchesClosedForm(t *testing.T) {
	r := 3.0
	body, err := brep.SolidSphere(m.P3(0, 0, 0), r, "sphere")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	gp, ok := AnalyticGeometryProperties(body)
	if !ok {
		t.Fatal("sphere not analytically integrable")
	}
	if e := relErrMP(gp.Volume, 4.0/3*stdmath.Pi*r*r*r); e > massRelTol {
		t.Errorf("Volume rel err %g", e)
	}
	if e := relErrMP(gp.Area, 4*stdmath.Pi*r*r); e > massRelTol {
		t.Errorf("Area rel err %g", e)
	}
	it, _ := AnalyticInertia(body)
	wantI := 2.0 / 5 * (4.0 / 3 * stdmath.Pi * r * r * r) * r * r
	for _, d := range []struct {
		got  float64
		name string
	}{{it.Ixx, "Ixx"}, {it.Iyy, "Iyy"}, {it.Izz, "Izz"}} {
		if e := relErrMP(d.got, wantI); e > 1e-5 {
			t.Errorf("%s = %v, want %v (rel %g)", d.name, d.got, wantI, e)
		}
	}
}

// A cylinder is two planar caps plus a periodic side face whose loop wraps the seam — the uv
// reconstruction and Green's-theorem integral must recover the closed forms.
func TestAnalyticCylinderMatchesClosedForm(t *testing.T) {
	r, h := 2.0, 5.0
	body, err := brep.SolidCylinder(m.P3(0, 0, 0), m.V3(0, 0, 1), r, h)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	gp, ok := AnalyticGeometryProperties(body)
	if !ok {
		t.Fatal("cylinder not analytically integrable")
	}
	wantVol := stdmath.Pi * r * r * h
	wantArea := 2*stdmath.Pi*r*r + 2*stdmath.Pi*r*h
	if e := relErrMP(gp.Volume, wantVol); e > massRelTol {
		t.Errorf("Volume = %v, want %v (rel %g)", gp.Volume, wantVol, e)
	}
	if e := relErrMP(gp.Area, wantArea); e > massRelTol {
		t.Errorf("Area = %v, want %v (rel %g)", gp.Area, wantArea, e)
	}
	if stdmath.Abs(float64(gp.Centroid.Z)-h/2) > 1e-6 ||
		stdmath.Abs(float64(gp.Centroid.X)) > 1e-6 || stdmath.Abs(float64(gp.Centroid.Y)) > 1e-6 {
		t.Errorf("Centroid = %v, want (0,0,%v)", gp.Centroid, h/2)
	}
	it, _ := AnalyticInertia(body)
	wantIzz := wantVol * r * r / 2
	wantIxx := wantVol * (3*r*r + h*h) / 12
	if e := relErrMP(it.Izz, wantIzz); e > 1e-5 {
		t.Errorf("Izz = %v, want %v (rel %g)", it.Izz, wantIzz, e)
	}
	if e := relErrMP(it.Ixx, wantIxx); e > 1e-5 {
		t.Errorf("Ixx = %v, want %v (rel %g)", it.Ixx, wantIxx, e)
	}
}

// A torus is one boundary-less face; check volume 2π²Rr² and area 4π²Rr.
func TestAnalyticTorusMatchesClosedForm(t *testing.T) {
	bigR, smallR := 5.0, 1.0
	body, err := brep.SolidTorus(m.P3(0, 0, 0), m.V3(0, 0, 1), bigR, smallR, "torus")
	if err != nil {
		t.Fatalf("SolidTorus: %v", err)
	}
	gp, ok := AnalyticGeometryProperties(body)
	if !ok {
		t.Fatal("torus not analytically integrable")
	}
	if e := relErrMP(gp.Volume, 2*stdmath.Pi*stdmath.Pi*bigR*smallR*smallR); e > massRelTol {
		t.Errorf("Volume rel err %g", e)
	}
	if e := relErrMP(gp.Area, 4*stdmath.Pi*stdmath.Pi*bigR*smallR); e > massRelTol {
		t.Errorf("Area rel err %g", e)
	}
}

// The analytic volume must be translation-invariant (the divergence integrand references the
// origin, but the off-origin parts cancel).
func TestAnalyticVolumeTranslationInvariant(t *testing.T) {
	at := func(x, y, z float64) float64 {
		body, _ := brep.SolidCylinder(m.P3(m.Scalar(x), m.Scalar(y), m.Scalar(z)), m.V3(0, 0, 1), 2, 5)
		gp, ok := AnalyticGeometryProperties(body)
		if !ok {
			t.Fatal("cylinder not integrable")
		}
		return gp.Volume
	}
	if e := relErrMP(at(100, -50, 25), at(0, 0, 0)); e > 1e-9 {
		t.Errorf("volume changed under translation: rel %g", e)
	}
}

// Cross-check the analytic path against a fine tessellation for a cone frustum (a bounded
// curved side face). The analytic value is the exact one; the fine mesh should be close.
func TestAnalyticConeAgreesWithFineMesh(t *testing.T) {
	body, err := brep.SolidCylinderCone(m.P3(0, 0, 0), m.P3(0, 0, 8), 3, 1.5, "frustum")
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	gp, ok := AnalyticGeometryProperties(body)
	if !ok {
		t.Fatal("cone not analytically integrable")
	}
	// Closed-form frustum volume: πh(R²+Rr+r²)/3.
	wantVol := stdmath.Pi * 8 * (9 + 4.5 + 2.25) / 3
	if e := relErrMP(gp.Volume, wantVol); e > massRelTol {
		t.Errorf("Volume = %v, want %v (rel %g)", gp.Volume, wantVol, e)
	}
	mesh, _ := TessellateBody(body, Quality{ChordTolerance: 1e-4, AngleTolerance: 0.5 * stdmath.Pi / 180})
	meshVol := meshGeometryProperties(mesh).Volume
	if e := relErrMP(gp.Volume, meshVol); e > 1e-3 {
		t.Errorf("analytic %v vs fine mesh %v disagree (rel %g)", gp.Volume, meshVol, e)
	}
}
