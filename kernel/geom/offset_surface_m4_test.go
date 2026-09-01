// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// Regression suite for Oblikovati/Oblikovati#1322: OffsetSurface returned the BASE ParamAt (wrong
// (u,v) off the radial/perpendicular analytic projections), used a fixed domain-unaware FD step, and
// never detected self-intersection. ParamAt now inverts onto the offset by Gauss–Newton, the FD step
// scales to the domain span, and NewOffsetSurface/SelfIntersects reject a folded offset.

// nurbsBump builds a non-trivial rational biquadratic NURBS patch on the unit domain (a saddle-ish
// bump), the non-analytic base the old ParamAt could not invert.
func nurbsBump(t *testing.T) BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{{X: 0, Y: 0, Z: 0}, {X: 0, Y: 1, Z: 1}, {X: 0, Y: 2, Z: 0}},
		{{X: 1, Y: 0, Z: 1}, {X: 1, Y: 1, Z: 2}, {X: 1, Y: 2, Z: 1}},
		{{X: 2, Y: 0, Z: 0}, {X: 2, Y: 1, Z: 1}, {X: 2, Y: 2, Z: 0}},
		{{X: 3, Y: 0, Z: -1}, {X: 3, Y: 1, Z: 0}, {X: 3, Y: 2, Z: -1}},
	}
	weights := [][]float64{{1, 2, 1}, {1, 1, 1}, {2, 1, 2}, {1, 1, 1}}
	s, err := NewBSplineSurface(2, 2, ctrl, weights,
		[]float64{0, 0, 0, 0.5, 1, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	return s
}

// TestOffsetParamAtInvertsNurbs is the core #1322 fix: for a NURBS base, ParamAt(PointAt(u,v)) must
// round-trip to (u,v) — the old o.Base.ParamAt returned the base projection (wrong) here.
func TestOffsetParamAtInvertsNurbs(t *testing.T) {
	t.Parallel()
	off := OffsetSurface{Base: nurbsBump(t), Distance: 0.25}
	for _, u := range []float64{0.2, 0.5, 0.8} {
		for _, v := range []float64{0.2, 0.5, 0.8} {
			p := off.PointAt(u, v)
			gu, gv := off.ParamAt(p)
			if stdmath.Abs(gu-u) > 1e-6 || stdmath.Abs(gv-v) > 1e-6 {
				t.Errorf("ParamAt(PointAt(%g,%g)) = (%g,%g), want (%g,%g)", u, v, gu, gv, u, v)
			}
			// And the recovered params must land back on the offset point.
			if back := off.PointAt(gu, gv); back.DistanceTo(p) > 1e-7 {
				t.Errorf("recovered (u,v) maps to a point %g away from p", back.DistanceTo(p))
			}
		}
	}
}

// fdPartialU is a 4th-order central finite-difference reference for ∂(s.PointAt)/∂u — the accurate
// derivative of the offset surface to validate DerivativesAt against.
func fdPartialU(s Surface, u, v, h float64) math.Vector3 {
	p2 := s.PointAt(u+2*h, v).AsVector()
	p1 := s.PointAt(u+h, v).AsVector()
	m1 := s.PointAt(u-h, v).AsVector()
	m2 := s.PointAt(u-2*h, v).AsVector()
	return p2.Scale(-1).Add(p1.Scale(8)).Add(m1.Scale(-8)).Add(m2).Scale(math.Scalar(1.0 / (12 * h)))
}

func fdPartialV(s Surface, u, v, h float64) math.Vector3 {
	p2 := s.PointAt(u, v+2*h).AsVector()
	p1 := s.PointAt(u, v+h).AsVector()
	m1 := s.PointAt(u, v-h).AsVector()
	m2 := s.PointAt(u, v-2*h).AsVector()
	return p2.Scale(-1).Add(p1.Scale(8)).Add(m1.Scale(-8)).Add(m2).Scale(math.Scalar(1.0 / (12 * h)))
}

func relDiff(a, b math.Vector3) float64 {
	scale := stdmath.Max(float64(a.Length()), 1e-9)
	return float64(a.Add(b.Scale(-1)).Length()) / scale
}

// TestOffsetDerivativesMatchReference checks DerivativesAt against the 4th-order FD reference on an
// analytic base ([0,2π] domain) and a NURBS base ([0,1] domain) — two parameter scales, both within
// tolerance now that the FD step scales to the domain span.
func TestOffsetDerivativesMatchReference(t *testing.T) {
	t.Parallel()
	sphere, err := NewSphere(math.P3(0, 0, 0), 4)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	cases := []struct {
		name string
		base Surface
		us   []float64
		vs   []float64
		h    float64
	}{
		{"sphere", sphere, []float64{0.8, 1.6, 2.4}, []float64{-0.6, 0.3, 0.9}, 1e-4},
		{"nurbs", nurbsBump(t), []float64{0.3, 0.6}, []float64{0.3, 0.6}, 1e-4},
	}
	for _, c := range cases {
		off := OffsetSurface{Base: c.base, Distance: 0.3}
		for _, u := range c.us {
			for _, v := range c.vs {
				du, dv := off.DerivativesAt(u, v)
				if e := relDiff(du, fdPartialU(off, u, v, c.h)); e > 1e-5 {
					t.Errorf("%s ∂u(%g,%g) rel error %g exceeds 1e-5", c.name, u, v, e)
				}
				if e := relDiff(dv, fdPartialV(off, u, v, c.h)); e > 1e-5 {
					t.Errorf("%s ∂v(%g,%g) rel error %g exceeds 1e-5", c.name, u, v, e)
				}
			}
		}
	}
}

// TestOffsetSelfIntersectionRejected checks the fold detector and validating constructor: a cylinder
// offset inward by MORE than its radius inverts (self-intersects) and must be rejected; a safe inward
// offset is accepted.
func TestOffsetSelfIntersectionRejected(t *testing.T) {
	t.Parallel()
	cyl, err := NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	folded := OffsetSurface{Base: cyl, Distance: -6} // inward past the radius → inverted
	if !folded.SelfIntersects() {
		t.Error("inward offset beyond the cylinder radius must report SelfIntersects")
	}
	if _, err := NewOffsetSurface(cyl, -6); err == nil {
		t.Error("NewOffsetSurface must reject a self-intersecting offset")
	}
	safe := OffsetSurface{Base: cyl, Distance: -2}
	if safe.SelfIntersects() {
		t.Error("inward offset within the radius must not report SelfIntersects")
	}
	if _, err := NewOffsetSurface(cyl, -2); err != nil {
		t.Errorf("NewOffsetSurface rejected a valid offset: %v", err)
	}
	// Outward offset never folds.
	if (OffsetSurface{Base: cyl, Distance: 10}).SelfIntersects() {
		t.Error("outward offset must never self-intersect")
	}
}

// TestOffsetSphereDoesNotFalseFoldAtPole guards the degenerate-point skip: a sphere's poles have a
// zero normal, which must not be misread as a fold for a safe offset.
func TestOffsetSphereDoesNotFalseFoldAtPole(t *testing.T) {
	t.Parallel()
	sphere, err := NewSphere(math.P3(0, 0, 0), 4)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	if (OffsetSurface{Base: sphere, Distance: -1}).SelfIntersects() {
		t.Error("a sphere offset well within its radius must not false-fold at the poles")
	}
	if !(OffsetSurface{Base: sphere, Distance: -5}).SelfIntersects() {
		t.Error("a sphere offset inward beyond its radius must self-intersect")
	}
}
