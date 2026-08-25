// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// TestDraggableForModeByGeometryNotDimension is the #2160 (symptom A) gate change. A determined
// entity is now direct-draggable when geometric constraints hold it (the fillet-arc case) but still
// refused when a driving dimension holds it (that stays Relax Mode's job), and a Fix-pinned entity
// never drags. All three land in the same MoveableByDimensionChange/Fixed classes, so the gate
// distinguishes them by whether a driving dimension acts on the entity.
func TestDraggableForModeByGeometryNotDimension(t *testing.T) {
	s, sk := sketchSession(t)

	// A Fix-pinned point click-selects, never drags.
	fixed := sk.Points().Add(math.P2(3, 3))
	sk.GeometricConstraints().AddFix(fixed)
	if s.draggableForMode(fixed) {
		t.Error("a Fix-pinned point must not be direct-draggable")
	}

	// A point determined purely by geometry (coincident to a fixed point, no dimension) drags now —
	// the fillet-arc class the gate used to refuse.
	geom := sk.Points().Add(math.P2(3, 3))
	sk.GeometricConstraints().AddCoincident(geom, fixed)
	if got := sk.MoveableStatus(geom); got != types.MoveableByDimensionChange {
		t.Fatalf("setup: geometrically-determined point should be byDimensionChange, got %v", got)
	}
	if !s.draggableForMode(geom) {
		t.Error("a geometrically-determined entity must be direct-draggable now (#2160)")
	}

	// A point held by a driving dimension stays Relax Mode's job — a normal drag is refused.
	anchor := sk.Points().Add(math.P2(0, 0))
	sk.GeometricConstraints().AddFix(anchor)
	dp := sk.Points().Add(math.P2(2, 0))
	sk.GeometricConstraints().AddHorizontal(anchor, dp)
	if _, err := sk.DimensionConstraints().AddDistance(anchor, dp, "2 cm"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	if s.draggableForMode(dp) {
		t.Error("a dimension-determined entity must stay Relax Mode's job, not a normal drag")
	}
}
