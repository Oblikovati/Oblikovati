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
	if err := NewSession().SetDockableWindow(wire.DockableWindowSpec{Title: "x"}); err == nil {
		t.Error("Set without id should fail")
	}
}
