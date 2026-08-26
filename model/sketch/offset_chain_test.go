// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	gmath "oblikovati.org/math"
)

// TestConnectedChainFromProjectedLoop: Loop Select must trace a loop made of projected reference
// curves (a projected face perimeter), which Paths() does not report — the offset workflow the
// #2158 follow-up enables.
func TestConnectedChainFromProjectedLoop(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	corners := []gmath.Point2{gmath.P2(0, 0), gmath.P2(4, 0), gmath.P2(4, 4), gmath.P2(0, 4)}
	var first Entity
	for i := range 4 {
		pc := s.RestoreProjectedCurve(nextID(), []gmath.Point2{corners[i], corners[(i+1)%4]}, "edge", "E")
		if i == 0 {
			first = pc
		}
	}
	path, ok := s.ConnectedChainFrom(first)
	if !ok {
		t.Fatal("ConnectedChainFrom returned no chain")
	}
	if path.Count() != 4 || !path.IsClosed() {
		t.Errorf("projected square chain = %d entities closed=%v, want 4 closed", path.Count(), path.IsClosed())
	}
}

// TestConnectedChainFromOpenChain: an open chain of native lines traces as an open path in order.
func TestConnectedChainFromOpenChain(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l1 := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(2, 0))
	s.Lines().AddByTwoPoints(gmath.P2(2, 0), gmath.P2(2, 2))
	s.Lines().AddByTwoPoints(gmath.P2(2, 2), gmath.P2(4, 2))
	path, _ := s.ConnectedChainFrom(l1)
	if path.Count() != 3 || path.IsClosed() {
		t.Errorf("open 3-line chain = %d entities closed=%v, want 3 open", path.Count(), path.IsClosed())
	}
}

// TestConnectedChainFromCircle: a full circle is its own one-element closed chain.
func TestConnectedChainFromCircle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	c := s.Circles().AddByCenterRadius(gmath.P2(0, 0), 2)
	path, ok := s.ConnectedChainFrom(c)
	if !ok || path.Count() != 1 || !path.IsClosed() {
		t.Errorf("circle chain ok=%v count=%d closed=%v, want 1 closed", ok, path.Count(), path.IsClosed())
	}
}
