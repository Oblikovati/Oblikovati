// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/event"
)

func simWindow(visible bool) wire.DockableWindowSpec {
	return wire.DockableWindowSpec{
		ID: "sim.panel", Title: "Simulation", Dock: types.DockRight, Visible: visible,
		Controls: []wire.PanelControlSpec{
			{Kind: types.PanelLabel, Text: "Mesh: 12k"},
			{Kind: types.PanelButton, Text: "Run", CommandID: "Sim.Run"},
		},
	}
}

// dockEventLog subscribes to visibility events and records them.
func dockEventLog(s *Session) *[]DockableWindowChanged {
	var got []DockableWindowChanged
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e DockableWindowChanged) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})
	return &got
}

func TestSetDockableWindowEmitsOnVisibilityTransitions(t *testing.T) {
	t.Parallel()
	s := NewSession()
	got := dockEventLog(s)

	if err := s.SetDockableWindow(simWindow(true)); err != nil { // new visible ⇒ shown
		t.Fatalf("SetDockableWindow: %v", err)
	}
	if err := s.SetDockableWindow(simWindow(true)); err != nil { // same visibility ⇒ silent
		t.Fatalf("SetDockableWindow(re-set): %v", err)
	}
	if err := s.SetDockableWindow(simWindow(false)); err != nil { // hide
		t.Fatalf("SetDockableWindow(hide): %v", err)
	}
	want := []DockableWindowChanged{{ID: "sim.panel", Visible: true}, {ID: "sim.panel", Visible: false}}
	if len(*got) != 2 || (*got)[0] != want[0] || (*got)[1] != want[1] {
		t.Fatalf("events = %+v, want shown then hidden", *got)
	}
}

func TestSetDockableWindowVisibleRoundTrip(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := s.SetDockableWindow(simWindow(false)); err != nil {
		t.Fatalf("SetDockableWindow: %v", err)
	}
	got := dockEventLog(s)

	if err := s.SetDockableWindowVisible("sim.panel", true); err != nil {
		t.Fatalf("SetVisible(true): %v", err)
	}
	if err := s.SetDockableWindowVisible("sim.panel", true); err != nil { // no-op
		t.Fatalf("SetVisible(repeat): %v", err)
	}
	spec, ok := s.DockableWindows().Get("sim.panel")
	if !ok || !spec.Visible {
		t.Fatalf("Get = (%+v, %v), want visible window", spec, ok)
	}
	if len(*got) != 1 || !(*got)[0].Visible {
		t.Fatalf("events = %+v, want exactly one show", *got)
	}
	if err := s.SetDockableWindowVisible("ghost", true); err == nil {
		t.Error("SetVisible(unknown) should fail")
	}
}

func TestDeleteDockableWindowEmitsHideFirst(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := s.SetDockableWindow(simWindow(true)); err != nil {
		t.Fatalf("SetDockableWindow: %v", err)
	}
	got := dockEventLog(s)

	if err := s.DeleteDockableWindow("sim.panel"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(*got) != 1 || (*got)[0].Visible {
		t.Fatalf("events = %+v, want one hide before removal", *got)
	}
	if len(s.DockableWindows().List()) != 0 {
		t.Error("window survived Delete")
	}
	if err := s.DeleteDockableWindow("sim.panel"); err == nil {
		t.Error("Delete(unknown) should fail")
	}
}

func TestSetDockableWindowRejectsMissingIdentity(t *testing.T) {
	t.Parallel()
	if err := NewSession().SetDockableWindow(wire.DockableWindowSpec{Title: "x"}); err == nil {
		t.Error("Set without id should fail")
	}
}

// TestPanelValueChangedUpdatesControlAndEmits covers the editable-panel edit path: the stored
// control's value is updated and a PanelValueChanged event is emitted for the add-in.
func TestPanelValueChangedUpdatesControlAndEmits(t *testing.T) {
	t.Parallel()
	s := NewSession()
	w := simWindow(true)
	w.Controls = append(w.Controls, wire.PanelControlSpec{Kind: types.PanelTextBox, ID: "poles", Value: "10"})
	if err := s.SetDockableWindow(w); err != nil {
		t.Fatalf("SetDockableWindow: %v", err)
	}
	var got []PanelValueChanged
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e PanelValueChanged) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})

	s.PanelValueChanged("sim.panel", "poles", "12")

	spec, _ := s.DockableWindows().Get("sim.panel")
	var poles string
	for _, c := range spec.Controls {
		if c.ID == "poles" {
			poles = c.Value
		}
	}
	if poles != "12" {
		t.Errorf("stored control value = %q, want 12", poles)
	}
	if len(got) != 1 || got[0].ControlID != "poles" || got[0].Value != "12" {
		t.Errorf("emitted = %+v, want one poles=12", got)
	}
}

// TestGraphicsLabelsEmptyWithoutGraphics exercises GraphicsLabels (no add-in graphics → none).
func TestGraphicsLabelsEmptyWithoutGraphics(t *testing.T) {
	t.Parallel()
	if got := NewSession().GraphicsLabels(); len(got) != 0 {
		t.Errorf("labels without graphics = %d, want 0", len(got))
	}
}
