// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// stubPlanePicker is a Picker that always returns the given work-plane handle, so a
// test can drive the Create Sketch tool's plane pick without a camera/geometry.
type stubPlanePicker struct{ handle WorkPlaneHandle }

func (p stubPlanePicker) Pick(_, _ float64, filter *SelectionFilter) (Selectable, bool) {
	if !filter.Accepts(SelectWorkPlane) {
		return nil, false
	}
	return p.handle, true
}

func TestCreateSketchToolFlow(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register: %v", err)
	}
	plane := def.OriginPlanes()[1] // XZ Plane
	s.SetPicker(stubPlanePicker{handle: WorkPlaneHandle{Plane: plane}})

	if err := s.Execute("Sketch.Create2D"); err != nil {
		t.Fatalf("Create 2D Sketch: %v", err)
	}
	// The tool is now active and waiting for a plane pick (no sketch yet).
	if s.ActiveTool() == nil || s.InSketch() {
		t.Fatal("Create 2D Sketch should start a tool and wait for a plane pick")
	}
	// Click in the viewport: the picker returns the plane, the tool auto-commits.
	s.Click(100, 100)
	if s.ActiveTool() != nil {
		t.Error("tool should end once the plane is picked")
	}
	if !s.InSketch() {
		t.Fatal("clicking the plane did not enter the sketch environment")
	}
	// The new sketch is hosted on the XZ plane (a Y-component normal).
	if n := s.ActiveSketch().Plane().Normal().AsVector(); n.Y == 0 {
		t.Errorf("sketch plane normal = %v, expected the XZ plane", n)
	}
}

func TestCreateSketchToolPicksViaBrowser(t *testing.T) {
	s, _ := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := s.Execute("Sketch.Create2D"); err != nil {
		t.Fatalf("Create 2D Sketch: %v", err)
	}
	// Selecting a plane node in the browser while the tool is active feeds it as a pick.
	origin := originFolder(BuildBrowser(s))
	s.SelectBrowserNode(origin.Children[2]) // YZ Plane
	if s.ActiveTool() != nil || !s.InSketch() {
		t.Error("browser plane pick should auto-enter the sketch")
	}
}

func TestCreate2DSketchUsesPreselectedPlane(t *testing.T) {
	s, _ := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register: %v", err)
	}
	// Pre-select a plane, then run the command: it sketches immediately (no tool).
	origin := originFolder(BuildBrowser(s))
	s.SelectBrowserNode(origin.Children[1]) // XZ Plane
	if err := s.Execute("Sketch.Create2D"); err != nil {
		t.Fatalf("Create 2D Sketch: %v", err)
	}
	if s.ActiveTool() != nil {
		t.Error("a pre-selected plane should sketch immediately, not start a tool")
	}
	if !s.InSketch() {
		t.Error("pre-selected Create 2D Sketch did not enter the environment")
	}
}

func TestCreateSketchToolMetadataAndEmptyCommit(t *testing.T) {
	s, _ := emptyPartSession(t)
	tool := NewCreateSketchTool()
	tool.Start(s)
	if tool.Name() == "" || tool.Prompt(s) == "" {
		t.Error("Create Sketch tool should expose a name and a prompt")
	}
	// Committing with no plane picked errors (and restores the filter).
	if err := tool.Commit(s); err == nil {
		t.Error("committing with no plane should error")
	}
}

func TestPickAtUsesInstalledPicker(t *testing.T) {
	s, def := emptyPartSession(t)
	if _, ok := s.PickAt(1, 1, NewSelectionFilter()); ok {
		t.Error("PickAt with no picker should miss")
	}
	plane := def.OriginPlanes()[0]
	s.SetPicker(stubPlanePicker{handle: WorkPlaneHandle{Plane: plane}})
	sel, ok := s.PickAt(1, 1, NewSelectionFilter(SelectWorkPlane))
	if !ok || sel.(WorkPlaneHandle).Plane != plane {
		t.Errorf("PickAt = %v/%v, want the installed picker's plane", sel, ok)
	}
}

func TestSelectBrowserNodeIgnoresNonSelectable(t *testing.T) {
	s, _ := emptyPartSession(t)
	s.SelectBrowserNode(BrowserNode{Label: "Parameters", Kind: "parameters"}) // no Select
	if s.Selection().Count() != 0 {
		t.Error("a non-selectable browser node should not change the selection")
	}
}

func TestCreateSketchToolCancelRestoresFilter(t *testing.T) {
	s, _ := emptyPartSession(t)
	tool := NewCreateSketchTool()
	s.StartTool(tool)
	// The tool restricts to the sketch hosts (work planes and planar faces), so edges
	// (not a host) are excluded while it is active.
	if !s.Selection().Filter().IsRestricted() || s.Selection().Filter().Accepts(SelectEdge) {
		t.Error("Create Sketch tool should restrict the filter to sketch hosts (planes and faces)")
	}
	s.CancelTool()
	if !s.Selection().Filter().Accepts(SelectEdge) {
		t.Error("cancelling should restore the all-accepting filter")
	}
}
