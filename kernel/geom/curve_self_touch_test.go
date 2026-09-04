// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// lemniscate is a figure-eight that touches itself at the origin at t = 0 and t = π (Bernoulli's,
// parameterised so the touch is TANGENTIAL in the sense this solves for: the two branches pass
// through one point).
type lemniscate struct{ a float64 }

func (l lemniscate) Domain() (float64, float64) { return 0, 2 * stdmath.Pi }

func (l lemniscate) PointAt(t float64) math.Point3 {
	d := 1 + stdmath.Sin(t)*stdmath.Sin(t)
	return math.P3(math.Scalar(l.a*stdmath.Cos(t)/d), math.Scalar(l.a*stdmath.Sin(t)*stdmath.Cos(t)/d), 0)
}

func (l lemniscate) TangentAt(t float64) math.Vector3 {
	const h = 1e-6
	a, b := l.PointAt(t-h), l.PointAt(t+h)
	return a.VectorTo(b).Scale(math.Scalar(1 / (2 * h)))
}

func TestCurveSelfTouchFindsTheCrossingOfAFigureEight(t *testing.T) {
	t.Parallel()
	a, b, ok := CurveSelfTouch(lemniscate{a: 4}, 1e-9)
	if !ok {
		t.Fatal("a lemniscate touches itself at its centre and was reported not to")
	}
	// The two visits are at t = π/2 and t = 3π/2, where cos t = 0.
	for _, got := range []float64{a, b} {
		if stdmath.Abs(stdmath.Cos(got)) > 1e-6 {
			t.Errorf("touch parameter %g is not a visit to the centre (cos = %g)", got, stdmath.Cos(got))
		}
	}
	if d := float64(lemniscate{a: 4}.PointAt(a).DistanceTo(lemniscate{a: 4}.PointAt(b))); d > 1e-9 {
		t.Errorf("the two visits are %g apart, want coincident", d)
	}
}

// A circle returns to its start and nowhere else: its closure is not a self-touch.
func TestCurveSelfTouchIgnoresAClosedCurvesOwnClosure(t *testing.T) {
	t.Parallel()
	c, err := NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	if err != nil {
		t.Fatalf("circle: %v", err)
	}
	if a, b, ok := CurveSelfTouch(c, 1e-9); ok {
		t.Errorf("a circle was reported to touch itself at %g, %g", a, b)
	}
}

// An open curve that never returns has no touch.
func TestCurveSelfTouchIsFalseForASimpleArc(t *testing.T) {
	t.Parallel()
	arc, err := NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 2, 0, stdmath.Pi/2)
	if err != nil {
		t.Fatalf("arc: %v", err)
	}
	if a, b, ok := CurveSelfTouch(arc, 1e-9); ok {
		t.Errorf("a quarter arc was reported to touch itself at %g, %g", a, b)
	}
}
