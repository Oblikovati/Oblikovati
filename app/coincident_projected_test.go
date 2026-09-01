// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestCoincidentToProjectedOriginMovesGeometry reproduces #1268: with the origin centre point
// projected into a sketch, a coincident constraint between a free point and the projected anchor
// must pick the anchor and pull the geometry to it. Before the fix the anchor was absent from the
// pick set (AllPoints), so the constraint could never be formed and nothing moved.
func TestCoincidentToProjectedOriginMovesGeometry(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	sk, err := s.CreateSketch(sketch.XYPlane()) // auto-projects the origin centre at (0,0)
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	free := sk.Points().Add(math.P2(3, 3))

	// The real pick path: nearestEntityPoint gathers AllPoints, which must now include the
	// projected anchor so a click at the origin selects it.
	anchorEnt, ok := s.nearestEntityPoint(math.P2(0, 0), 0.5)
	if !ok {
		t.Fatal("the projected origin anchor is not pickable at (0,0)")
	}
	if _, isPoint := anchorEnt.(*sketch.Point); !isPoint {
		t.Fatalf("picked entity at the origin is %T, want a *sketch.Point anchor", anchorEnt)
	}

	if err := applyCoincident(s, []sketch.Entity{anchorEnt, free}); err != nil {
		t.Fatalf("applyCoincident: %v", err)
	}
	if !free.Position().IsEqualTo(math.P2(0, 0), 1e-9) {
		t.Errorf("free point at %v, want pulled to the projected origin (0,0)", free.Position())
	}
}
