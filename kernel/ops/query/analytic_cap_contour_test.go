// SPDX-License-Identifier: GPL-2.0-only

package query_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A CAP — a sphere face bounded by one latitude circle — could not be integrated at all. Its rim wraps
// the azimuth and returns to its own latitude, so the ∮ −P du reduction's antiderivative vanishes
// identically on the only contour there was, and the region measured nothing. The contour was
// incomplete, not the reduction: a cap's boundary in the parameter rectangle is its rim AND the line at
// the pole, which is one point in space but a full 2π of parameter — the degenerate edge OCCT stores
// explicitly, and the segment sphereFaceUV already adds when it charts a sphere (ADR-0062).
//
// Every body with a cap in it was measured by TESSELLATION instead, which the ground rules forbid as an
// oracle: "mass properties integrate the analytic B-rep".

const capR = 5.0

// hemisphereBelowZ is the lower half of a sphere of radius capR.
func hemisphereBelowZ(t *testing.T) *topo.Body {
	t.Helper()
	sphere, err := brep.SolidSphere(math.P3(0, 0, 0), capR, "s")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	hemi, err := brep.HalfSpaceCut(sphere, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	return hemi
}

// TestCapFaceAreaIsExact: a hemisphere's spherical face is 2πR², and it must come from the analytic
// integral rather than from a mesh.
func TestCapFaceAreaIsExact(t *testing.T) {
	t.Parallel()
	for _, f := range hemisphereBelowZ(t).Faces() {
		if _, ok := f.Geometry().(geom.Sphere); !ok {
			continue
		}
		got, ok := query.AnalyticFaceArea(f)
		if !ok {
			t.Fatal("a hemisphere's face declined the analytic integral")
		}
		if want := 2 * stdmath.Pi * capR * capR; stdmath.Abs(got-want) > 1e-6 {
			t.Errorf("cap area = %.6f, want %.6f (2πR²)", got, want)
		}
		return
	}
	t.Fatal("the hemisphere has no spherical face")
}

// TestCapBodyVolumeIsExact: and the body it closes measures ⅔πR³ analytically, not approximately.
func TestCapBodyVolumeIsExact(t *testing.T) {
	t.Parallel()
	p, ok := query.AnalyticGeometryProperties(hemisphereBelowZ(t))
	if !ok {
		t.Fatal("a hemisphere declined the analytic integral")
	}
	if want := 2.0 / 3.0 * stdmath.Pi * capR * capR * capR; stdmath.Abs(p.Volume-want) > 1e-6 {
		t.Errorf("hemisphere volume = %.6f, want %.6f (⅔πR³)", p.Volume, want)
	}
}

// TestSphereBoxCornerIsExactlyAnalytic: two composed cuts leave a patch bounded by an equator arc and a
// cap arc — still one wrapping rim, so still a contour the pole closes. Every face of it integrates
// exactly, which is what takes the whole body off the tessellated fallback.
func TestSphereBoxCornerIsExactlyAnalytic(t *testing.T) {
	t.Parallel()
	pX, err := geom.NewPlane(math.P3(2, 0, 0), math.V3(1, 0, 0))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	corner, err := brep.HalfSpaceCut(hemisphereBelowZ(t), pX)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	capV := stdmath.Pi * 9 * (3*capR - 3) / 3
	want := (2.0/3.0)*stdmath.Pi*capR*capR*capR - capV/2
	p, ok := query.AnalyticGeometryProperties(corner)
	if !ok {
		t.Fatal("the sphere∩box corner declined the analytic integral")
	}
	if stdmath.Abs(p.Volume-want) > 1e-6 {
		t.Errorf("corner volume = %.6f, want %.6f", p.Volume, want)
	}
	// The gate is per-face, not a whole-body smoke test: a hemisphere minus the z<0 half of the x>2
	// cap, and the two planar lids.
	for _, f := range corner.Faces() {
		got, ok := query.AnalyticFaceArea(f)
		if !ok {
			t.Fatalf("%T declined", f.Geometry())
		}
		var face float64
		switch f.Geometry().(type) {
		case geom.Sphere:
			face = 2*stdmath.Pi*capR*capR - 2*stdmath.Pi*capR*3/2
		default:
			continue
		}
		if stdmath.Abs(got-face) > 1e-6 {
			t.Errorf("sphere patch area = %.6f, want %.6f", got, face)
		}
	}
}

// TestTorusVLimitIsNoPole: a torus's v limit is a PERIOD, not a pole — it closes onto itself, so there
// is no line to close a contour with and a torus band must not be given one. Its two rims close each
// other, and its perpendicular cut still measures exactly.
func TestTorusVLimitIsNoPole(t *testing.T) {
	t.Parallel()
	tor, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "t")
	if err != nil {
		t.Fatalf("SolidTorus: %v", err)
	}
	plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	half, err := brep.HalfSpaceCut(tor, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	got := query.BodyGeometryProperties(half, ops.DefaultQuality()).Volume
	if want := stdmath.Pi * stdmath.Pi * 5 * 4; stdmath.Abs(got-want)/want > 1e-6 {
		t.Errorf("half torus volume = %.6f, want %.6f (π²Rr²)", got, want)
	}
}
