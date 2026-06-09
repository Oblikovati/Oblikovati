// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
)

// newPartWithSquare sets up a session whose active document is a part containing a
// sketch with one closed square profile, and returns the session + that profile
// handle (what the viewport would resolve a click to).
func newPartWithSquare(t *testing.T, side float64) (*Session, ProfileHandle) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)

	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(side, 0))
	c2 := sk.Points().Add(math.P2(side, side))
	c3 := sk.Points().Add(math.P2(0, side))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return s, ProfileHandle{Sketch: sk, ProfileIndex: 0}
}

// TestExtrudeToolEndToEnd is the headline integration test: it operates the UI with
// synthetic input — start the Extrude tool, click the profile, set a distance, hit
// OK — and asserts a real solid was produced in the active part. No GPU involved.
func TestExtrudeToolEndToEnd(t *testing.T) {
	s, profile := newPartWithSquare(t, 2)
	s.SetPicker(stubPicker{sel: profile})

	ext := NewExtrudeTool()
	s.StartTool(ext)               // ribbon: click "Extrude"
	s.Click(120, 90)               // viewport: click the profile
	ext.SetDistance(5)             // mini-toolbar: type the distance
	if err := s.OK(); err != nil { // dialog: OK
		t.Fatalf("OK: %v", err)
	}

	// The active part now holds an extrude feature and a validated prism solid.
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("part has %d bodies after extrude, want 1", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if !body.IsSolid() || len(body.Faces()) != 6 {
		t.Errorf("extruded body: solid=%v faces=%d, want solid/6", body.IsSolid(), len(body.Faces()))
	}
	if z := body.RangeBox().Diagonal().Z; z < 4.99 || z > 5.01 {
		t.Errorf("extrude height = %v, want 5", z)
	}
	if def.Features().Count() != 1 || !def.Features().Item(0).Health().OK() {
		t.Error("extrude feature missing or unhealthy")
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

func TestExtrudeToolNeedsProfileAndDistance(t *testing.T) {
	s, profile := newPartWithSquare(t, 2)
	s.SetPicker(stubPicker{sel: profile})
	ext := NewExtrudeTool()
	s.StartTool(ext)
	// No profile picked yet, no distance → not ready.
	if ext.CanCommit() {
		t.Error("tool ready with no profile/distance")
	}
	s.Click(0, 0)      // pick profile
	ext.SetDistance(3) // set distance
	if !ext.CanCommit() {
		t.Error("tool not ready after profile + distance")
	}
}

// TestExtrudeViaCommandThenTool shows the ribbon command launching the tool.
func TestExtrudeViaCommandAlias(t *testing.T) {
	s, profile := newPartWithSquare(t, 4)
	s.SetPicker(stubPicker{sel: profile})
	var started *ExtrudeTool
	_ = s.Commands().Add(NewCommand("Part.Extrude", "Extrude", "Create", func(sess *Session) error {
		started = NewExtrudeTool()
		sess.StartTool(started)
		return nil
	}).WithAlias("E"))

	if err := s.PressKey(KeyEvent{Key: "E"}); err != nil { // type the "E" alias
		t.Fatalf("alias: %v", err)
	}
	if s.ActiveTool() == nil || s.ActiveTool().Tool() != started {
		t.Fatal("Extrude alias did not start the tool")
	}
	s.Click(1, 1)
	started.SetDistance(2)
	started.SetOperation(ops.NewBody)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Error("alias-launched extrude produced no body")
	}
}
