// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// TestTrimmedCurve3RestrictsDomain: a forward restriction re-presents [Lo,Hi] over [0,1],
// mapping the unit endpoints to the base points at Lo and Hi.
func TestTrimmedCurve3RestrictsDomain(t *testing.T) {
	base := NewLineSegment(math.P3(0, 0, 0), math.P3(9, 0, 0))
	tc := TrimmedCurve3{Base: base, Lo: 1.0 / 3, Hi: 2.0 / 3}
	if lo, hi := tc.Domain(); lo != 0 || hi != 1 {
		t.Fatalf("Domain()=(%g,%g), want (0,1)", lo, hi)
	}
	if got := tc.PointAt(0); got != base.PointAt(1.0/3) {
		t.Fatalf("PointAt(0)=%v, want base.PointAt(1/3)=%v", got, base.PointAt(1.0/3))
	}
	if got := tc.PointAt(1); got != base.PointAt(2.0/3) {
		t.Fatalf("PointAt(1)=%v, want base.PointAt(2/3)=%v", got, base.PointAt(2.0/3))
	}
	// Interior parameter maps affinely into the sub-interval.
	if got := tc.PointAt(0.5); got != base.PointAt(0.5) {
		t.Fatalf("PointAt(0.5)=%v, want base.PointAt(0.5)=%v", got, base.PointAt(0.5))
	}
}

// TestTrimmedCurve3ReversedOrientation: Lo>Hi presents the base sub-span REVERSED — PointAt(0)
// is the HIGH base point and the tangent flips sign, so a canal sub-edge sampled in reverse still
// runs pts[i]→pts[i+1] in loop order.
func TestTrimmedCurve3ReversedOrientation(t *testing.T) {
	base := NewLineSegment(math.P3(0, 0, 0), math.P3(9, 0, 0))
	tc := TrimmedCurve3{Base: base, Lo: 2.0 / 3, Hi: 1.0 / 3}
	if got := tc.PointAt(0); got != base.PointAt(2.0/3) {
		t.Fatalf("reversed PointAt(0)=%v, want base.PointAt(2/3)=%v", got, base.PointAt(2.0/3))
	}
	if got := tc.PointAt(1); got != base.PointAt(1.0/3) {
		t.Fatalf("reversed PointAt(1)=%v, want base.PointAt(1/3)=%v", got, base.PointAt(1.0/3))
	}
	if tan := tc.TangentAt(0.5); tan.X >= 0 {
		t.Fatalf("reversed TangentAt(0.5).X=%g, want negative (base runs +X)", tan.X)
	}
}

// TestTrimmedCurve3FollowsCurvedBase: on an arc the restriction's interior lies ON the arc (not on
// the chord) — the property that lets the trimmed sub-span kill the residual patch folds a straight
// chord would leave (n7-tessellation-diagnosis.md §2).
func TestTrimmedCurve3FollowsCurvedBase(t *testing.T) {
	arc, err := Arc3dByThreePoints(math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(-1, 0, 0))
	if err != nil {
		t.Fatalf("Arc3dByThreePoints: %v", err)
	}
	lo, hi := arc.Domain()
	mid := lo + 0.5*(hi-lo)
	tc := TrimmedCurve3{Base: arc, Lo: lo, Hi: mid}
	// The trimmed sub-arc's midpoint lies on the unit circle (radius 1), not on the chord.
	p := tc.PointAt(0.5)
	if r := float64(math.P3(0, 0, 0).DistanceTo(p)); r < 0.999 || r > 1.001 {
		t.Fatalf("trimmed arc midpoint radius %g, want ~1 (on the arc, not the chord)", r)
	}
	if got, want := tc.PointAt(1), arc.PointAt(mid); float64(got.DistanceTo(want)) > 1e-9 {
		t.Fatalf("PointAt(1)=%v, want arc.PointAt(mid)=%v", got, want)
	}
}
