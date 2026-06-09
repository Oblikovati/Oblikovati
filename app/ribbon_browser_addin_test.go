// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/sketch"
)

func TestBuildRibbonFromCommands(t *testing.T) {
	s, _ := emptyPartSession(t) // a part is active, so default-Part commands appear (not ZeroDoc)
	_ = s.Commands().Add(NewCommand("Sketch.Line", "Line", "Sketch", noop))
	_ = s.Commands().Add(NewCommand("Sketch.Circle", "Circle", "Sketch", noop))
	on := false
	_ = s.Commands().Add(NewCommand("Create.Extrude", "Extrude", "Create", noop).
		WithEnable(func(*Session) bool { return on }))

	r := BuildRibbon(s)
	sk, ok := r.Panel("Sketch")
	if !ok || len(sk.Buttons) != 2 {
		t.Fatalf("Sketch panel = %+v ok=%v, want 2 buttons", sk, ok)
	}
	create, _ := r.Panel("Create")
	if create.Buttons[0].Enabled {
		t.Error("disabled command should render a disabled button")
	}
	on = true
	create, _ = BuildRibbon(s).Panel("Create")
	if !create.Buttons[0].Enabled {
		t.Error("button did not re-enable when its predicate flipped")
	}
}

func TestBuildRibbonGroupsByTabThenPanel(t *testing.T) {
	s, _ := emptyPartSession(t) // a part is active, so default-Part commands appear (not ZeroDoc)
	_ = s.Commands().Add(NewCommand("Sketch.Line", "Line", "Create", noop).WithTab("Sketch"))
	_ = s.Commands().Add(NewCommand("Sketch.Rectangle", "Rectangle", "Create", noop).WithTab("Sketch"))
	_ = s.Commands().Add(NewCommand("Model.Extrude", "Extrude", "Create", noop).WithTab("3D Model"))
	_ = s.Commands().Add(NewCommand("Loose.Tool", "Tool", "Misc", noop)) // no tab ⇒ DefaultTab

	r := BuildRibbon(s)
	if len(r.Tabs) != 3 || r.Tabs[0].Name != "Sketch" || r.Tabs[1].Name != "3D Model" {
		t.Fatalf("tabs = %+v, want [Sketch, 3D Model, %s] in registration order", tabNames(r), DefaultTab)
	}
	// The two Sketch-tab commands collapse into one shared "Create" panel.
	sketch, ok := r.Tab("Sketch")
	if !ok || len(sketch.Panels) != 1 || len(sketch.Panels[0].Buttons) != 2 {
		t.Fatalf("Sketch tab = %+v, want one Create panel with 2 buttons", sketch)
	}
	// A command with no tab lands on the default tab.
	if def, ok := r.Tab(DefaultTab); !ok || len(def.Panels) != 1 || def.Panels[0].Name != "Misc" {
		t.Fatalf("default tab = %+v, want a Misc panel", def)
	}
}

func tabNames(r Ribbon) []string {
	names := make([]string, len(r.Tabs))
	for i, t := range r.Tabs {
		names[i] = t.Name
	}
	return names
}

func TestBrowserReflectsPartStructure(t *testing.T) {
	s, def := emptyPartSession(t)
	_, _ = def.Parameters().AddUserParameter("width", "10 mm")
	def.Sketches().Add(sketch.XYPlane())

	root := BuildBrowser(s)
	if root.Kind != "document" {
		t.Fatalf("root kind = %q, want document", root.Kind)
	}
	byKind := map[string]int{}
	for _, c := range root.Children {
		byKind[c.Kind]++
	}
	if byKind["parameters"] != 1 || byKind["sketches"] != 1 {
		t.Errorf("browser branches = %v, want parameters+sketches", byKind)
	}
	// The parameter we added shows under Parameters.
	for _, c := range root.Children {
		if c.Kind == "parameters" {
			if len(c.Children) != 1 || c.Children[0].Label != "width" {
				t.Errorf("parameters branch = %+v, want [width]", c.Children)
			}
		}
	}
}

func TestBrowserEmptySession(t *testing.T) {
	if BuildBrowser(NewSession()).Kind != "document" {
		t.Error("empty session browser should still have a document root")
	}
}

// sampleAddIn is a minimal add-in that, on activation, registers an Extrude command
// which launches the interactive tool — the M05 exit criterion.
type sampleAddIn struct{ id string }

func (a sampleAddIn) ID() string { return a.id }
func (a sampleAddIn) Activate(s *Session) error {
	return s.Commands().Add(NewCommand("AddIn.Extrude", "Extrude", "AddInTab", func(sess *Session) error {
		sess.StartTool(NewExtrudeTool())
		return nil
	}).WithAlias("E"))
}
func (a sampleAddIn) Deactivate(*Session) error { return nil }

