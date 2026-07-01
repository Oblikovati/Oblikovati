// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/event"
)

// Add-in dockable windows (M05-F03, #247): titled panels an add-in declares through
// dockableWindows.set; the head renders their declarative content and docks them.
// Like browser panes, the wire spec is the model (pure declared data).

// DockableWindowChanged fires (After) when a window's visibility changes — by the
// add-in, or by the user closing it in the head. The events relay forwards it as a
// wire dockableWindow.changed push event (DockableWindowsEvents OnShow/OnHide).
type DockableWindowChanged struct {
	ID      string
	Visible bool
}

// EventID implements event.Event.
func (DockableWindowChanged) EventID() event.TypeID { return tidDockableWinChanged }

// AddInDockableWindows stores the declared windows in creation order.
type AddInDockableWindows struct {
	order   []string
	windows map[string]wire.DockableWindowSpec
}

// NewAddInDockableWindows returns an empty window store.
func NewAddInDockableWindows() *AddInDockableWindows {
	return &AddInDockableWindows{windows: map[string]wire.DockableWindowSpec{}}
}

// List returns the windows in creation order.
func (w *AddInDockableWindows) List() []wire.DockableWindowSpec {
	out := make([]wire.DockableWindowSpec, len(w.order))
	for i, id := range w.order {
		out[i] = w.windows[id]
	}
	return out
}

// Get returns one window by id.
func (w *AddInDockableWindows) Get(id string) (wire.DockableWindowSpec, bool) {
	spec, ok := w.windows[id]
	return spec, ok
}

// DockableWindows returns the add-in dockable-window store.
func (s *Session) DockableWindows() *AddInDockableWindows { return s.dockableWindows }

// SetDockableWindow creates the window or replaces its title/content, emitting a
// visibility event when the effective visibility changed (a new visible window is a
// show; re-declaring with Visible:false hides).
func (s *Session) SetDockableWindow(spec wire.DockableWindowSpec) error {
	if spec.ID == "" || spec.Title == "" {
		return fmt.Errorf("app: dockable window needs id and title, got id=%q title=%q", spec.ID, spec.Title)
	}
	if err := validateControlTree(spec.ID, spec.Controls, 1); err != nil {
		return err
	}
	prev, existed := s.dockableWindows.windows[spec.ID]
	if !existed {
		s.dockableWindows.order = append(s.dockableWindows.order, spec.ID)
	}
	s.dockableWindows.windows[spec.ID] = spec
	if !existed && spec.Visible || existed && prev.Visible != spec.Visible {
		event.Emit(s.bus, event.After, DockableWindowChanged{ID: spec.ID, Visible: spec.Visible})
	}
	return nil
}

// SetDockableWindowVisible shows or hides a window without touching its content —
// also the path the head takes when the user closes the window, so the owning
// add-in always observes the change.
func (s *Session) SetDockableWindowVisible(id string, visible bool) error {
	spec, ok := s.dockableWindows.windows[id]
	if !ok {
		return fmt.Errorf("app: no dockable window %q", id)
	}
	if spec.Visible == visible {
		return nil
	}
	spec.Visible = visible
	s.dockableWindows.windows[id] = spec
	event.Emit(s.bus, event.After, DockableWindowChanged{ID: id, Visible: visible})
	return nil
}

// PanelValueChanged records the user's edit of an editable dockable-window control and
// notifies the owning add-in: it updates the stored control's Value (so the window state stays
// consistent if re-read) and emits PanelValueChanged on the bus, which the add-in event relay
// forwards to the add-in as wire.PanelValueChangedEvent. The head calls this when a control
// (text box, value editor, checkbox, dropdown, combo, slider) is edited.
func (s *Session) PanelValueChanged(windowID, controlID, value string) {
	if spec, ok := s.dockableWindows.windows[windowID]; ok {
		// Walk the whole tree: an edited control may be a grid cell several levels deep.
		if setControlValue(spec.Controls, controlID, value) {
			s.dockableWindows.windows[windowID] = spec
		}
	}
	event.Emit(s.bus, event.After, PanelValueChanged{WindowID: windowID, ControlID: controlID, Value: value})
}

// SetDockableWindowReferences replaces a referenceList control's rows with refs (one row per
// ref, Label left empty for host derivation) and notifies the owning add-in. The stored
// DockableWindowSpec is updated so a subsequent read reflects the new state. Mirrors
// PanelValueChanged. Called by the add-in router for dockableWindows.setReferences.
func (s *Session) SetDockableWindowReferences(windowID, controlID string, refs []string) {
	if spec, ok := s.dockableWindows.windows[windowID]; ok {
		if setControlRefs(spec.Controls, controlID, refs) {
			s.dockableWindows.windows[windowID] = spec
		}
	}
	event.Emit(s.bus, event.After, PanelReferencesChanged{
		WindowID: windowID, ControlID: controlID, Refs: refs, Action: "set",
	})
}

// DeleteDockableWindow removes a window, emitting a hide first when it was visible
// (the add-in sees a consistent shown→hidden→gone sequence).
func (s *Session) DeleteDockableWindow(id string) error {
	spec, ok := s.dockableWindows.windows[id]
	if !ok {
		return fmt.Errorf("app: no dockable window %q", id)
	}
	if spec.Visible {
		event.Emit(s.bus, event.After, DockableWindowChanged{ID: id, Visible: false})
	}
	delete(s.dockableWindows.windows, id)
	for i, x := range s.dockableWindows.order {
		if x == id {
			s.dockableWindows.order = append(s.dockableWindows.order[:i], s.dockableWindows.order[i+1:]...)
			break
		}
	}
	return nil
}
