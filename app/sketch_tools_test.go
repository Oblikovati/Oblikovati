// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
	"oblikovati.org/scene"
)

// emptyPartSession returns a session whose active document is an empty part, with a
// camera framing the XY plane straight on so sketch-plane clicks map predictably.
func emptyPartSession(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	// Camera on +Z looking down at the XY plane: a pixel maps to (worldX, worldY).
	cam := scene.NewCamera(200, 200)
	cam.Eye = math.P3(0, 0, 10)
	cam.Target = math.P3(0, 0, 0)
	cam.Up = math.V3(0, 1, 0)
	s.SetCamera(cam)
	return s, def
}

func TestRectangleToolDrawsClosedProfileFromClicks(t *testing.T) {
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	if !sk.IsEditing() || s.ActiveSketch() != sk {
		t.Fatal("EnterSketch did not open the sketch")
	}

	rect := NewRectangleTool()
	s.StartTool(rect)
	s.Click(60, 60)   // one corner
	s.Click(140, 140) // opposite corner — auto-commits
	if !rect.CanCommit() {
		t.Fatal("rectangle not ready after two corner clicks")
	}
	// Four lines, and a single closed profile is detectable.
	if sk.Lines().Count() != 4 {
		t.Errorf("rectangle added %d lines, want 4", sk.Lines().Count())
	}
	if profiles := sk.Profiles(); profiles.Count() != 1 || !profiles.Item(0).IsClosed() {
		t.Errorf("clicked rectangle did not form one closed profile (count=%d)", profiles.Count())
	}
}

// TestFullModelingFlowViaUI is the broadest end-to-end test: a brand-new part is
// modeled entirely through synthetic UI input — create a sketch, draw a rectangle by
// clicking, finish the sketch, then extrude the profile — and the result is a real
// solid. No GPU, no hand-built geometry.
func TestFullModelingFlowViaUI(t *testing.T) {
	s, def := emptyPartSession(t)

	// 1. Create + enter a sketch on the XY plane.
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)

	// 2. Draw a rectangle by clicking two corners.
	rect := NewRectangleTool()
	s.StartTool(rect)
	s.Click(40, 40)
	s.Click(160, 160) // auto-commits the rectangle

	// 3. Finish the sketch.
	s.ExitSketch()
	if s.ActiveSketch() != nil || sk.IsEditing() {
		t.Fatal("ExitSketch did not close the sketch")
	}

	// 4. Extrude the resulting profile (picked via a profile stub = clicking it).
	s.SetPicker(stubPicker{sel: ProfileHandle{Sketch: sk, ProfileIndex: 0}})
	ext := NewExtrudeTool()
	s.StartTool(ext)
	s.Click(100, 100)
	ext.SetDistance(8)
	if err := s.OK(); err != nil {
		t.Fatalf("extrude OK: %v", err)
	}

	// Result: one watertight solid, extruded 8 high.
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("flow produced %d bodies, want 1", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if !body.IsSolid() || len(body.Faces()) != 6 {
		t.Errorf("flow body: solid=%v faces=%d, want solid/6", body.IsSolid(), len(body.Faces()))
	}
	if z := body.RangeBox().Diagonal().Z; z < 7.99 || z > 8.01 {
		t.Errorf("flow extrude height = %v, want 8", z)
	}
}

func TestLineToolAndCancel(t *testing.T) {
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)

	line := NewLineTool()
	s.StartTool(line)
	s.Click(50, 50)
	if line.CanCommit() {
		t.Error("line ready after a single click")
	}
	s.Click(150, 70)
	_ = s.PressKey(KeyEvent{Key: "Escape"}) // Escape ends the chain, keeping the segment (#2024)
	if sk.Lines().Count() != 1 {
		t.Errorf("line tool added %d lines, want 1", sk.Lines().Count())
	}

	// Escape cancels an in-progress line tool with no geometry added.
	l2 := NewLineTool()
	s.StartTool(l2)
	s.Click(0, 0)
	_ = s.PressKey(KeyEvent{Key: "Escape"})
	if s.ActiveTool() != nil || sk.Lines().Count() != 1 {
		t.Error("Escape did not cancel the line tool cleanly")
	}
}

func TestSketchClickRequiresSketchTool(t *testing.T) {
	s, _ := emptyPartSession(t)
	// With no active tool, a sketch click is not consumed as a sketch-plane click.
	if s.sketchClick(10, 10) {
		t.Error("sketchClick consumed a click with no active tool")
	}
}
