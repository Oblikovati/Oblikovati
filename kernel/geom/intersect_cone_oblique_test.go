// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// An oblique plane tilted STEEPER than the cone's generators (relative to the axis) cuts a closed
// ELLIPSE on one nappe (Oblikovati/Oblikovati#1375). Every sampled point must lie on BOTH the cone
// surface and the plane, and the curve must be a geom.EllipseFull.
func TestPlaneConeObliqueEllipse(t *testing.T) {
	cone, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/6) // α = 30°
	pl, _ := NewPlane(math.P3(0, 0, 5), math.V3(1, 0, 1))                // tilt 45° > 30° → ellipse on the +z nappe
	curves, handled := IntersectSurfacesAnalytic(pl, cone, ResolutionForSize(1))
	if !handled || len(curves) != 1 {
		t.Fatalf("imprint handled=%v curves=%d, want one curve", handled, len(curves))
	}
	el, ok := curves[0].(EllipseFull)
	if !ok {
		t.Fatalf("imprint curve is %T, want EllipseFull", curves[0])
	}
	if el.MajorRadius < el.MinorRadius {
		t.Errorf("major radius %g < minor %g", el.MajorRadius, el.MinorRadius)
	}
	for i := 0; i < 12; i++ {
		assertOnConeAndPlane(t, cone, pl, el.PointAt(float64(i)/12))
	}
}

// An oblique plane tilted SHALLOWER than the generators cuts a HYPERBOLA; the returned branch lies on
// the cone's real (+axis) nappe, every point on both surfaces.
func TestPlaneConeObliqueHyperbola(t *testing.T) {
	cone, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/3) // α = 60°, steep cone
	pl, _ := NewPlane(math.P3(3, 0, 0), math.V3(1, 0, 0.6))              // tilt ~31° < 60° → hyperbola
	curves, handled := IntersectSurfacesAnalytic(pl, cone, ResolutionForSize(1))
	if !handled || len(curves) != 1 {
		t.Fatalf("imprint handled=%v curves=%d, want one curve", handled, len(curves))
	}
	h, ok := curves[0].(Hyperbola)
	if !ok {
		t.Fatalf("imprint curve is %T, want Hyperbola", curves[0])
	}
	if v := float64(cone.Apex.VectorTo(h.PointAt(0)).Dot(cone.AxisDir.AsVector())); v <= 0 {
		t.Errorf("hyperbola vertex on the wrong nappe (apex-distance %g ≤ 0)", v)
	}
	for _, theta := range []float64{-1.2, -0.4, 0, 0.4, 1.2} {
		assertOnConeAndPlane(t, cone, pl, h.PointAt(theta))
	}
}

// The parabolic boundary tilt (plane parallel to a generator, φ = α) cuts an exact PARABOLA; every
// sampled point lies on both the cone and the plane (Oblikovati/Oblikovati#1375). A cone whose axis is
// tilted by its own half-angle has one generator vertical, so an axis-aligned x-plane is parallel to it.
func TestPlaneConeParabola(t *testing.T) {
	a := stdmath.Atan(0.3)
	cone, _ := NewCone(math.P3(0, 0, 0), math.V3(stdmath.Sin(a), 0, stdmath.Cos(a)), a)
	pl, _ := NewPlane(math.P3(2, 0, 0), math.V3(1, 0, 0)) // x=2, parallel to the vertical generator
	curves, handled := IntersectSurfacesAnalytic(pl, cone, ResolutionForSize(1))
	if !handled || len(curves) != 1 {
		t.Fatalf("imprint handled=%v curves=%d, want one curve", handled, len(curves))
	}
	par, ok := curves[0].(Parabola)
	if !ok {
		t.Fatalf("imprint curve is %T, want Parabola", curves[0])
	}
	for _, tt := range []float64{-3, -1, 0, 1, 3} {
		assertOnConeAndPlane(t, cone, pl, par.PointAt(tt))
	}
}

// A plane through the apex is degenerate (the section collapses through the centre) and defers.
func TestPlaneConeObliqueThroughApexDeferred(t *testing.T) {
	cone, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/6)
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(1, 0, 1)) // oblique AND through the apex
	if _, handled := IntersectSurfacesAnalytic(pl, cone, ResolutionForSize(1)); handled {
		t.Error("oblique plane through the apex should defer (handled=false)")
	}
}

// A hyperbolic oblique cut on the OPPOSITE side of the axis (x = −3) exercises the branch-direction
// flip: the transverse axis must still point to the vertex on the cone's real nappe.
func TestPlaneConeObliqueHyperbolaOtherSide(t *testing.T) {
	cone, _ := NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), stdmath.Pi/3)
	pl, _ := NewPlane(math.P3(-3, 0, 0), math.V3(1, 0, -0.6)) // mirror of the +x hyperbola case
	curves, handled := IntersectSurfacesAnalytic(pl, cone, ResolutionForSize(1))
	if !handled || len(curves) != 1 {
		t.Fatalf("imprint handled=%v curves=%d, want one curve", handled, len(curves))
	}
	h := curves[0].(Hyperbola)
	if v := float64(cone.Apex.VectorTo(h.PointAt(0)).Dot(cone.AxisDir.AsVector())); v <= 0 {
		t.Errorf("hyperbola vertex on the wrong nappe (apex-distance %g ≤ 0)", v)
	}
	for _, theta := range []float64{-1, 0, 1} {
		assertOnConeAndPlane(t, cone, pl, h.PointAt(theta))
	}
}

// symmetricEig2 on an already-diagonal form (h=0) returns axis-aligned eigenvectors — the branch
// eigenvector2 takes when the quadratic form has no cross term.
func TestSymmetricEig2Diagonal(t *testing.T) {
	l1, l2, u1, u2 := symmetricEig2(4, 0, 1)
	if stdmath.Abs(l1-4) > 1e-12 || stdmath.Abs(l2-1) > 1e-12 {
		t.Errorf("eigenvalues (%g,%g), want (4,1)", l1, l2)
	}
	if stdmath.Abs(u1[0]-1) > 1e-12 || stdmath.Abs(u1[1]) > 1e-12 {
		t.Errorf("u1 = %v, want (1,0)", u1)
	}
	if stdmath.Abs(u2[0]) > 1e-12 || stdmath.Abs(stdmath.Abs(u2[1])-1) > 1e-12 {
		t.Errorf("u2 = %v, want ±(0,1)", u2)
	}
	// a < c: the larger eigenvalue now matches the SECOND diagonal entry, so the eigenvector is (0,1).
	_, _, v1, _ := symmetricEig2(1, 0, 4)
	if stdmath.Abs(v1[0]) > 1e-12 || stdmath.Abs(v1[1]-1) > 1e-12 {
		t.Errorf("v1 = %v, want (0,1)", v1)
	}
}

// assertOnConeAndPlane fails unless p lies on both the cone surface (radial residual ≈ 0 on the
// +nappe) and the cutting plane (signed distance ≈ 0).
func assertOnConeAndPlane(t *testing.T, cone Cone, pl Plane, p math.Point3) {
	t.Helper()
	if off := coneRadialResidual(cone, p); off > 1e-7 {
		t.Errorf("point %v not on cone (residual=%g)", p, off)
	}
	if d := float64(pl.Origin.VectorTo(p).Dot(pl.Normal())); stdmath.Abs(d) > 1e-7 {
		t.Errorf("point %v not on plane (signed distance=%g)", p, d)
	}
}
