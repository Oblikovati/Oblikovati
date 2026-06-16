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
