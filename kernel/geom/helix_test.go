// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/math"
)

// zHelix builds a helix about +Z with RefDir +X for the given parameters.
func zHelix(t *testing.T, startR, axialPerTurn, radialPerTurn, turns float64, cw bool) Helix3d {
	t.Helper()
	h, err := NewHelix3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), startR, axialPerTurn, radialPerTurn, turns, cw)
	if err != nil {
		t.Fatalf("NewHelix3d: %v", err)
	}
	return h
}

func TestNewHelix3dErrors(t *testing.T) {
	t.Parallel()
	if _, err := NewHelix3d(math.P3(0, 0, 0), math.V3(0, 0, 0), math.V3(1, 0, 0), 1, 1, 0, 3, false); err == nil {
		t.Error("a zero axis should error")
	}
	_, err := NewHelix3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 1, 1, 0, 0, false)
	if err == nil {
		t.Error("zero turns should error")
	} else if !strings.Contains(err.Error(), "turns") {
		t.Errorf("error %q should mention the offending turn count", err)
	}
	if _, err := NewHelix3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 1, 1, 0, -2, false); err == nil {
		t.Error("negative turns should error")
	}
}

// TestHelixCylindricalGeometry checks the endpoints, constant radius, and axial advance
// of a right-handed cylindrical helix.
func TestHelixCylindricalGeometry(t *testing.T) {
	t.Parallel()
	h := zHelix(t, 4, 10, 0, 3, false)

	start := h.StartPoint()
	if start.DistanceTo(math.P3(4, 0, 0)) > 1e-9 {
		t.Errorf("StartPoint = %v, want (4,0,0)", start)
	}
	end := h.EndPoint()
	// 3 full turns ⇒ angle 6π ≡ 0 ⇒ back on +X; height = pitch·turns = 30.
	if end.DistanceTo(math.P3(4, 0, 30)) > 1e-9 {
		t.Errorf("EndPoint = %v, want (4,0,30)", end)
	}
	// Radius is constant along a cylindrical helix.
	for _, tt := range []float64{0, 0.25, 0.5, 0.75, 1} {
		p := h.PointAt(tt)
		r := stdmath.Hypot(float64(p.X), float64(p.Y))
		if stdmath.Abs(r-4) > 1e-9 {
			t.Errorf("radius at t=%v = %v, want 4", tt, r)
		}
		if z := float64(p.Z); stdmath.Abs(z-30*tt) > 1e-9 {
			t.Errorf("height at t=%v = %v, want %v", tt, z, 30*tt)
		}
	}
	// A quarter turn into the first revolution sits on +Y (right-handed about +Z).
	q := h.PointAt(1.0 / 12.0) // 1/12 of 3 turns = quarter turn
	if float64(q.Y) <= 0 {
		t.Errorf("a right-handed quarter turn should have +Y, got %v", q)
	}
}

// TestHelixClockwiseFlipsWinding checks the handedness flag negates the winding sense.
func TestHelixClockwiseFlipsWinding(t *testing.T) {
	t.Parallel()
	ccw := zHelix(t, 4, 10, 0, 3, false)
	cw := zHelix(t, 4, 10, 0, 3, true)
	if float64(ccw.PointAt(1.0/12.0).Y) <= 0 || float64(cw.PointAt(1.0/12.0).Y) >= 0 {
		t.Error("clockwise should mirror the winding sign in Y")
	}
}

// TestHelixLengthCylindricalClosedForm checks the analytic length √((2πr)²+pitch²)·turns.
func TestHelixLengthCylindricalClosedForm(t *testing.T) {
	t.Parallel()
	h := zHelix(t, 4, 10, 0, 3, false)
	want := stdmath.Hypot(twoPi*4, 10) * 3
	if got := h.Length(); stdmath.Abs(got-want) > 1e-9 {
		t.Errorf("Length = %v, want %v", got, want)
	}
}

// TestHelixLengthTaperedMatchesReference cross-checks the Simpson length of a tapered
// helix against an independent fine-grained trapezoid integral.
func TestHelixLengthTaperedMatchesReference(t *testing.T) {
	t.Parallel()
	h := zHelix(t, 2, 6, 1.5, 4, false) // conical: radius grows 1.5/turn
	ref := trapezoidSpeed(h, 20000)
	if got := h.Length(); stdmath.Abs(got-ref) > 1e-6 {
		t.Errorf("tapered Length = %v, reference %v", got, ref)
	}
}

