// SPDX-License-Identifier: GPL-2.0-only

package geom_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

func mustCone(t *testing.T, apexZ float64) geom.Cone {
	t.Helper()
	c, err := geom.NewCone(math.P3(0, 0, math.Scalar(apexZ)), math.V3(0, 0, -1), stdmath.Pi/4)
	if err != nil {
		t.Fatalf("NewCone: %v", err)
	}
	return c
}

// TestSurfacesApartParallelCones is the case that matters: an emboss pad's seat is the host cone
// with a shifted apex, so the two are everywhere |Δ|·sin(halfAngle) apart and no patch of one can
// touch a patch of the other, however their boxes overlap.
func TestSurfacesApartParallelCones(t *testing.T) {
	t.Parallel()
	host := mustCone(t, 50)
	seat := mustCone(t, 49.9)
	gap := 0.1 * stdmath.Sin(stdmath.Pi/4) // ≈0.0707
	if !geom.SurfacesApart(host, seat, gap*0.9) {
		t.Error("parallel cones 0.0707 apart reported not-apart at a 0.0636 gap")
	}
	if geom.SurfacesApart(host, seat, gap*1.1) {
		t.Error("parallel cones must NOT be reported apart by more than their true separation")
	}
	if geom.SurfacesApart(host, host, 1e-12) {
		t.Error("a cone is not apart from itself")
	}
}

// TestSurfacesApartRefusesIntersectingPairs pins the one-sided contract: false must mean "no
// proof", and it must never claim separation for surfaces that meet.
func TestSurfacesApartRefusesIntersectingPairs(t *testing.T) {
	t.Parallel()
	// Coaxial cones of DIFFERENT half-angle meet in a circle.
	wide, err := geom.NewCone(math.P3(0, 0, 50), math.V3(0, 0, -1), stdmath.Pi/3)
	if err != nil {
		t.Fatalf("NewCone: %v", err)
	}
	if geom.SurfacesApart(mustCone(t, 49.9), wide, 1e-9) {
		t.Error("cones of different half-angle intersect; they must not be proven apart")
	}
	// A cone and a cylinder: no closed form here, so no proof.
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	if geom.SurfacesApart(mustCone(t, 50), cyl, 1e-9) {
		t.Error("a cone and a coaxial cylinder meet in a circle; they must not be proven apart")
	}
}

// TestSurfacesApartCoaxialCylindersAndParallelPlanes covers the other two closed forms, including
// the parallel-but-not-coaxial cylinders the proof deliberately does NOT claim.
func TestSurfacesApartCoaxialCylindersAndParallelPlanes(t *testing.T) {
	t.Parallel()
	inner, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	outer, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 7)
	offset, _ := geom.NewCylinder(math.P3(3, 0, 0), math.V3(0, 0, 1), 5)
	if !geom.SurfacesApart(inner, outer, 1.5) {
		t.Error("coaxial cylinders r=5 and r=7 are 2 apart")
	}
	if geom.SurfacesApart(inner, offset, 1e-9) {
		t.Error("cylinders on DIFFERENT parallel axes are not proven here, even when apart")
	}
	pa, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	pb, _ := geom.NewPlane(math.P3(0, 0, 3), math.V3(0, 0, 1))
	pc, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(1, 0, 0))
	if !geom.SurfacesApart(pa, pb, 2.5) {
		t.Error("parallel planes 3 apart")
	}
	if geom.SurfacesApart(pa, pc, 1e-9) {
		t.Error("perpendicular planes meet in a line")
	}
}

// TestAsConicCoversThePlaneQuadricSections pins the one description all three sections share, and
// the amplitude contract a wall band reads them through.
func TestAsConicCoversThePlaneQuadricSections(t *testing.T) {
	t.Parallel()
	axis := math.V3(0, 0, 1)
	circ, err := geom.NewCircle(math.P3(0, 0, 4), math.V3(0, 0, 1), 3)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	cf, ok := geom.AsConic(circ)
	if !ok || cf.Hyperbolic || cf.A != 3 || cf.B != 3 {
		t.Fatalf("AsConic(circle) = %+v, ok=%v", cf, ok)
	}
	if amp := cf.AxialAmplitude(axis); amp > 1e-12 {
		t.Errorf("a circle perpendicular to the axis has amplitude %v, want 0", amp)
	}
	hyp, err := geom.NewHyperbola(math.P3(0, 0, 0), math.V3(1, 0, 0), math.V3(0, 0, 1), 2, 3)
	if err != nil {
		t.Fatalf("NewHyperbola: %v", err)
	}
	hf, ok := geom.AsConic(hyp)
	if !ok || !hf.Hyperbolic {
		t.Fatalf("AsConic(hyperbola) = %+v, ok=%v — the branch must be recognised", hf, ok)
	}
	// A branch runs to infinity along the axis, so its half-extent IS infinite. It used to report 0
	// with a caveat that callers must read that as "no amplitude"; a caller read it as "flat" and a
	// cone cut parallel to its axis lost the tool's face (ADR-0062).
	if amp := hf.AxialAmplitude(axis); !stdmath.IsInf(amp, 1) {
		t.Errorf("an UNBOUNDED branch reports amplitude %v; it reaches every v, so the half-extent is +Inf", amp)
	}
	if _, ok := geom.AsConic(geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0))); ok {
		t.Error("a line segment is not a conic")
	}
}
