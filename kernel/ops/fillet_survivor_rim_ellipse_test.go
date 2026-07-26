// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// White-box tests for the ELLIPSE arm of the curved-survivor rim carry
// (fillet_survivor_rim_ellipse.go). Each pins a helper against its closed form on the corpus's own
// fixture geometry: the F6/F7 oblique elliptic prism's top cap rim — `ellipse 15 10` × `tscale 10`, so
// major radius 150, minor 100, centred at (20,0,100) — whose upper half is the survivor rim the corner
// substitution re-trims.

// f6CapRim is the F6/F7 top cap rim: the upper half-ellipse from (170,0,100) through (20,100,100) to
// (-130,0,100), the survivor curve whose retained span the carry must re-derive.
func f6CapRim(t *testing.T) geom.EllipticalArc {
	t.Helper()
	ea, err := geom.NewEllipticalArc(math.P3(20, 0, 100), math.V3(0, 0, 1), math.V3(1, 0, 0), 150, 100, 0, stdmath.Pi)
	if err != nil {
		t.Fatalf("build F6 cap rim: %v", err)
	}
	return ea
}

// TestProjectOntoEllipseLandsOnTheEllipse pins projectOntoEllipse: a point already on the rim is returned
// to machine precision, and a point pushed OFF the rim (radially in the D⁻¹-scaled frame) lands back on it
// at the SAME eccentric angle — the property the sub-span algebra depends on.
func TestProjectOntoEllipseLandsOnTheEllipse(t *testing.T) {
	ea := f6CapRim(t)
	for _, theta := range []float64{0, 0.4, 1.2, stdmath.Pi / 2, 2.5, stdmath.Pi} {
		on := ea.PointAt(theta / stdmath.Pi)
		if d := float64(on.DistanceTo(projectOntoEllipse(ea, on))); d > 1e-12 {
			t.Errorf("theta=%.3f: an ON-rim point moved %.6g under projection, want ≤1e-12", theta, d)
		}
		// Push the point 25 out along the (scaled-frame) radial direction: cosθ/sinθ are unchanged, so
		// the projection must return exactly the on-rim point.
		off := math.P3(float64(on.X)+25*stdmath.Cos(theta), float64(on.Y)+25*stdmath.Sin(theta)*100.0/150.0, 100)
		back := projectOntoEllipse(ea, off)
		if d := float64(back.DistanceTo(on)); d > 1e-9 {
			t.Errorf("theta=%.3f: projecting an off-rim point gave (%.6g,%.6g), want the rim point (%.6g,%.6g) — off by %.6g",
				theta, back.X, back.Y, on.X, on.Y, d)
		}
	}
}

// TestProjectOntoEllipseReturnsCentreUnchanged pins the documented degeneracy: the ellipse's own centre has
// no eccentric angle, so it is returned unchanged rather than snapped to an arbitrary rim point.
func TestProjectOntoEllipseReturnsCentreUnchanged(t *testing.T) {
	ea := f6CapRim(t)
	if got := projectOntoEllipse(ea, ea.Center); got != ea.Center {
		t.Errorf("projecting the centre gave (%.6g,%.6g,%.6g), want it unchanged", got.X, got.Y, got.Z)
	}
}

// TestEllipseSpanIsExactSeparatesOnAndOffRim pins the carry's gate: a span whose endpoints are ON the rim
// is exact (so the parent's sub-span IS the wall's boundary), one pulled off it is not.
func TestEllipseSpanIsExactSeparatesOnAndOffRim(t *testing.T) {
	ea := f6CapRim(t)
	on0, on1 := ea.PointAt(0.1), ea.PointAt(0.8)
	if !ellipseSpanIsExact(ea, on0, on1) {
		t.Error("an on-rim span was reported inexact")
	}
	nudged := math.P3(on1.X, on1.Y+math.Scalar(1e-3), on1.Z) // 1e-3 off, 1e-5 of the major radius
	if ellipseSpanIsExact(ea, on0, nudged) {
		t.Error("a span with an endpoint 1e-3 off the rim was reported exact")
	}
}

