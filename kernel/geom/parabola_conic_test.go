// SPDX-License-Identifier: GPL-2.0-only

package geom_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The parabola is the THIRD conic, and it was missing from AsConic — which is not a rare case: it is
// the section a plane parallel to a cone's own generator makes. Everything that asked "is this a conic"
// answered no, so such a section fell to the polygonal route whose currency is straight segments, and
// the cut declined (ADR-0062).

func testParabola(t *testing.T) geom.Parabola {
	t.Helper()
	p, err := geom.NewParabola(math.P3(1, 2, 3), math.V3(0, 0, 1), math.V3(1, 0, 0), 2)
	if err != nil {
		t.Fatalf("NewParabola: %v", err)
	}
	return p
}

// TestAsConicTakesAParabola: it reports the parabolic form — vertex for a centre, cross direction for a
// major axis, and the focal length in place of semi-axes it does not have.
func TestAsConicTakesAParabola(t *testing.T) {
	t.Parallel()
	p := testParabola(t)
	cf, ok := geom.AsConic(p)
	if !ok {
		t.Fatal("AsConic refused a parabola")
	}
	if !cf.Parabolic || cf.Hyperbolic {
		t.Errorf("form is parabolic=%v hyperbolic=%v, want parabolic alone", cf.Parabolic, cf.Hyperbolic)
	}
	if cf.Center != p.Vertex {
		t.Errorf("centre %v, want the vertex %v", cf.Center, p.Vertex)
	}
	if cf.Focal != 2 {
		t.Errorf("focal %g, want 2", cf.Focal)
	}
	// A bounded arc runs ON the parabola, and the conic it runs on is what a clipper needs.
	if af, ok := geom.AsConic(p.Arc(-3, 4)); !ok || !af.Parabolic || af.Focal != 2 {
		t.Errorf("AsConic(arc) = %+v, ok=%v; want the same parabolic form", af, ok)
	}
}

// TestParabolaAxialAmplitudeIsUnbounded: a parabola runs to infinity along every direction in its own
// plane, exactly as a hyperbola branch does, and is flat only along its normal.
func TestParabolaAxialAmplitudeIsUnbounded(t *testing.T) {
	t.Parallel()
	cf, _ := geom.AsConic(testParabola(t))
	for _, axis := range []math.Vector3{math.V3(0, 0, 1), math.V3(1, 0, 0), math.V3(1, 0, 1)} {
		if a := cf.AxialAmplitude(axis); !stdmath.IsInf(a, 1) {
			t.Errorf("amplitude along %v = %g, want +Inf: the parabola is unbounded there", axis, a)
		}
	}
	if a := cf.AxialAmplitude(math.V3(0, 1, 0)); a != 0 {
		t.Errorf("amplitude along the parabola's normal = %g, want 0", a)
	}
}

// TestParabolaParamInvertsExactly: the parabola's own parameter IS its cross coordinate, so the
// inversion is a projection — exact, single-valued, and needing no root.
func TestParabolaParamInvertsExactly(t *testing.T) {
	t.Parallel()
	p := testParabola(t)
	for _, want := range []float64{-5, -1, 0, 0.25, 3, 7} {
		got, ok := geom.ConicParamAt(p, p.PointAt(want))
		if !ok {
			t.Fatalf("ConicParamAt refused a point on the parabola at t=%g", want)
		}
		if stdmath.Abs(got-want) > 1e-12 {
			t.Errorf("ConicParamAt = %g, want %g", got, want)
		}
	}
	arc := p.Arc(-2, 6)
	for _, s := range []float64{0, 0.25, 0.5, 1} {
		got, ok := geom.ConicParamAt(arc, arc.PointAt(s))
		if !ok || stdmath.Abs(got-s) > 1e-12 {
			t.Errorf("ConicParamAt(arc) = %g ok=%v, want %g", got, ok, s)
		}
	}
}

// TestParabolaSubArcKeepsTheCurve: restricting a parabola to a span gives an arc that agrees with it
// point for point, so a clipped section is the SAME curve the other side of the imprint carries.
func TestParabolaSubArcKeepsTheCurve(t *testing.T) {
	t.Parallel()
	p := testParabola(t)
	sub, ok := geom.ConicSubArc(p, -1, 3)
	if !ok {
		t.Fatal("ConicSubArc refused a parabola")
	}
	for k := 0; k <= 8; k++ {
		s := float64(k) / 8
		got := sub.PointAt(s)
		want := p.PointAt(-1 + s*4)
		if d := float64(got.DistanceTo(want)); d > 1e-12 {
			t.Errorf("sub-arc at s=%g is %g off the parabola", s, d)
		}
	}
	// And restricting the ARC again composes: a sub-arc of a sub-arc is still on the curve.
	again, ok := geom.ConicSubArc(sub, 0.25, 0.75)
	if !ok {
		t.Fatal("ConicSubArc refused a parabolic arc")
	}
	if d := float64(again.PointAt(0).DistanceTo(p.PointAt(0))); d > 1e-12 {
		t.Errorf("the composed sub-arc starts %g off t=0", d)
	}
}
