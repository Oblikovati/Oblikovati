// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"reflect"
	"testing"
)

// TestWithTabsRibbonTabs covers the multi-tab primitive directly: the first tab is primary
// (Tab()), all are returned by ribbonTabs, and an unset tab falls back to the default.
func TestWithTabsRibbonTabs(t *testing.T) {
	multi := NewCommand("X", "X", "Cat", noop).WithTabs("A", "B", "C")
	if multi.Tab() != "A" {
		t.Errorf("primary tab = %q, want A", multi.Tab())
	}
	if got := multi.ribbonTabs(); !reflect.DeepEqual(got, []string{"A", "B", "C"}) {
		t.Errorf("ribbonTabs = %v, want [A B C]", got)
	}
	if got := NewCommand("Y", "Y", "Cat", noop).ribbonTabs(); !reflect.DeepEqual(got, []string{DefaultTab}) {
		t.Errorf("unset ribbonTabs = %v, want [%s]", got, DefaultTab)
	}
	if got := NewCommand("Z", "Z", "Cat", noop).WithTabs().ribbonTabs(); !reflect.DeepEqual(got, []string{DefaultTab}) {
		t.Errorf("WithTabs() with no args = %v, want [%s]", got, DefaultTab)
	}
}

// TestSketchPanelAppearsOnBothModellingTabs locks the WithTabs multi-tab behaviour: the sketch
// starters repeat on the Create & Modify and Surfaces & Mesh tabs from a single registration.
func TestSketchPanelAppearsOnBothModellingTabs(t *testing.T) {
	r := BuildRibbon(registeredSession(t))
	for _, tabName := range []string{"Create & Modify", "Surfaces & Mesh"} {
		tab, ok := r.Tab(tabName)
		if !ok {
			t.Fatalf("Part ribbon has no %q tab", tabName)
		}
		panel, ok := tab.Panel("Sketch")
		if !ok {
			t.Fatalf("%q tab has no Sketch panel", tabName)
		}
		if _, ok := buttonNamed(panel, "New 2D Sketch"); !ok {
			t.Errorf("%q tab's Sketch panel is missing New 2D Sketch", tabName)
		}
	}
}

// TestWorkFeaturesPanelHeadsCreateModifyTab locks the panel order: Work Features is first.
func TestWorkFeaturesPanelHeadsCreateModifyTab(t *testing.T) {
	tab, ok := BuildRibbon(registeredSession(t)).Tab("Create & Modify")
	if !ok {
		t.Fatal("Part ribbon has no Create & Modify tab")
	}
	if len(tab.Panels) == 0 || tab.Panels[0].Name != "Work Features" {
		t.Errorf("first panel = %q, want Work Features", firstPanelName(tab))
	}
}

func firstPanelName(t RibbonTab) string {
	if len(t.Panels) == 0 {
		return "<none>"
	}
	return t.Panels[0].Name
}

// TestSurfacesMeshTabPanelOrder locks the requested panel order on the new tab.
func TestSurfacesMeshTabPanelOrder(t *testing.T) {
	tab, ok := BuildRibbon(registeredSession(t)).Tab("Surfaces & Mesh")
	if !ok {
		t.Fatal("Part ribbon has no Surfaces & Mesh tab")
	}
	want := []string{"Sketch", "Surface", "Freeform", "Mesh", "Point Cloud", "Mold"}
	if len(tab.Panels) != len(want) {
		t.Fatalf("Surfaces & Mesh has %d panels, want %d", len(tab.Panels), len(want))
	}
	for i, name := range want {
		if tab.Panels[i].Name != name {
			t.Errorf("panel[%d] = %q, want %q", i, tab.Panels[i].Name, name)
		}
	}
}

