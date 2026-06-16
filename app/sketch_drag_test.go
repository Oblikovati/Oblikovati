// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/scene"
)

// centerOnSketchPoint aims a known 200×200 camera straight down −Z at sketch point (x,0) on the
// XY plane, so that sketch point projects to screen centre (100,100) and a +x screen drag is a
// +x sketch move — letting the drag tests drive real screen→sketch picking deterministically.
func centerOnSketchPoint(s *Session, x float64) {
	cam := scene.NewCamera(200, 200)
	cam.Eye = math.P3(x, 0, 10)
	cam.Target = math.P3(x, 0, 0)
	cam.Up = math.V3(0, 1, 0)
	s.SetCamera(cam)
}

func TestEntityDragMovesFreePoint(t *testing.T) {
	s, sk := sketchSession(t)
	s.Grid().SnapToGrid = false
	p := sk.Points().Add(math.P2(1, 0)) // an unconstrained (free) point
	centerOnSketchPoint(s, 1)           // (1,0) is at screen centre (100,100)

	if !s.BeginEntityDrag(100, 100) {
		t.Fatal("BeginEntityDrag did not start a drag on the free point at screen centre")
	}
	if !s.EntityDragActive() {
		t.Fatal("EntityDragActive should be true after BeginEntityDrag")
	}
	s.UpdateEntityDrag(140, 100) // drag 40px to the right → +x in the sketch
	s.CommitEntityDrag()

	if s.EntityDragActive() {
		t.Error("CommitEntityDrag should end the drag")
	}
	if p.Position().X <= 1.0 {
		t.Errorf("dragged point did not move in +x: now at %v", p.Position())
	}
	if y := p.Position().Y; y > 1e-6 || y < -1e-6 {
		t.Errorf("a horizontal drag should not move the point in y: %v", p.Position())
	}
}

func TestEntityDragResolvesConstraint(t *testing.T) {
	s, sk := sketchSession(t)
	s.Grid().SnapToGrid = false
	a := sk.Points().Add(math.P2(1, 0))
	b := sk.Points().Add(math.P2(1, 0)) // coincident with a
	sk.GeometricConstraints().AddCoincident(a, b)
	centerOnSketchPoint(s, 1) // (1,0) is at screen centre

	if !s.BeginEntityDrag(100, 100) {
		t.Fatal("did not begin a drag on the constrained point")
	}
	s.UpdateEntityDrag(150, 100) // drag right
	s.CommitEntityDrag()

	if a.Position().X <= 1.0 {
		t.Errorf("dragged point did not move in +x: %v", a.Position())
	}
	if !b.Position().IsEqualTo(a.Position(), 1e-6) {
		t.Errorf("the coincident point did not follow the drag: a=%v b=%v", a.Position(), b.Position())
	}
}

func TestEntityDragGuardsAndCancel(t *testing.T) {
	s, sk := sketchSession(t)
	s.Grid().SnapToGrid = false

	// A fixed (grounded) point is not MoveableFree → BeginEntityDrag declines it.
	fixed := sk.Points().Add(math.P2(1, 0))
	sk.GeometricConstraints().AddFix(fixed)
	centerOnSketchPoint(s, 1)
	if s.BeginEntityDrag(100, 100) {
		t.Error("a fixed point must not begin a drag")
	}

	// UpdateEntityDrag with no active drag is a no-op (must not panic).
	s.UpdateEntityDrag(120, 100)

	// Begin on a free point, then cancel → the drag is dropped.
	sk.Points().Add(math.P2(5, 0))
	centerOnSketchPoint(s, 5)
	if !s.BeginEntityDrag(100, 100) {
		t.Fatal("should begin a drag on the free point")
	}
	s.CancelEntityDrag()
	if s.EntityDragActive() {
		t.Error("CancelEntityDrag should clear the drag")
	}
}

// TestDragAnchorsDedupesSharedPoints checks that dragging a selection of two lines that share an
// endpoint pins that shared point once (the de-dup branch of dragAnchors).
func TestDragAnchorsDedupesSharedPoints(t *testing.T) {
	s, sk := sketchSession(t)
	shared := sk.Points().Add(math.P2(0, 0))
	l1 := sk.Lines().Add(shared, sk.Points().Add(math.P2(2, 0)))
	l2 := sk.Lines().Add(shared, sk.Points().Add(math.P2(0, 2)))
	s.Selection().Add(SketchEntityHandle{Entity: l1})
	s.Selection().Add(SketchEntityHandle{Entity: l2})

	anchors := s.dragAnchors(l1) // l1 is selected → drag the whole selection
	// l1 (shared, p2) + l2 (shared, p3) → shared counted once → 3 distinct points.
	if len(anchors) != 3 {
		t.Errorf("dragAnchors over two lines sharing a point = %d anchors, want 3 (de-duplicated)", len(anchors))
	}
}

func TestBeginEntityDragRejectsWhenNotDraggable(t *testing.T) {
	// No active sketch → no drag.
	plain := NewSession()
	if plain.BeginEntityDrag(10, 10) {
		t.Error("BeginEntityDrag must be a no-op outside a sketch")
	}

	// An active tool owns clicks → no drag.
	s, sk := sketchSession(t)
	sk.Points().Add(math.P2(1, 0))
	centerOnSketchPoint(s, 1)
	s.StartTool(stubSketchTool{})
	if s.BeginEntityDrag(100, 100) {
		t.Error("BeginEntityDrag must defer to an active tool")
	}
}

func TestDragSetUsesSelectionOrClickedEntity(t *testing.T) {
	s, sk := sketchSession(t)
	a := sk.Points().Add(math.P2(1, 0))
	b := sk.Points().Add(math.P2(2, 0))

	// Clicking an unselected entity drags just it (and selects it).
	got := s.dragSet(a)
	if len(got) != 1 || got[0] != a || !s.Selection().Contains(SketchEntityHandle{Entity: a}) {
		t.Fatalf("dragSet(unselected) = %v, want [a] and a selected", got)
	}

	// With a multi-selection that includes the clicked entity, the whole selection drags.
	s.Selection().Add(SketchEntityHandle{Entity: b})
	got = s.dragSet(a)
	if len(got) != 2 {
		t.Fatalf("dragSet(selected, multi) = %d entities, want 2 (the selection)", len(got))
	}
}

// stubSketchTool is a minimal Tool so BeginEntityDrag sees an active tool.
type stubSketchTool struct{}

func (stubSketchTool) Name() string              { return "stub" }
func (stubSketchTool) Start(*Session)            {}
func (stubSketchTool) Pick(*Session, Selectable) {}
func (stubSketchTool) CanCommit() bool           { return false }
func (stubSketchTool) Commit(*Session) error     { return nil }
func (stubSketchTool) Cancel(*Session)           {}
