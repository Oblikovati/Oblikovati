// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// registeredSession returns a part session with the standard ribbon commands wired.
func registeredSession(t *testing.T) *Session {
	t.Helper()
	s, _ := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	return s
}

func TestStandardRibbonHasSketchCreatePanel(t *testing.T) {
	s := registeredSession(t)
	r := BuildRibbon(s)
	tab, ok := r.Tab("Sketch")
	if !ok {
		t.Fatal("ribbon has no Sketch tab")
	}
	panel, ok := tab.Panel("Create")
	if !ok {
		t.Fatal("Sketch tab has no Create panel")
	}
	// Line, Rectangle, Circle, Arc, Spline, Ellipse, Polygon, Point.
	if len(panel.Buttons) != 8 {
		t.Errorf("Sketch Create panel has %d tools, want 8", len(panel.Buttons))
	}
}

func TestModelTabHasCreate2DSketch(t *testing.T) {
	s := registeredSession(t)
	r := BuildRibbon(s)
	tab, ok := r.Tab("3D Model")
	if !ok {
		t.Fatal("ribbon has no 3D Model tab")
	}
	if _, ok := tab.Panel("Sketch"); !ok {
		t.Error("3D Model tab has no Sketch panel (Create 2D Sketch)")
	}
}

func TestSketchToolsDisabledOutsideSketchEnvironment(t *testing.T) {
	s := registeredSession(t)
	line, _ := s.Commands().ByID("Sketch.Line")
	if line.IsEnabled(s) {
		t.Error("Line should be disabled outside the sketch environment")
	}
	create, _ := s.Commands().ByID("Sketch.Create2D")
	if !create.IsEnabled(s) {
		t.Error("Create 2D Sketch should be enabled on an active part")
	}
	finish, _ := s.Commands().ByID("Sketch.Finish")
	if finish.IsEnabled(s) {
		t.Error("Finish Sketch should be disabled outside the sketch environment")
	}
}

// enterSketchEnv pre-selects the XY plane and runs Create 2D Sketch, so the session is
// in the sketch environment (the pre-selected-plane path, no interactive pick needed).
func enterSketchEnv(t *testing.T, s *Session) {
	t.Helper()
	origin := originFolder(BuildBrowser(s))
	s.SelectBrowserNode(origin.Children[0]) // XY Plane
	if err := s.Execute("Sketch.Create2D"); err != nil {
		t.Fatalf("Create 2D Sketch: %v", err)
	}
	if !s.InSketch() {
		t.Fatal("Create 2D Sketch with a pre-selected plane did not enter the environment")
	}
}

func TestSketchToolsEnableInsideSketchEnvironment(t *testing.T) {
	s := registeredSession(t)
	enterSketchEnv(t, s)
	for _, id := range []string{"Sketch.Line", "Sketch.Circle", "Sketch.Arc", "Sketch.Polygon", "Sketch.Finish"} {
		c, _ := s.Commands().ByID(id)
		if !c.IsEnabled(s) {
			t.Errorf("%s should be enabled inside the sketch environment", id)
		}
	}
	// Create 2D Sketch is disabled while a sketch is open (no nesting).
	create, _ := s.Commands().ByID("Sketch.Create2D")
	if create.IsEnabled(s) {
		t.Error("Create 2D Sketch should be disabled while editing a sketch")
	}
}

func TestExecuteSketchToolStartsTool(t *testing.T) {
	s := registeredSession(t)
	enterSketchEnv(t, s)
	if err := s.Execute("Sketch.Circle"); err != nil {
		t.Fatalf("execute Circle: %v", err)
	}
	if s.ActiveTool() == nil || s.ActiveTool().Name() != "Circle" {
		t.Error("executing Sketch.Circle did not start the Circle tool")
	}
}

func TestExecuteFinishSketchExits(t *testing.T) {
	s := registeredSession(t)
	enterSketchEnv(t, s)
	if err := s.Execute("Sketch.Finish"); err != nil {
		t.Fatalf("execute Finish: %v", err)
	}
	if s.InSketch() {
		t.Error("executing Sketch.Finish did not leave the sketch environment")
	}
}

func TestExtrudeDisabledInsideSketch(t *testing.T) {
	s := registeredSession(t)
	ext, _ := s.Commands().ByID("Create.Extrude")
	if !ext.IsEnabled(s) {
		t.Error("Extrude should be enabled in the part environment")
	}
	enterSketchEnv(t, s)
	if ext.IsEnabled(s) {
		t.Error("Extrude should be disabled while editing a sketch")
	}
}

func TestStandardCommandsRegisterOnce(t *testing.T) {
	s := registeredSession(t)
	if err := RegisterStandardCommands(s); err == nil {
		t.Error("registering the standard commands twice should error on duplicate ids")
	}
}