func TestAddInAddsRibbonButtonThatRunsTool(t *testing.T) {
	s, _ := emptyPartSession(t) // an add-in's command defaults to the Part ribbon, so a part must be active
	if err := s.AddIns().Register(sampleAddIn{id: "acme.extrude"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	// Before activation the command/button do not exist.
	if _, ok := s.Commands().ByID("AddIn.Extrude"); ok {
		t.Fatal("add-in command present before activation")
	}
	if err := s.AddIns().Activate(s, "acme.extrude"); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if !s.AddIns().IsActive("acme.extrude") {
		t.Error("add-in not marked active")
	}
	// The exit criterion: the add-in's command now appears as a ribbon button…
	panel, ok := BuildRibbon(s).Panel("AddInTab")
	if !ok || len(panel.Buttons) != 1 || panel.Buttons[0].Command.DisplayName() != "Extrude" {
		t.Fatalf("add-in ribbon button missing: %+v", panel)
	}
	// …and running it starts the interactive tool.
	if err := s.Execute("AddIn.Extrude"); err != nil {
		t.Fatalf("run add-in command: %v", err)
	}
	if s.ActiveTool() == nil || s.ActiveTool().Name() != "Extrude" {
		t.Error("add-in command did not start the Extrude tool")
	}
}

func TestAddInManagerLifecycle(t *testing.T) {
	s := NewSession()
	a := sampleAddIn{id: "x"}
	_ = s.AddIns().Register(a)
	if err := s.AddIns().Register(a); err == nil {
		t.Error("duplicate add-in registration accepted")
	}
	if err := s.AddIns().Activate(s, "missing"); err == nil {
		t.Error("activating an unknown add-in should error")
	}
	_ = s.AddIns().Activate(s, "x")
	_ = s.AddIns().Activate(s, "x") // idempotent
	_ = s.AddIns().Deactivate(s, "x")
	if s.AddIns().IsActive("x") {
		t.Error("add-in still active after Deactivate")
	}
	if len(s.AddIns().Registered()) != 1 {
		t.Errorf("Registered = %v, want 1", s.AddIns().Registered())
	}
}

// TestAddInUnregisterEnablesReload covers the hot-reload path: an active add-in
// cannot be unregistered, but once deactivated it can be removed and a replacement
// re-registered under the same id.
func TestAddInUnregisterEnablesReload(t *testing.T) {
	s := NewSession()
	a := sampleAddIn{id: "acme.extrude"}
	_ = s.AddIns().Register(a)
	_ = s.AddIns().Activate(s, a.id)

	if err := s.AddIns().Unregister(a.id); err == nil {
		t.Error("unregistering an active add-in should error (handlers would leak)")
	}
	_ = s.AddIns().Deactivate(s, a.id)
	if err := s.AddIns().Unregister(a.id); err != nil {
		t.Fatalf("Unregister after Deactivate: %v", err)
	}
	if len(s.AddIns().Registered()) != 0 {
		t.Errorf("Registered = %v, want empty after Unregister", s.AddIns().Registered())
	}
	if err := s.AddIns().Unregister(a.id); err == nil {
		t.Error("unregistering an unknown add-in should error")
	}
	// The id is free again, so a replacement library registers under it — the reload
	// outcome (re-registration is what hot-reload needs the manager to permit).
	if err := s.AddIns().Register(sampleAddIn{id: a.id}); err != nil {
		t.Fatalf("re-register after Unregister: %v", err)
	}
}

// failAddIn errors on both Activate and Deactivate to exercise the error paths.
type failAddIn struct {
	activated bool
}

func (a *failAddIn) ID() string { return "fail" }
func (a *failAddIn) Activate(*Session) error {
	if !a.activated {
		a.activated = true
		return errBoom
	}
	return nil
}
func (a *failAddIn) Deactivate(*Session) error { return errBoom }

func TestAddInActivateAndDeactivateErrorsPropagate(t *testing.T) {
	s := NewSession()
	a := &failAddIn{}
	_ = s.AddIns().Register(a)
	if err := s.AddIns().Activate(s, "fail"); err == nil {
		t.Fatal("Activate error not propagated")
	}
	if s.AddIns().IsActive("fail") {
		t.Error("add-in marked active despite Activate error")
	}
	// Second attempt succeeds (activated flag flipped), so it becomes active…
	if err := s.AddIns().Activate(s, "fail"); err != nil {
		t.Fatalf("second Activate: %v", err)
	}
	// …and a failing Deactivate propagates while leaving it active.
	if err := s.AddIns().Deactivate(s, "fail"); err == nil {
		t.Error("Deactivate error not propagated")
	}
	if !s.AddIns().IsActive("fail") {
		t.Error("add-in should remain active when Deactivate errors")
	}
}

func TestBrowserShowsFeatureNodes(t *testing.T) {
	s, profile := newPartWithSquare(t, 2)
	s.SetPicker(stubPicker{sel: profile})
	ext := NewExtrudeTool()
	s.StartTool(ext)
	s.Click(120, 90)
	ext.SetDistance(5)
	if err := s.OK(); err != nil {
		t.Fatalf("extrude OK: %v", err)
	}

	for _, c := range BuildBrowser(s).Children {
		if c.Kind == "feature" {
			return // a feature node appeared — the history branch is covered
		}
	}
	t.Error("browser shows no feature node after an extrude")
}

var errBoom = boomError("boom")

type boomError string

func (e boomError) Error() string { return string(e) }

func noop(*Session) error { return nil }
