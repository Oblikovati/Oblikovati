// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"github.com/Oblikovati/oblikovati/math"
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