// TestRetainedEllipticRimCurveKeepsTheParentSpan pins the carried sub-arc: it starts and ends at the
// segment's own endpoints and stays ON the parent ellipse in between — the invariant that makes the wall's
// boundary faithful. The 89.44-off chord it replaces is the whole point of the fix.
func TestRetainedEllipticRimCurveKeepsTheParentSpan(t *testing.T) {
	ea := f6CapRim(t)
	p0, p1 := ea.PointAt(0.06), ea.PointAt(0.94) // an 88%-of-the-rim retained span, as F7's is
	sub := retainedEllipticRimCurve(ea, p0, p1)
	if sub == nil {
		t.Fatal("retainedEllipticRimCurve declined an exactly-on-rim span")
	}
	lo, hi := sub.Domain()
	if d := float64(sub.PointAt(lo).DistanceTo(p0)); d > 1e-9 {
		t.Errorf("sub-arc starts %.6g from the segment's own start", d)
	}
	if d := float64(sub.PointAt(hi).DistanceTo(p1)); d > 1e-9 {
		t.Errorf("sub-arc ends %.6g from the segment's own end", d)
	}
	chord := float64(p0.DistanceTo(p1))
	for i := 1; i < 32; i++ {
		p := sub.PointAt(lo + (hi-lo)*float64(i)/32)
		if d := float64(p.DistanceTo(projectOntoEllipse(ea, p))); d > 1e-9 {
			t.Fatalf("sub-arc station %d sits %.6g OFF the parent ellipse", i, d)
		}
		if float64(p.DistanceTo(p0))+float64(p.DistanceTo(p1)) <= chord*(1+1e-12) {
			t.Fatalf("sub-arc station %d is ON the chord — the rim degraded to a straight line", i)
		}
	}
}

// TestRetainedEllipticRimCurveDeclinesOffRimSpan pins the decline: an endpoint that is NOT on the parent
// keeps the base straight chord (nil), the pre-fix behaviour, rather than inventing a span.
func TestRetainedEllipticRimCurveDeclinesOffRimSpan(t *testing.T) {
	ea := f6CapRim(t)
	off := math.P3(ea.PointAt(0.8).X, ea.PointAt(0.8).Y+math.Scalar(5), 100)
	if got := retainedEllipticRimCurve(ea, ea.PointAt(0.1), off); got != nil {
		t.Errorf("an off-rim span was carried as %T, want nil (the base chord)", got)
	}
}

// TestEllipseSpansItsSegmentDetectsAnOvershoot pins the consistency test alignCarriedEllipse gates on: a
// parent carried WHOLE past a pulled-back loop vertex must be reported as not spanning its segment.
func TestEllipseSpansItsSegmentDetectsAnOvershoot(t *testing.T) {
	ea := f6CapRim(t)
	lo, hi := ea.Domain()
	if !ellipseSpansItsSegment(ea, ea.PointAt(lo), ea.PointAt(hi)) {
		t.Error("the parent was reported as not spanning its own endpoints")
	}
	if ellipseSpansItsSegment(ea, ea.PointAt(lo), ea.PointAt(0.7)) {
		t.Error("a parent overshooting its segment's end by 30% of its sweep was reported as spanning it")
	}
}

// TestCarriableRimNamesTheTrimmableKinds pins the single place the carry's admissible parent set is
// declared: circular and elliptic arcs are carried, every other kind keeps the base straight chord.
func TestCarriableRimNamesTheTrimmableKinds(t *testing.T) {
	arc, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 50, 0, 1)
	if err != nil {
		t.Fatalf("build arc: %v", err)
	}
	for _, tc := range []struct {
		name string
		in   geom.Curve3
		want bool
	}{
		{"Arc3d", arc, true},
		{"EllipticalArc", f6CapRim(t), true},
		{"LineSegment", geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0)), false},
		{"nil", nil, false},
	} {
		got, ok := carriableRim(tc.in)
		if ok != tc.want {
			t.Errorf("carriableRim(%s) = %v, want %v", tc.name, ok, tc.want)
		}
		if ok && got != tc.in {
			t.Errorf("carriableRim(%s) returned a different curve than it was given", tc.name)
		}
	}
}