// TestHelixSpiral checks a flat spiral (no axial advance) stays in its plane and grows
// in radius.
func TestHelixSpiral(t *testing.T) {
	t.Parallel()
	h := zHelix(t, 1, 0, 2, 5, false) // pitch 0 ⇒ flat spiral, radius grows 2/turn
	for _, tt := range []float64{0, 0.5, 1} {
		if z := float64(h.PointAt(tt).Z); stdmath.Abs(z) > 1e-9 {
			t.Errorf("spiral should stay at z=0, got %v at t=%v", z, tt)
		}
	}
	r0 := stdmath.Hypot(float64(h.PointAt(0).X), float64(h.PointAt(0).Y))
	r1 := stdmath.Hypot(float64(h.PointAt(1).X), float64(h.PointAt(1).Y))
	if r0 > r1 || stdmath.Abs(r1-(1+2*5)) > 1e-9 {
		t.Errorf("spiral radius %v→%v, want growing to %v", r0, r1, 1+2*5)
	}
	if h.Length() <= 0 {
		t.Error("spiral length should be positive")
	}
}

// TestHelixTangentMatchesFiniteDifference checks the analytic TangentAt against a central
// finite difference of PointAt for cylindrical, tapered and spiral helices (the standard
// curve oracle that catches missing chain factors).
func TestHelixTangentMatchesFiniteDifference(t *testing.T) {
	t.Parallel()
	cases := []Helix3d{
		zHelix(t, 4, 10, 0, 3, false),
		zHelix(t, 4, 10, 0, 3, true),
		zHelix(t, 2, 6, 1.5, 4, false),
		zHelix(t, 1, 0, 2, 5, false),
	}
	const eps = 1e-6
	for ci, h := range cases {
		for _, tt := range []float64{0.1, 0.37, 0.5, 0.83} {
			fd := h.PointAt(tt + eps).VectorTo(h.PointAt(tt - eps)).Scale(-1).Scale(1.0 / (2 * eps))
			an := h.TangentAt(tt)
			if d := fd.Sub(an).Length(); float64(d) > 1e-4 {
				t.Errorf("case %d t=%v: tangent %v vs finite-diff %v (Δ=%v)", ci, tt, an, fd, d)
			}
		}
	}
}

// TestHelixRegularAndAdvancing checks the helix is a regular curve (positive speed
// everywhere) and advances monotonically (each consecutive sample is a positive step,
// and the accumulated polyline length approaches the analytic arc length from below).
func TestHelixRegularAndAdvancing(t *testing.T) {
	t.Parallel()
	h := zHelix(t, 3, 8, 0.5, 4, false)
	const n = 2000
	var accum float64
	prev := h.PointAt(0)
	for i := 1; i <= n; i++ {
		tt := float64(i) / n
		if s := h.speedAt(tt); s <= 0 {
			t.Fatalf("speed at t=%v = %v, want > 0 (regular curve)", tt, s)
		}
		cur := h.PointAt(tt)
		step := float64(prev.DistanceTo(cur))
		if step <= 0 {
			t.Fatalf("consecutive samples did not advance at step %d", i)
		}
		accum += step
		prev = cur
	}
	// The inscribed polyline underestimates arc length but converges to it.
	if l := h.Length(); accum > l+1e-9 || accum < l-1e-2 {
		t.Errorf("polyline length %v should approach arc length %v from below", accum, l)
	}
	if lo, hi := h.Domain(); lo != 0 || hi != 1 {
		t.Errorf("Domain = [%v,%v], want [0,1]", lo, hi)
	}
}

// trapezoidSpeed integrates |TangentAt| over [0,1] with n trapezoid intervals — an
// independent reference for the Simpson arc length.
func trapezoidSpeed(h Helix3d, n int) float64 {
	step := 1.0 / float64(n)
	sum := (h.speedAt(0) + h.speedAt(1)) / 2
	for i := 1; i < n; i++ {
		sum += h.speedAt(float64(i) * step)
	}
	return sum * step
}
