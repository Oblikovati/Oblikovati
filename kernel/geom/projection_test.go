// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

func TestProjectPointToSurfaceOnSurface(t *testing.T) {
	s := cProfileSurface(t)
	for _, uv := range [][2]float64{{0.5, 0.5}, {0.25, 0.7}, {0.8, 0.15}} {
		want := s.PointAt(uv[0], uv[1])
		u, v, dist := ProjectPointToSurface(s, want)
		if dist > 1e-4 {
			t.Errorf("on-surface point at %v: residual %.6f, want ~0", uv, dist)
		}
		if got := s.PointAt(u, v); float64(got.DistanceTo(want)) > 1e-4 {
			t.Errorf("on-surface point at %v: PointAt(%g,%g)=%v want %v", uv, u, v, got, want)
		}
	}
}

func TestProjectPointToSurfaceResidualIsOffSurfaceGap(t *testing.T) {
	s := cProfileSurface(t)
	// Offset a surface point along the normal by a known gap (the M25 edge-off-surface case).
	u0, v0 := 0.5, 0.5
	base := s.PointAt(u0, v0)
	n := s.NormalAt(u0, v0)
	const gap = 0.1
	off := base.TranslateBy(n.Scale(gap))
	_, _, dist := ProjectPointToSurface(s, off)
	if stdmath.Abs(dist-gap) > 1e-3 {
		t.Errorf("residual for a point %.3f off the surface = %.4f, want ~%.3f", gap, dist, gap)
	}
}

func TestProjectCurveToSurfaceIsContinuousPcurve(t *testing.T) {
	s := cProfileSurface(t)
	// A polyline of on-surface points along v=0.5, increasing u: its pcurve must round-trip and stay
	// continuous + monotone in u (no branch jump).
	var pts []math.Point3
	const n = 12
	for i := 0; i <= n; i++ {
		u := 0.1 + 0.8*float64(i)/n
		pts = append(pts, s.PointAt(u, 0.5))
	}
	pc := ProjectCurveToSurface(s, pts)
	if len(pc) != len(pts) {
		t.Fatalf("pcurve length %d, want %d", len(pc), len(pts))
	}
	for i := range pts {
		if got := s.PointAt(float64(pc[i].X), float64(pc[i].Y)); float64(got.DistanceTo(pts[i])) > 1e-3 {
			t.Errorf("pcurve[%d] does not round-trip: PointAt=%v want %v", i, got, pts[i])
		}
		if i > 0 && float64(pc[i].X) <= float64(pc[i-1].X) {
			t.Errorf("pcurve u not monotone at %d: %g then %g (branch jump)", i, pc[i-1].X, pc[i].X)
		}
	}
}

func TestProjectCurveToSurfaceEmpty(t *testing.T) {
	if got := ProjectCurveToSurface(cProfileSurface(t), nil); len(got) != 0 {
		t.Errorf("empty input gave %d points", len(got))
	}
}
