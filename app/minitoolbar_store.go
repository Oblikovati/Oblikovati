// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/event"
)

// In-canvas mini-toolbars (M05-F07, #614): floating toolbars an interactive command
// or add-in declares; the head renders them in the viewport and reports the user's
// edits back as events. Like the other declared UI surfaces, the wire spec is the
// model.

// MiniToolbarControlChanged fires (After) when the user edits one control; the
// events relay forwards it as a miniToolbar.changed push event.
type MiniToolbarControlChanged struct {
	Toolbar  string
	Control  string
	Value    string
	Checked  bool
	Number   float64
	Selected int
}

// EventID implements event.Event.
func (MiniToolbarControlChanged) EventID() event.TypeID { return tidMiniToolbarChanged }

// MiniToolbarCommitted fires (After) on the toolbar's OK/Apply/Cancel.
type MiniToolbarCommitted struct {
	Toolbar string
	Gesture string // "ok", "apply" or "cancel"
}

// EventID implements event.Event.
func (MiniToolbarCommitted) EventID() event.TypeID { return tidMiniToolbarCommitted }

// The commit gestures of a mini-toolbar.
const (
	MiniToolbarOK     = "ok"
	MiniToolbarApply  = "apply"
	MiniToolbarCancel = "cancel"
)

// MiniToolbarRack holds the declared toolbars in creation order.
type MiniToolbarRack struct {
	order    []string
	toolbars map[string]wire.MiniToolbarSpec
}

// NewMiniToolbarRack returns an empty rack.
func NewMiniToolbarRack() *MiniToolbarRack {
	return &MiniToolbarRack{toolbars: map[string]wire.MiniToolbarSpec{}}
}

// List returns the toolbars in creation order.
func (r *MiniToolbarRack) List() []wire.MiniToolbarSpec {
	out := make([]wire.MiniToolbarSpec, len(r.order))
	for i, id := range r.order {
		out[i] = r.toolbars[id]
	}
	return out
}

// Get returns one toolbar by id.
func (r *MiniToolbarRack) Get(id string) (wire.MiniToolbarSpec, bool) {
	spec, ok := r.toolbars[id]
	return spec, ok
}

// MiniToolbars returns the session's mini-toolbar rack.
func (s *Session) MiniToolbars() *MiniToolbarRack { return s.miniToolbars }

// SetMiniToolbar creates or wholly replaces a toolbar.
func (s *Session) SetMiniToolbar(spec wire.MiniToolbarSpec) error {
	if spec.ID == "" {
		return fmt.Errorf("app: mini-toolbar needs an id, got %+v", spec)
	}
	if _, exists := s.miniToolbars.toolbars[spec.ID]; !exists {
		s.miniToolbars.order = append(s.miniToolbars.order, spec.ID)
	}
	s.miniToolbars.toolbars[spec.ID] = spec
	return nil
}

// UpdateMiniToolbarControls merges control values by control id.
func (s *Session) UpdateMiniToolbarControls(id string, controls []wire.MiniToolbarControlSpec) error {
	spec, ok := s.miniToolbars.toolbars[id]
	if !ok {
		return fmt.Errorf(errNoMiniToolbar, id)
	}
	for _, incoming := range controls {
		if !mergeMiniToolbarControl(&spec, incoming) {
			return fmt.Errorf("app: mini-toolbar %q has no control %q", id, incoming.ID)
		}
	}
	s.miniToolbars.toolbars[id] = spec
	return nil
}

// mergeMiniToolbarControl writes incoming's value fields over the matching control.
func mergeMiniToolbarControl(spec *wire.MiniToolbarSpec, incoming wire.MiniToolbarControlSpec) bool {
	for i := range spec.Controls {
		if spec.Controls[i].ID != incoming.ID {
			continue
		}
		c := &spec.Controls[i]
		c.Value, c.Checked, c.Number, c.Selected = incoming.Value, incoming.Checked, incoming.Number, incoming.Selected
		return true
	}
	return false
}

// RemoveMiniToolbar dismisses a toolbar.
func (s *Session) RemoveMiniToolbar(id string) error {
	if _, ok := s.miniToolbars.toolbars[id]; !ok {
		return fmt.Errorf(errNoMiniToolbar, id)
	}
	delete(s.miniToolbars.toolbars, id)
	for i, x := range s.miniToolbars.order {
		if x == id {
			s.miniToolbars.order = append(s.miniToolbars.order[:i], s.miniToolbars.order[i+1:]...)
			break
		}
	}
	return nil
}

// ChangeMiniToolbarControl records a user edit (the head's widgets call it),
// keeping the stored spec current and emitting the change for the owner.
func (s *Session) ChangeMiniToolbarControl(toolbarID string, control wire.MiniToolbarControlSpec) error {
	if err := s.UpdateMiniToolbarControls(toolbarID, []wire.MiniToolbarControlSpec{control}); err != nil {
		return err
	}
	event.Emit(s.bus, event.After, MiniToolbarControlChanged{
		Toolbar: toolbarID, Control: control.ID,
		Value: control.Value, Checked: control.Checked, Number: control.Number, Selected: control.Selected,
	})
	return nil
}

// CommitMiniToolbar reports the toolbar's OK/Apply/Cancel; ok and cancel also
// dismiss it (apply keeps it for further edits).
func (s *Session) CommitMiniToolbar(id, gesture string) error {
	if _, ok := s.miniToolbars.toolbars[id]; !ok {
		return fmt.Errorf(errNoMiniToolbar, id)
	}
	switch gesture {
	case MiniToolbarOK, MiniToolbarApply, MiniToolbarCancel:
	default:
		return fmt.Errorf("app: unknown mini-toolbar gesture %q (ok/apply/cancel)", gesture)
	}
	event.Emit(s.bus, event.After, MiniToolbarCommitted{Toolbar: id, Gesture: gesture})
	if gesture != MiniToolbarApply {
		return s.RemoveMiniToolbar(id)
	}
	return nil
}

// dropCommandMiniToolbars removes the toolbars tied to a command's lifetime — the
// interaction-graphics lifecycle: the tool ended, its toolbar goes with it.
func (s *Session) dropCommandMiniToolbars() {
	for _, id := range append([]string(nil), s.miniToolbars.order...) {
		if s.miniToolbars.toolbars[id].Command != "" {
			_ = s.RemoveMiniToolbar(id)
		}
	}
}