// Test3DSketchTabIsContextual: the 3D Sketch tab is absent on the part ribbon and appears only
// once a 3D sketch is open (its own environment, distinct from the 2D Sketch tab).
func Test3DSketchTabIsContextual(t *testing.T) {
	s := registeredSession(t)
	if _, ok := BuildRibbon(s).Tab("3D Sketch"); ok {
		t.Error("3D Sketch tab should be absent outside the 3D-sketch environment")
	}
	if CurrentEnvironment(s) != BaseEnvironment {
		t.Errorf("environment = %v, want base", CurrentEnvironment(s))
	}
	if _, err := s.CreateSketch3D(); err != nil {
		t.Fatalf("CreateSketch3D: %v", err)
	}
	if CurrentEnvironment(s) != Sketch3DEnvironment {
		t.Errorf("environment in 3D sketch = %v, want sketch3d", CurrentEnvironment(s))
	}
	tab, ok := BuildRibbon(s).Tab("3D Sketch")
	if !ok {
		t.Fatal("3D Sketch tab should appear inside the 3D-sketch environment")
	}
	if _, ok := tab.Panel("Exit"); !ok {
		t.Error("3D Sketch tab has no Exit panel (Finish 3D Sketch)")
	}
}

// Test2DAnd3DSketchTabsNeverCoexist: opening a 2D sketch shows the Sketch tab, not the 3D one.
func Test2DAnd3DSketchTabsNeverCoexist(t *testing.T) {
	s := registeredSession(t)
	enterSketchEnv(t, s)
	r := BuildRibbon(s)
	if _, ok := r.Tab("Sketch"); !ok {
		t.Error("2D Sketch tab should be present in the 2D-sketch environment")
	}
	if _, ok := r.Tab("3D Sketch"); ok {
		t.Error("3D Sketch tab must not appear in the 2D-sketch environment")
	}
}

// TestViewAndInspectTabsOnAssemblyRibbon: the View and Inspect tabs are shared with the
// Assembly ribbon (onRibbons), not Part-only.
func TestViewAndInspectTabsOnAssemblyRibbon(t *testing.T) {
	s := registeredSession(t)
	if _, err := s.NewAssembly(); err != nil {
		t.Fatalf("NewAssembly: %v", err)
	}
	r := BuildRibbon(s)
	if r.Key != AssemblyRibbon {
		t.Fatalf("ribbon key = %q, want Assembly", r.Key)
	}
	for _, name := range []string{"View", "Inspect"} {
		if _, ok := r.Tab(name); !ok {
			t.Errorf("Assembly ribbon is missing the %q tab", name)
		}
	}
}

// TestGetStartedManagePanel: the ZeroDoc ribbon offers the AddIn Catalogue and Preferences
// launch buttons, and each command raises the matching head-window request.
func TestGetStartedManagePanel(t *testing.T) {
	s := zeroDocSession(t)
	panel, ok := BuildRibbon(s).Panel("Manage")
	if !ok {
		t.Fatal("Get Started ribbon has no Manage panel")
	}
	for _, name := range []string{"AddIn Catalogue", "Preferences"} {
		if _, ok := buttonNamed(panel, name); !ok {
			t.Errorf("Manage panel is missing %q", name)
		}
	}
	if err := s.Execute("GetStarted.AddInCatalogue"); err != nil {
		t.Fatalf("AddInCatalogue: %v", err)
	}
	if !s.TakeAddInCatalogueRequest() {
		t.Error("AddInCatalogue command did not raise the catalogue request")
	}
	if err := s.Execute("GetStarted.Preferences"); err != nil {
		t.Fatalf("Preferences: %v", err)
	}
	if !s.TakePreferencesRequest() {
		t.Error("Preferences command did not raise the preferences request")
	}
}

// TestWindowOpenRequestsAreOneShot: Take* returns true once, then false (one-shot consumption).
func TestWindowOpenRequestsAreOneShot(t *testing.T) {
	s := NewSession()
	if s.TakeAddInCatalogueRequest() || s.TakePreferencesRequest() {
		t.Fatal("requests should start clear")
	}
	s.RequestAddInCatalogue()
	s.RequestPreferences()
	if !s.TakeAddInCatalogueRequest() || s.TakeAddInCatalogueRequest() {
		t.Error("AddInCatalogue request should be consumed exactly once")
	}
	if !s.TakePreferencesRequest() || s.TakePreferencesRequest() {
		t.Error("Preferences request should be consumed exactly once")
	}
}
