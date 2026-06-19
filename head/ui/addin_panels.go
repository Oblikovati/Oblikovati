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
		for i, control := range spec.Controls {
			drawAddInPanelControl(s, spec.ID, i, control)
		}
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

// drawAddInPanelControl renders one declared control by kind, pushing edits back to the owning
// add-in via Session.PanelValueChanged. The index joins the id stack so two controls never
// collide.
func drawAddInPanelControl(s *app.Session, windowID string, index int, control wire.PanelControlSpec) {
	native.PushIDInt(index)
	defer native.PopID()
	switch control.Kind {
	case types.PanelButton:
		if native.Button(control.Text) && control.CommandID != "" {
			if err := s.Execute(control.CommandID); err != nil {
				fmt.Fprintf(os.Stderr, "add-in window %q button %q: %v\n", windowID, control.CommandID, err)
			}
		}
	case types.PanelSeparator:
		native.Separator()
	case types.PanelTextBox, types.PanelValueEditor, types.PanelComboBox:
		buf := panelBuffer(windowID+"/"+control.ID, control.Value)
		if native.InputText(control.Text, buf) {
			s.PanelValueChanged(windowID, control.ID, bufString(buf))
		}
	case types.PanelCheckBox:
		checked := control.Value == "true"
		if native.Checkbox(control.Text, &checked) {
			s.PanelValueChanged(windowID, control.ID, strconv.FormatBool(checked))
		}
	case types.PanelDropdown:
		drawPanelDropdown(s, windowID, control)
	case types.PanelSlider:
		v, _ := strconv.ParseFloat(control.Value, 64)
		f := float32(v)
		if native.SliderFloat(control.Text, &f, float32(control.Min), float32(control.Max)) {
			s.PanelValueChanged(windowID, control.ID, strconv.FormatFloat(float64(f), 'g', -1, 64))
		}
	default: // PanelLabel (and any future kind degrades to its text)
		native.Text(control.Text)
	}
}

// drawPanelDropdown renders a single-select dropdown; picking an option pushes it to the add-in.
func drawPanelDropdown(s *app.Session, windowID string, control wire.PanelControlSpec) {
	if !native.BeginCombo(control.Text, control.Value) {
		return
	}
	for _, opt := range control.Options {
		if native.Selectable(opt, opt == control.Value) {
			s.PanelValueChanged(windowID, control.ID, opt)
		}
	}
	native.EndCombo()
}
