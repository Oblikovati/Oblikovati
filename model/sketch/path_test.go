// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati/math"
)

func TestConnectedChainYieldsSweepPath(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(2, 0))
	c := s.Points().Add(math.P2(2, 2))
	s.Lines().Add(a, b)
	s.Lines().Add(b, c)
	paths := s.Paths()
	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
	p := paths[0]
	if p.Count() != 2 || p.IsClosed() {
		t.Errorf("path count=%d closed=%v, want 2 / open", p.Count(), p.IsClosed())
	}
	if len(p.Entities()) != 2 {
		t.Error("Entities() length mismatch")
	}
}

// TestArcPathIsSampledNotChorded is the regression for the curved-sweep-rail bug: an arc
// path used to contribute only its two endpoints, so a sweep along it collapsed to the
// chord (an 11% under-volume on a 90° bend). Path.Points() now samples the arc into a
// polyline that traces the curve.
func TestArcPathIsSampledNotChorded(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	// 90° arc: centre (2,0), start (0,0), end (2,2) → radius 2, clockwise.
	s.Arcs().AddByCenterStartEnd(math.P2(2, 0), math.P2(0, 0), math.P2(2, 2), false)
	paths := s.Paths()
	if len(paths) != 1 {
		t.Fatalf("expected 1 arc path, got %d", len(paths))
	}
	pts := paths[0].Points()
	if len(pts) < 6 {
		t.Fatalf("arc path sampled to %d points, want many (was collapsing to 2 = the chord)", len(pts))
	}
	// The sampled polyline length must approach the arc length (π·R/2 ≈ 3.1416), not the
	// chord (√8 ≈ 2.828).
	var length float64
	for i := 1; i < len(pts); i++ {
		length += float64(pts[i-1].DistanceTo(pts[i]))
	}
	const arcLen = 3.14159
	if d := length - arcLen; d < -0.05 || d > 0.001 {
		t.Errorf("sampled arc-path length = %.5f, want ≈ %.5f (chord √8=2.828 would be the bug)", length, arcLen)
	}
}

func TestClosedLoopIsAlsoAPath(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	addRectangle(s, 0, 0, 1, 1)
	paths := s.Paths()
	if len(paths) != 1 || !paths[0].IsClosed() {
		t.Fatalf("rectangle path: count=%d closed=%v", len(paths), paths[0].IsClosed())
	}
}

func TestPath3D(t *testing.T) {
	pts := []*Point3D{
		NewPoint3D(math.P3(0, 0, 0)),
		NewPoint3D(math.P3(1, 0, 0)),
		NewPoint3D(math.P3(1, 1, 1)),
	}
	p := NewPath3D(pts, false)
	if p.Count() != 3 || p.IsClosed() {
		t.Fatalf("Path3D count=%d closed=%v", p.Count(), p.IsClosed())
	}
	got := p.Points()
	if len(got) != 3 || !got[2].IsEqualTo(math.P3(1, 1, 1), 1e-9) {
		t.Errorf("Path3D points wrong: %v", got)
	}
	if !NewPath3D(pts, true).IsClosed() {
		t.Error("closed Path3D not reported closed")
	}
}
