// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/event"
	"oblikovati.org/math"
)

// triadEventLog subscribes to drag events.
func triadEventLog(s *Session) *[]TriadDragged {
	var got []TriadDragged
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e TriadDragged) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})
	return &got
}

func TestTriadAxisDragTranslatesAlongX(t *testing.T) {
	s := NewSession()
	if err := s.ShowTriad(wire.TriadSpec{Position: types.NewPoint(0, 0, 0), Visible: true}); err != nil {
		t.Fatalf("ShowTriad: %v", err)
	}
	got := triadEventLog(s)

	// Looking straight down -Z from z=10: a ray through (2,0) grabs the X axis at x≈2.
	down := math.V3(0, 0, -1)
	if err := s.BeginTriadDrag(types.TriadXAxis, math.P3(2, 0, 10), down, 0, false, false); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := s.DragTriad(math.P3(5, 0, 10), down, false, false); err != nil {
		t.Fatalf("Drag: %v", err)
	}
	if err := s.EndTriadDrag(math.P3(5, 0, 10), down, false, false); err != nil {
		t.Fatalf("End: %v", err)
	}

	if len(*got) != 3 || (*got)[0].Phase != DragStart || (*got)[2].Phase != DragEnd {
		t.Fatalf("phases = %+v, want start/move/end", *got)
	}
	move := (*got)[1]
	if move.MoveType != types.TriadTranslate {
		t.Errorf("moveType = %v, want translate", move.MoveType)
	}
	if dx := move.Delta[3]; stdmath.Abs(dx-3) > 1e-6 {
		t.Errorf("delta.tx = %v, want 3 (grabbed at 2, dragged to 5)", dx)
	}
	if dy, dz := move.Delta[7], move.Delta[11]; dy != 0 || dz != 0 {
		t.Errorf("off-axis translation (%v, %v), want the X constraint to hold", dy, dz)
	}
	if pos := s.TriadSpec().Position; stdmath.Abs(pos.X-3) > 1e-6 {
		t.Errorf("triad landed at x=%v, want 3 (the delta baked in)", pos.X)
	}
}

func TestTriadRingDragRotatesAboutZ(t *testing.T) {
	s := NewSession()
	if err := s.ShowTriad(wire.TriadSpec{Position: types.NewPoint(0, 0, 0), Visible: true}); err != nil {
		t.Fatalf("ShowTriad: %v", err)
	}
	got := triadEventLog(s)

	down := math.V3(0, 0, -1)
	// Grab the Z ring at (1,0), drag to (0,1): a +90° rotation about Z.
	if err := s.BeginTriadDrag(types.TriadZRing, math.P3(1, 0, 10), down, 0, false, false); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := s.DragTriad(math.P3(0, 1, 10), down, false, false); err != nil {
		t.Fatalf("Drag: %v", err)
	}
	move := (*got)[1]
	if move.MoveType != types.TriadRotate {
		t.Fatalf("moveType = %v, want rotate", move.MoveType)
	}
	// The delta must map (1,0,0) → (0,1,0).
	x := [3]float64{move.Delta[0], move.Delta[4], move.Delta[8]}
	if stdmath.Abs(x[0]) > 1e-6 || stdmath.Abs(x[1]-1) > 1e-6 {
		t.Errorf("R·ex = %v, want (0,1,0) for a +90° Z rotation", x)
	}
}

func TestTriadPlanarDragStaysInPlane(t *testing.T) {
	s := NewSession()
	if err := s.ShowTriad(wire.TriadSpec{Position: types.NewPoint(0, 0, 0), Visible: true}); err != nil {
		t.Fatalf("ShowTriad: %v", err)
	}
	got := triadEventLog(s)
	down := math.V3(0, 0, -1)
	if err := s.BeginTriadDrag(types.TriadXYPlane, math.P3(1, 1, 10), down, 0, false, false); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := s.DragTriad(math.P3(4, -2, 10), down, false, false); err != nil {
		t.Fatalf("Drag: %v", err)
	}
	move := (*got)[1]
	if move.Delta[3] != 3 || move.Delta[7] != -3 || move.Delta[11] != 0 {
		t.Errorf("planar delta = (%v,%v,%v), want (3,-3,0)", move.Delta[3], move.Delta[7], move.Delta[11])
	}
}

