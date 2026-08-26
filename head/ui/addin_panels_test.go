// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"regexp"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// TestPanelBufferReseedsOnDeclaredChange covers the editable-control text buffer: it seeds from
// the declared value, persists a user edit, but re-seeds in place when the add-in pushes a
// different value (populating a form from a loaded document) without clobbering an echoed edit.
func TestPanelBufferReseedsOnDeclaredChange(t *testing.T) {
	delete(panelEditBuffers, "w/poles")
	delete(panelDeclared, "w/poles")

	if got := bufString(panelBuffer("w/poles", "10")); got != "10" {
		t.Fatalf("seed = %q, want 10", got)
	}
	if got := bufString(panelBuffer("w/poles", "10")); got != "10" {
		t.Errorf("re-set same value = %q, want 10", got)
	}
	if got := bufString(panelBuffer("w/poles", "12")); got != "12" {
		t.Errorf("after add-in pushed 12, buffer = %q, want 12", got)
	}
}

// TestSyncMaterialSelectionFollowsActiveDocument covers the Materials selector resync: switching
// the active document repoints the selectors at that part's assigned material.
func TestSyncMaterialSelectionFollowsActiveDocument(t *testing.T) {
	s := app.NewSession()
	mats := s.Materials().Materials()
	if len(mats) < 2 {
		t.Skip("need two materials")
	}
	matSelectionSynced = false
	selectedMaterialID = ""

	if _, err := s.NewPart(); err != nil {
		t.Fatalf("new part A: %v", err)
	}
	if err := s.AssignMaterial("", mats[0].ID()); err != nil {
		t.Fatalf("assign A: %v", err)
	}
	syncMaterialSelection(s)
	if selectedMaterialID != mats[0].ID() {
		t.Errorf("after A, selected = %q, want %q", selectedMaterialID, mats[0].ID())
	}

	if _, err := s.NewPart(); err != nil {
		t.Fatalf("new part B: %v", err)
	}
	if err := s.AssignMaterial("", mats[1].ID()); err != nil {
		t.Fatalf("assign B: %v", err)
	}
	syncMaterialSelection(s)
	if selectedMaterialID != mats[1].ID() {
		t.Errorf("after switching to B, selected = %q, want %q (stale A)", selectedMaterialID, mats[1].ID())
	}
}

// editableFormWindow is a dockable window with one of every editable control kind.
func editableFormWindow() wire.DockableWindowSpec {
	return wire.DockableWindowSpec{
		ID: "form", Title: "Form", Visible: true,
		Controls: []wire.PanelControlSpec{
			{Kind: types.PanelLabel, Text: "— header —"},
			{Kind: types.PanelTextBox, ID: "name", Text: "Name", Value: "x"},
			{Kind: types.PanelValueEditor, ID: "len", Text: "Length", Value: "5 mm"},
			{Kind: types.PanelCheckBox, ID: "on", Text: "On", Value: "true"},
			{Kind: types.PanelDropdown, ID: "type", Text: "Type", Options: []string{"a", "b"}, Value: "a"},
			{Kind: types.PanelComboBox, ID: "grade", Text: "Grade", Value: "N42"},
			{Kind: types.PanelSlider, ID: "arc", Text: "Arc", Value: "0.8", Min: 0, Max: 1, Step: 0.01},
			{Kind: types.PanelButton, ID: "go", Text: "Generate", CommandID: "X.Go"},
			{Kind: types.PanelSeparator},
		},
	}
}

// TestInWindowAddInPanelRendersEditableControls drives a real frame rendering a dockable window
// with every editable control kind, covering drawAddInPanelControl + drawPanelDropdown (the
// native widget paths). Skips when no Vulkan is available.
func TestInWindowAddInPanelRendersEditableControls(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := app.NewSession()
	if err := s.SetDockableWindow(editableFormWindow()); err != nil {
		t.Fatalf("SetDockableWindow: %v", err)
	}
	for range 2 { // two frames: immediate-mode buffers seed on the first
		win.BeginFrame()
		drawAddInPanels(s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	if _, ok := panelEditBuffers["form/name"]; !ok {
		t.Error("text-control buffer was not seeded while rendering the panel")
	}
}

// TestAddInPanelValueControlsStackTheirLabel is the #1490 guard: an add-in dockable panel is
// narrow and its captions are long descriptive sentences ("Pressure on loaded faces (MPa)").
// Passing the caption straight into an ImGui value widget draws it to the RIGHT of the
// ~65%-wide control, so it is cropped against the panel edge. The fix stacks the caption on its
// own line (panelFieldLabel → TextWrapped) above a full-width input, with the widget's own label
// suppressed ("##field"). This guard fails if a value control regresses to the label-on-the-right
// form, so the cropping cannot come back.
func TestAddInPanelValueControlsStackTheirLabel(t *testing.T) {
	src, err := os.ReadFile("addin_panels.go")
	if err != nil {
		t.Fatalf("read addin_panels.go: %v", err)
	}
	text := string(src)

	// No value widget may take control.Text as its (right-side) label.
	banned := map[string]*regexp.Regexp{
		"InputText":   regexp.MustCompile(`native\.InputText\(\s*control\.Text`),
		"SliderFloat": regexp.MustCompile(`native\.SliderFloat\(\s*control\.Text`),
		"BeginCombo":  regexp.MustCompile(`native\.BeginCombo\(\s*control\.Text`),
	}
	for widget, re := range banned {
		if re.MatchString(text) {
			t.Errorf("addin_panels.go passes control.Text straight into native.%s — the caption draws "+
				"to the widget's right and crops in a narrow docked panel (#1490). Stack it with "+
				"panelFieldLabel and pass a suppressed \"##\" label instead.", widget)
		}
	}

	// And the stacked-label helper must exist and be used by the value controls.
	if !regexp.MustCompile(`func panelFieldLabel\(`).MatchString(text) {
		t.Error("addin_panels.go must define panelFieldLabel (stacks a wrapped caption above a full-width input, #1490)")
	}
	if !regexp.MustCompile(`panelFieldLabel\(control\.Text\)`).MatchString(text) {
		t.Error("addin_panels.go value controls must call panelFieldLabel(control.Text) (#1490)")
	}
}
