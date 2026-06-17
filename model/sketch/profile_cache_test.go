// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// TestProfilesCacheReuseAndInvalidate checks Profiles() returns the same cached
// result while the geometry is unchanged and rebuilds after an edit — the property
// that stops the hover picker from rerunning region detection every frame on a
// dense sketch.
func TestProfilesCacheReuseAndInvalidate(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	s.Lines().AddByTwoPoints(math.P2(1, 0), math.P2(1, 1))

	p1 := s.Profiles()
	if p2 := s.Profiles(); p2 != p1 {
		t.Fatal("Profiles() recomputed despite no geometry change")
	}

	// Adding a line must invalidate the cache.
	s.Lines().AddByTwoPoints(math.P2(1, 1), math.P2(0, 1))
	if p3 := s.Profiles(); p3 == p1 {
		t.Fatal("Profiles() not rebuilt after adding a line")
	}

	// Moving a point (in place, no count change) must also invalidate it.
	stable := s.Profiles()
	s.pts[0].X += 2.5
	if again := s.Profiles(); again == stable {
		t.Fatal("Profiles() not rebuilt after moving a point")
	}
}

// TestProfilesCacheInvalidatesOnConstructionToggle guards the regression where
// toggling an entity's construction flag (which removes it from the profile
// geometry) left no coordinate changed, so the cached profiles went stale.
func TestProfilesCacheInvalidatesOnConstructionToggle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	pts := make([]*Point, 4)
	for i, c := range []math.Point2{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}} {
		pts[i] = s.NewPoint(c)
	}
	var sides []*Line
	for i := range pts {
		sides = append(sides, s.Lines().Add(pts[i], pts[(i+1)%4]))
	}
	before := s.Profiles()
	sides[0].SetConstruction(true) // changes the profile geometry without moving any point
	if s.Profiles() == before {
		t.Fatal("Profiles() cache not invalidated after a construction toggle")
	}
}

// TestProfilesCacheInvalidatesOnRadiusChange guards the regression where resizing a circle via
// its radius (a stored DOF, not a point) — e.g. a "radius = od/2" dimension driven by a
// parameter — left every point unmoved, so the cached profiles went stale and a parametric
// resize produced the OLD geometry (the spacer/flange resize failures).
func TestProfilesCacheInvalidatesOnRadiusChange(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.Circles().AddByCenterRadius(math.P2(0, 0), 0.3)
	before := s.Profiles()
	s.circles.items[0].Radius = 0.4 // resize in place — the centre point does not move
	if s.Profiles() == before {
		t.Fatal("Profiles() cache not invalidated after a circle radius change")
	}
}

// TestProfilesCappedForHugeSketch checks that a sketch past the entity cap (an
// imported drawing) offers no profiles and returns immediately, so the hover
// picker never arranges hundreds of thousands of segments.
func TestProfilesCappedForHugeSketch(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	for i := 0; i < maxProfileEntities+10; i++ {
		x := float64(i)
		s.Lines().AddByTwoPoints(math.P2(x, 0), math.P2(x+0.5, 1))
	}
	if n := s.Profiles().Count(); n != 0 {
		t.Fatalf("huge sketch offered %d profiles, want 0 (capped)", n)
	}
}