func TestTriadRespectsAllowedMaskAndVisibility(t *testing.T) {
	s := NewSession()
	if err := s.ShowTriad(wire.TriadSpec{
		Visible: true, Allowed: []types.TriadSegment{types.TriadXAxis},
	}); err != nil {
		t.Fatalf("ShowTriad: %v", err)
	}
	down := math.V3(0, 0, -1)
	if err := s.BeginTriadDrag(types.TriadYAxis, math.P3(0, 2, 10), down, 0, false, false); err == nil {
		t.Error("a segment outside the allowed mask must refuse to drag")
	}
	s.HideTriad()
	if err := s.BeginTriadDrag(types.TriadXAxis, math.P3(2, 0, 10), down, 0, false, false); err == nil {
		t.Error("a hidden triad must refuse to drag")
	}
	if err := s.ShowTriad(wire.TriadSpec{Allowed: []types.TriadSegment{99}}); err == nil {
		t.Error("an out-of-range allowed segment must be rejected")
	}
}

func TestTriadSegmentHoverEmitsOnTransition(t *testing.T) {
	s := NewSession()
	var got []TriadSegmentChanged
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e TriadSegmentChanged) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})
	s.HoverTriadSegment(types.TriadXAxis, true)
	s.HoverTriadSegment(types.TriadXAxis, true) // repeat: silent
	s.HoverTriadSegment(types.TriadZRing, true) // segment change
	s.HoverTriadSegment(types.TriadZRing, false)
	if len(got) != 3 || !got[0].Hovered || got[1].Segment != types.TriadZRing || got[2].Hovered {
		t.Fatalf("events = %+v, want hover X, hover ZRing, unhover", got)
	}
}

func TestManipulatorDragSlidesInViewPlane(t *testing.T) {
	s := NewSession()
	if err := s.SetManipulators("sim", []wire.ManipulatorHandleSpec{
		{ID: "tip", Position: types.NewPoint(1, 1, 0)},
	}, ""); err != nil {
		t.Fatalf("SetManipulators: %v", err)
	}
	var got []ManipulatorDragged
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e ManipulatorDragged) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})

	down := math.V3(0, 0, -1)
	viewNormal := math.V3(0, 0, 1) // camera looking down -Z ⇒ plane normal +Z
	if err := s.BeginManipulatorDrag("sim", "tip", viewNormal, math.P3(1, 1, 10), down, 0, false, false); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := s.DragManipulator(math.P3(4, 2, 10), down, false, false); err != nil {
		t.Fatalf("Drag: %v", err)
	}
	if err := s.EndManipulatorDrag(math.P3(4, 2, 10), down, false, false); err != nil {
		t.Fatalf("End: %v", err)
	}
	if len(got) != 3 || got[1].Position != types.NewPoint(4, 2, 0) {
		t.Fatalf("events = %+v, want the handle at (4,2,0)", got)
	}
	// The final position bakes into the spec.
	if h := s.Manipulators().Handles()["sim"][0]; h.Position != types.NewPoint(4, 2, 0) {
		t.Errorf("baked position = %v, want (4,2,0)", h.Position)
	}
}

func TestCommandBoundGizmosDieWithTool(t *testing.T) {
	s := NewSession()
	s.StartTool(FakeMiniToolbarTool{})
	if err := s.ShowTriad(wire.TriadSpec{Visible: true, Command: "Sim.Move"}); err != nil {
		t.Fatalf("ShowTriad: %v", err)
	}
	if err := s.SetManipulators("sim", []wire.ManipulatorHandleSpec{{ID: "tip"}}, "Sim.Move"); err != nil {
		t.Fatalf("SetManipulators: %v", err)
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if s.TriadSpec().Visible {
		t.Error("a command-bound triad must hide when the tool ends")
	}
	if len(s.Manipulators().Handles()) != 0 {
		t.Error("command-bound manipulators must die with the tool")
	}
}
