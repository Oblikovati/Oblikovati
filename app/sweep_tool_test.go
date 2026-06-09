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

// newPartWithProfileAndPath sets up a part with a 2×2 square on XY (the profile) and a
// straight line up Z on the XZ plane (the path), returning the section and path handles.
func newPartWithProfileAndPath(t *testing.T) (*Session, ProfileHandle, PathHandle) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	prof := centeredSquareSketch(def, sketch.XYPlane(), 1) // 2×2 square, normal +Z
	pathSketch := def.Sketches().Add(sketch.XZPlane())     // (u,v) → (u,0,v)
	a := pathSketch.Points().Add(math.P2(0, 0))            // model (0,0,0)
	b := pathSketch.Points().Add(math.P2(0, 5))            // model (0,0,5)
	pathSketch.Lines().Add(a, b)
	return s, ProfileHandle{Sketch: prof, ProfileIndex: 0}, PathHandle{Sketch: pathSketch, PathIndex: 0}
}

// TestSweepToolEndToEnd drives the Sweep UI: start the tool, click the profile and the
// path, OK — and asserts a valid swept solid (2×2 profile along a length-5 path ⇒ V=20).
func TestSweepToolEndToEnd(t *testing.T) {
	s, profile, path := newPartWithProfileAndPath(t)
	s.SetPicker(&seqPicker{sels: []Selectable{profile, path}})

	sw := NewSweepTool()
	s.StartTool(sw)  // ribbon: click "Sweep"
	s.Click(10, 10)  // viewport: click the profile
	s.Click(10, 200) // viewport: click the path
	if !sw.CanCommit() {
		t.Fatalf("sweep not ready: profile=%v path=%v", sw.profile != nil, sw.path != nil)
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("part has %d bodies after sweep, want 1", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("swept body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, 20) > 0.02 {
		t.Errorf("swept volume = %g, want ≈20 (area 4 × length 5)", got)
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

func TestSweepViaRibbonCommand(t *testing.T) {
	s, profile, path := newPartWithProfileAndPath(t)
	s.SetPicker(&seqPicker{sels: []Selectable{profile, path}})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Create.Sweep"); err != nil {
		t.Fatalf("execute Create.Sweep: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*SweepTool); !ok {
		t.Fatal("Sweep command did not start the sweep tool")
	}
	s.Click(0, 0)
	s.Click(0, 0)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Error("ribbon-launched sweep produced no body")
	}
}

func TestSweepToolNeedsProfileAndPath(t *testing.T) {
	s, profile, path := newPartWithProfileAndPath(t)
	s.SetPicker(&seqPicker{sels: []Selectable{profile, path}})
	sw := NewSweepTool()
	s.StartTool(sw)
	s.Click(0, 0) // profile only
	if sw.CanCommit() {
		t.Error("sweep ready with no path")
	}
	s.Click(0, 0) // path
	if !sw.CanCommit() {
		t.Error("sweep not ready after profile + path")
	}
}
