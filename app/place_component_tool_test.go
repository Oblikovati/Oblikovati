// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/scene"
)

// placeReady sets up an assembly session with a top-down camera and a started Place tool armed
// with the widget component — the fixture the placement tests click into. The default camera
// looks down −Z from (0,0,10) at the ground plane (Z=0), so a centre click maps to the origin.
func placeReady(t *testing.T) (*Session, *compdef.AssemblyComponentDefinition, *PlaceComponentTool) {
	t.Helper()
	s, asm := assemblyWithComponent(t)
	trackFromHere(s) // baseline = empty assembly, the state the first placement undoes to
	s.SetCamera(scene.NewCamera(800, 600))
	widget, ok := s.ActiveDocument().OpenReference("widget.obk")
	if !ok {
		t.Fatal("widget.obk should resolve as a reference of the active assembly")
	}
	tool := NewPlaceComponentTool()
	tool.SetComponentDocument(widget)
	s.StartTool(tool)
	return s, asm, tool
}

// translationOf returns the (x,y,z) translation cells of the occurrence's placement transform.
func translationOf(asm *compdef.AssemblyComponentDefinition, i int) (x, y, z float64) {
	cells := asm.Occurrences().Item(i).Transform().Cells()
	return cells[3], cells[7], cells[11]
}

// TestPlaceComponentDropsAtGroundClick is the headline placement test (#763): a click on the
// viewport drops one occurrence at the ground-plane point under the cursor. A centre click on
// the top-down camera resolves to the world origin.
func TestPlaceComponentDropsAtGroundClick(t *testing.T) {
	t.Parallel()
	s, asm, _ := placeReady(t)

	s.Click(400, 300) // viewport centre → ground origin
	if got := asm.Occurrences().Count(); got != 1 {
		t.Fatalf("after click: occurrence count = %d, want 1", got)
	}
	x, y, z := translationOf(asm, 0)
	if stdmath.Abs(x) > 1e-6 || stdmath.Abs(y) > 1e-6 || stdmath.Abs(z) > 1e-6 {
		t.Errorf("placement at (%g,%g,%g), want the ground origin (0,0,0)", x, y, z)
	}
	if name := asm.Occurrences().Item(0).Name(); name != "widget:1" {
		t.Errorf("instance name = %q, want %q", name, "widget:1")
	}
}

// TestPlaceComponentProjectsOffCentreClick checks the placement follows the cursor: a top-centre
// click resolves to a point offset along +Y on the ground plane, not trivially the origin. With
// the default 45° FOV camera at depth 10, the top edge projects to Y = 10·tan(π/8).
func TestPlaceComponentProjectsOffCentreClick(t *testing.T) {
	t.Parallel()
	s, asm, _ := placeReady(t)

	s.Click(400, 0) // top-centre (screen y-down) → ground +Y
	if got := asm.Occurrences().Count(); got != 1 {
		t.Fatalf("after click: occurrence count = %d, want 1", got)
	}
	x, y, z := translationOf(asm, 0)
	wantY := 10 * stdmath.Tan(stdmath.Pi/8)
	if stdmath.Abs(x) > 1e-6 || stdmath.Abs(z) > 1e-6 || stdmath.Abs(y-wantY) > 1e-6 {
		t.Errorf("placement at (%g,%g,%g), want (0,%g,0)", x, y, z, wantY)
	}
}

// TestPlaceComponentMultiDropEachUndoable checks the tool stays open across clicks (multi-place)
// and that each drop is its own undo step: two clicks place two occurrences, and undo removes
// them one at a time back to the empty assembly.
func TestPlaceComponentMultiDropEachUndoable(t *testing.T) {
	t.Parallel()
	s, asm, _ := placeReady(t)

	s.Click(400, 300)
	s.Click(400, 0)
	if got := asm.Occurrences().Count(); got != 2 {
		t.Fatalf("after two clicks: occurrence count = %d, want 2", got)
	}

	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if got := asm.Occurrences().Count(); got != 1 {
		t.Errorf("after one undo: occurrence count = %d, want 1 (each drop is its own step)", got)
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if got := asm.Occurrences().Count(); got != 0 {
		t.Errorf("after two undos: occurrence count = %d, want 0", got)
	}
}

// TestPlaceComponentCanCommitAfterFirstDrop checks OK is disabled until at least one instance is
// placed, then enabled — the tool finishes only once it has done something.
func TestPlaceComponentCanCommitAfterFirstDrop(t *testing.T) {
	t.Parallel()
	s, _, tool := placeReady(t)

	if tool.CanCommit() {
		t.Error("CanCommit should be false before any placement")
	}
	s.Click(400, 300)
	if !tool.CanCommit() {
		t.Error("CanCommit should be true after a placement")
	}
}

// TestPlaceComponentIgnoresClickWithoutComponent checks a click before a component is chosen
// places nothing and explains why, rather than panicking on a nil component.
func TestPlaceComponentIgnoresClickWithoutComponent(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	trackFromHere(s)
	s.SetCamera(scene.NewCamera(800, 600))
	s.StartTool(NewPlaceComponentTool()) // armed with no component

	s.Click(400, 300)
	if got := asm.Occurrences().Count(); got != 0 {
		t.Fatalf("a click without a component placed %d occurrences, want 0", got)
	}
	if s.notice == "" {
		t.Error("a click without a component should report why nothing was placed")
	}
}

// TestPlaceComponentBridgeFeedsFile checks the head's session bridge: the tool reports it is
// awaiting a file, and SetPlaceComponentDocument arms it so a subsequent click can place.
func TestPlaceComponentBridgeFeedsFile(t *testing.T) {
	t.Parallel()
	s, asm := assemblyWithComponent(t)
	trackFromHere(s)
	s.SetCamera(scene.NewCamera(800, 600))
	s.StartTool(NewPlaceComponentTool())

	if !s.PlaceComponentAwaitingFile() {
		t.Fatal("a fresh Place tool should await a component file")
	}
	widget, _ := s.ActiveDocument().OpenReference("widget.obk")
	s.SetPlaceComponentDocument(widget)
	if s.PlaceComponentAwaitingFile() {
		t.Error("the tool should not await a file once SetPlaceComponentDocument is called")
	}

	s.Click(400, 300)
	if got := asm.Occurrences().Count(); got != 1 {
		t.Errorf("after arming and clicking: occurrence count = %d, want 1", got)
	}
}

// TestPlaceCommandStartsTool checks the Assemble-tab Place button starts the Place Component
// tool (replacing the former stub), so the head can then drive its file dialog (#763).
func TestPlaceCommandStartsTool(t *testing.T) {
	t.Parallel()
	s := assemblySession(t)
	if err := s.Execute("Assembly.Place"); err != nil {
		t.Fatalf("Execute Assembly.Place: %v", err)
	}
	if s.ActivePlaceComponent() == nil {
		t.Error("Assembly.Place should start the Place Component tool")
	}
}
