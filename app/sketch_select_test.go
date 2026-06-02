// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

func TestIsSelectedEntity(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 1))
	other := sk.Lines().AddByTwoPoints(math.P2(2, 2), math.P2(3, 3))
	selectEntities(s, l)
	if !s.IsSelectedEntity(l) {
		t.Error("the selected line should report as selected")
	}
	if s.IsSelectedEntity(other) {
		t.Error("an unselected line should not report as selected")
	}
}

func TestSketchClickClearsAndShiftAdds(t *testing.T) {
	s, sk := sketchSession(t)
	sk.Circles().AddByCenterRadius(math.P2(0, 0), 2)               // crosses the centre click
	l := sk.Lines().AddByTwoPoints(math.P2(-2, -2), math.P2(2, 2)) // through origin
	_ = l

	// Centre click: the origin is on the line (distance 0) → selects the line.
	s.Click(100, 100)
	if s.Selection().Count() != 1 {
		t.Fatalf("first click selected %d, want 1", s.Selection().Count())
	}

	// A click on empty space (far corner) with no modifier clears the selection.
	s.Click(199, 1)
	if s.Selection().Count() != 0 {
		t.Errorf("empty click left %d selected, want 0", s.Selection().Count())
	}
}

func TestPickSketchEntityFindsCircleOutline(t *testing.T) {
	s, sk := sketchSession(t)
	// Circle centred at (5,0), radius 5 → its ring passes through the origin, while its
	// centre point is far from the origin click, so the curve (not a point) is picked.
	c := sk.Circles().AddByCenterRadius(math.P2(5, 0), 5)
	ent, ok := s.pickSketchEntity(100, 100) // centre pixel → sketch (0,0)
	if !ok || ent != c {
		t.Errorf("picked %v (ok=%v), want the circle outline", ent, ok)
	}
}
