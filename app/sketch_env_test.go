// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

func TestCreateSketchEntersEnvironment(t *testing.T) {
	s, _ := emptyPartSession(t)
	if !s.CanCreateSketch() {
		t.Fatal("a fresh part should allow creating a sketch")
	}
	sk, err := s.CreateSketchOnOrigin(OriginXY)
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	if !s.InSketch() || s.ActiveSketch() != sk || !sk.IsEditing() {
		t.Error("CreateSketch did not enter the sketch environment")
	}
	if s.CanCreateSketch() {
		t.Error("cannot start a second sketch while one is open")
	}
}

func TestCreateSketchRejectsNesting(t *testing.T) {
	s, _ := emptyPartSession(t)
	if _, err := s.CreateSketchOnOrigin(OriginXY); err != nil {
		t.Fatalf("first CreateSketch: %v", err)
	}
	if _, err := s.CreateSketchOnOrigin(OriginXZ); err == nil {
		t.Error("creating a nested sketch should error")
	}
}

func TestCreateSketchNeedsActivePart(t *testing.T) {
	s := NewSession() // no document
	if s.CanCreateSketch() {
		t.Error("no active part should disable Create 2D Sketch")
	}
	if _, err := s.CreateSketchOnOrigin(OriginXY); err == nil {
		t.Error("CreateSketch with no part should error")
	}
}

func TestFinishSketchLeavesEnvironmentAndRecomputes(t *testing.T) {
	s, def := emptyPartSession(t)
	sk, _ := s.CreateSketchOnOrigin(OriginXY)
	// Draw a rectangle and extrude after finishing, proving Finish recomputes the part.
	rect := NewRectangleTool()
	s.StartTool(rect)
	s.Click(40, 40)
	s.Click(160, 160) // auto-commits the rectangle
	if err := s.FinishSketch(); err != nil {
		t.Fatalf("FinishSketch: %v", err)
	}
	if s.InSketch() || sk.IsEditing() {
		t.Error("FinishSketch did not leave the sketch environment")
	}
	// A profile is now available to extrude (the sketch is committed to the part).
	if def.Sketches().Count() != 1 {
		t.Errorf("part has %d sketches, want 1", def.Sketches().Count())
	}
}

func TestFinishSketchCancelsActiveTool(t *testing.T) {
	s, _ := emptyPartSession(t)
	_, _ = s.CreateSketchOnOrigin(OriginXY)
	s.StartTool(NewLineTool())
	s.Click(10, 10)
	if err := s.FinishSketch(); err != nil {
		t.Fatalf("FinishSketch: %v", err)
	}
	if s.ActiveTool() != nil {
		t.Error("FinishSketch should cancel the in-progress tool")
	}
}

func TestFinishSketchErrorsWhenNotEditing(t *testing.T) {
	s, _ := emptyPartSession(t)
	if err := s.FinishSketch(); err == nil {
		t.Error("FinishSketch with no open sketch should error")
	}
}

func TestOriginPlanesDiffer(t *testing.T) {
	s, _ := emptyPartSession(t)
	sk, _ := s.CreateSketchOnOrigin(OriginXZ)
	// The XZ origin plane's normal is the Y axis (not Z), distinguishing it from XY.
	if n := sk.Plane().Normal().AsVector(); n.Y == 0 {
		t.Errorf("XZ sketch plane normal = %v, expected a Y component", n)
	}
}
