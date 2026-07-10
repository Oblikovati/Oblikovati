//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"os"
	"strconv"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Add-in dockable windows (M05-F03, #247): each visible declared window renders as a
// real ImGui window — closable, and dockable into the chrome's dockspace by the user
// like any panel. Content is the declared control list; a button executes the command
// it names, so the owning add-in observes clicks through command.ended.

// addInPanelDefaultSize is a sane first-show size for a declared window; afterwards
// ImGui's saved settings keep whatever the user made of it.
const addInPanelDefaultW, addInPanelDefaultH = 280, 200

// drawAddInPanels renders every visible add-in dockable window. Closing one via its
// title-bar X routes through the session so the owning add-in receives the
// visibility event (DockableWindowsEvents OnHide).
func drawAddInPanels(s *app.Session) {
	for _, spec := range s.DockableWindows().List() {
		if !spec.Visible {
			continue
		}
		drawAddInPanel(s, spec)
	}
}

// addInDockRightNode is the lazily created right band: the default layout has no
// right node (it would be empty and ImGui collapses empty nodes), so the first
// DockRight window splits one off the central node.
var addInDockRightNode uint32

// applyInitialDock docks the next window into the band its spec asks for, on first
// appearance only (FirstUseEver — the user's re-docking wins afterwards).
func applyInitialDock(dock types.DockingState) {
	switch dock {
	case types.DockLeft:
		native.SetNextWindowDock(dockSideNodes.Left)
	case types.DockBottom:
		native.SetNextWindowDock(dockSideNodes.Bottom)
	case types.DockRight:
		if addInDockRightNode == 0 && dockSideNodes.Center != 0 {
			addInDockRightNode = native.DockSplit(&dockSideNodes.Center, 1, 0.25)
		}
		native.SetNextWindowDock(addInDockRightNode)
	default: // DockFloating: leave it free
	}
}

// drawAddInPanel renders one declared window and its controls.
func drawAddInPanel(s *app.Session, spec wire.DockableWindowSpec) {
	native.SetNextWindowSizeOnce(addInPanelDefaultW, addInPanelDefaultH)
	applyInitialDock(spec.Dock)
	visible, open := native.BeginClosable(spec.Title + "###addin-" + spec.ID)
	if visible {
		drawControlList(s, spec.ID, spec.Controls)
	}
	native.End()
	if !open {
		if err := s.SetDockableWindowVisible(spec.ID, false); err != nil {
			fmt.Fprintf(os.Stderr, "add-in window %q: %v\n", spec.ID, err)
		}
	}
}

// panelEditBuffers holds the live text buffers for editable string controls (text box, value
// editor, combo box), keyed by "windowID/controlID"; panelDeclared remembers the last declared
// Value. The buffer persists across frames so editing is stable, but when the ADD-IN pushes a
// different value (e.g. populating the form from a loaded document) and the buffer doesn't
// already hold it, it is re-seeded in place — so a programmatic Value change shows up while a
// user's own echoed edits never clobber the field.
var (
	panelEditBuffers = map[string][]byte{}
	panelDeclared    = map[string]string{}
)

func panelBuffer(key, value string) []byte {
	buf, ok := panelEditBuffers[key]
	if !ok {
		buf = make([]byte, 256)
		copy(buf, value)
		panelEditBuffers[key] = buf
		panelDeclared[key] = value
		return buf
	}
	if panelDeclared[key] != value && bufString(buf) != value {
		for i := range buf {
			buf[i] = 0
		}
		copy(buf, value)
	}
	panelDeclared[key] = value
	return buf
}

// drawControlList renders a vertical run of controls — a panel body, or a group/tab pane.
// Each control's index joins the id stack (in drawAddInPanelControl) so nested controls get
// collision-free ImGui ids from their structural path while keeping their flat logical ID.
func drawControlList(s *app.Session, windowID string, controls []wire.PanelControlSpec) {
	for i, control := range controls {
		drawAddInPanelControl(s, windowID, i, control)
	}
}

// drawPanelContainer renders the container kinds (grid/group/tabs, ADR-0019), which recurse
// into their Children, and reports whether control was one — keeping the leaf dispatch in
// drawAddInPanelControl simple.
func drawPanelContainer(s *app.Session, windowID string, control wire.PanelControlSpec) bool {
	switch control.Kind {
	case types.PanelGrid:
		drawGrid(s, windowID, control)
	case types.PanelGroup:
		drawGroup(s, windowID, control)
	case types.PanelTabs:
		drawTabs(s, windowID, control)
	default:
		return false
	}
	return true
}

// drawAddInPanelControl renders one declared control by kind, pushing edits back to the owning
// add-in via Session.PanelValueChanged. The index joins the id stack so two controls never
// collide. Container kinds (grid/group/tabs, ADR-0019) recurse into their Children.
func drawAddInPanelControl(s *app.Session, windowID string, index int, control wire.PanelControlSpec) {
	native.PushIDInt(index)
	defer native.PopID()
	if drawPanelContainer(s, windowID, control) || drawEditableControl(s, windowID, control) {
		return
	}
	switch control.Kind {
	case types.PanelButton:
		if native.Button(control.Text) && control.CommandID != "" {
			if err := s.Execute(control.CommandID); err != nil {
				fmt.Fprintf(os.Stderr, "add-in window %q button %q: %v\n", windowID, control.CommandID, err)
			}
		}
	case types.PanelSeparator:
		native.Separator()
	default: // PanelLabel renders its text. A future unknown CONTAINER kind degrades to a
		// vertical stack of its children rather than vanishing; a leaf degrades to its text.
		if len(control.Children) > 0 {
			drawControlList(s, windowID, control.Children)
		} else {
			native.TextWrapped(control.Text)
		}
	}
}

// panelEditSession is the ≤1-method view of the session an editable panel widget needs (audit I5,
// the arrowSession pattern): just the sink that reports a user edit back to the owning add-in, so
// tree/table/field widgets don't couple to the whole *app.Session. *app.Session satisfies it.
type panelEditSession interface {
	PanelValueChanged(windowID, controlID, value string)
}

var _ panelEditSession = (*app.Session)(nil)

// drawEditableControl renders the editable control kinds with the stacked-caption layout
// (#1490) and reports whether control was one, keeping the leaf dispatch small. Edits push back
// through Session.PanelValueChanged.
func drawEditableControl(s *app.Session, windowID string, control wire.PanelControlSpec) bool {
	switch control.Kind {
	case types.PanelTextBox, types.PanelValueEditor, types.PanelComboBox:
		drawPanelTextField(s, windowID, control)
	case types.PanelCheckBox:
		drawPanelCheckBox(s, windowID, control)
	case types.PanelDropdown:
		drawPanelDropdown(s, windowID, control)
	case types.PanelSlider:
		drawPanelSlider(s, windowID, control)
	case types.PanelReferenceList:
		drawPanelReferenceList(s, windowID, control)
	case types.PanelTree:
		drawPanelTree(s, windowID, control)
	case types.PanelTable:
		drawPanelTable(s, windowID, control)
	default:
		return false
	}
	return true
}

// drawPanelTextField renders a stacked-caption text/combo input (#1490), pushing the new value to
// the add-in on change. Split out of drawEditableControl to keep the dispatch switch's statement
// count under the funlen gate as more control kinds land.
func drawPanelTextField(s panelEditSession, windowID string, control wire.PanelControlSpec) {
	buf := panelBuffer(windowID+"/"+control.ID, control.Value)
	panelFieldLabel(control.Text)
	if native.InputText("##field", buf) {
		s.PanelValueChanged(windowID, control.ID, bufString(buf))
	}
}

// drawPanelCheckBox renders a checkbox control, pushing the new state to the add-in on toggle.
func drawPanelCheckBox(s panelEditSession, windowID string, control wire.PanelControlSpec) {
	checked := control.Value == "true"
	if native.Checkbox(control.Text, &checked) {
		s.PanelValueChanged(windowID, control.ID, strconv.FormatBool(checked))
	}
}

// drawPanelSlider renders a bounded numeric slider with its caption stacked above (#1490),
// pushing the new value to the add-in on change.
func drawPanelSlider(s *app.Session, windowID string, control wire.PanelControlSpec) {
	v, _ := strconv.ParseFloat(control.Value, 64)
	f := float32(v)
	panelFieldLabel(control.Text)
	if native.SliderFloat("##field", &f, float32(control.Min), float32(control.Max)) {
		s.PanelValueChanged(windowID, control.ID, strconv.FormatFloat(float64(f), 'g', -1, 64))
	}
}

// panelFieldLabel draws an editable control's caption on its own line above a full-width input,
// then widens the next item to fill the panel. Add-in labels are long, descriptive captions
// ("Pressure on loaded faces (MPa)"); ImGui's default label-to-the-RIGHT-of-the-widget layout,
// with the widget at ~65% of the panel, cropped them against a narrow docked panel's right edge
// (#1490). Stacking a wrapped label above a full-width input keeps the whole caption readable at
// any panel width.
func panelFieldLabel(text string) {
	native.TextWrapped(text)
	native.SetNextItemWidth(-1) // fill the panel width; the input's own label is suppressed ("##field")
}

// drawPanelDropdown renders a single-select dropdown; picking an option pushes it to the add-in.
// The label is stacked above a full-width combo (#1490), like the other value controls.
func drawPanelDropdown(s *app.Session, windowID string, control wire.PanelControlSpec) {
	panelFieldLabel(control.Text)
	if !native.BeginCombo("##field", control.Value) {
		return
	}
	for _, opt := range control.Options {
		if native.Selectable(opt, opt == control.Value) {
			s.PanelValueChanged(windowID, control.ID, opt)
		}
	}
	native.EndCombo()
}

const (
	refListRowHeight = 22
	refListMaxRows   = 6
)

// drawPanelReferenceList renders a referenceList control: a scrollable list of picked refs with
// per-row Remove via right-click menu, plus Add-from-selection and Clear buttons. Edits route
// through the session which emits panel.referencesChanged to the owning add-in.
func drawPanelReferenceList(s *app.Session, windowID string, control wire.PanelControlSpec) {
	panelFieldLabel(control.Text)
	if remove := drawRefRows(control.Rows); remove >= 0 {
		s.SetDockableWindowReferences(windowID, control.ID, refsWithout(control.Rows, remove))
	}
	if native.Button("Add from selection") {
		s.AddReferencesFromSelection(windowID, control.ID, control.Accepts)
	}
	native.SameLine()
	if native.Button("Clear") {
		s.SetDockableWindowReferences(windowID, control.ID, []string{})
	}
}

// drawRefRows draws the scrollable child region with one Selectable row per ref and a right-click
// Remove menu item; returns the index to remove, or -1. EndChild is always called per the BeginChild
// contract (see imgui.go:389).
func drawRefRows(rows []wire.PanelReferenceRow) int {
	height := float32(min(len(rows), refListMaxRows)*refListRowHeight + 8)
	remove := -1
	if native.BeginChild("##reflist", -1, height, true) {
		for i, row := range rows {
			native.PushIDInt(i)
			native.Selectable(refRowLabel(row), false)
			if native.BeginPopupContextItem("##refmenu") {
				if native.MenuItem("Remove") {
					remove = i
				}
				native.EndPopup()
			}
			native.PopID()
		}
	}
	native.EndChild()
	return remove
}

// refRowLabel returns the display label for a reference row: the add-in-supplied label when
// present, otherwise the raw ref key (e.g. "face/abc123").
func refRowLabel(row wire.PanelReferenceRow) string {
	if row.Label != "" {
		return row.Label
	}
	return row.Ref
}

// refsWithout returns the Ref strings from rows with the entry at index drop removed.
func refsWithout(rows []wire.PanelReferenceRow, drop int) []string {
	refs := make([]string, 0, len(rows))
	for i, row := range rows {
		if i != drop {
			refs = append(refs, row.Ref)
		}
	}
	return refs
}
