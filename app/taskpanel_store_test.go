// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/event"
)

func TestShowListResolveTaskPanel(t *testing.T) {
	s := NewSession()
	if err := s.ShowTaskPanel(wire.TaskPanelSpec{ID: "fix", Title: "Fixed"}); err != nil {
		t.Fatalf("ShowTaskPanel: %v", err)
	}
	if l := s.TaskPanels().List(); len(l) != 1 || l[0].ID != "fix" {
		t.Fatalf("List = %+v, want one panel fix", l)
	}

	var captured []TaskPanelClosed
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e TaskPanelClosed) event.Outcome {
		captured = append(captured, e)
		return event.Continue()
	})

	if err := s.ResolveTaskPanel("fix", true); err != nil {
		t.Fatalf("ResolveTaskPanel: %v", err)
	}
	if len(captured) != 1 || captured[0].ID != "fix" || !captured[0].Accepted {
		t.Fatalf("closed event = %+v, want fix accepted=true", captured)
	}
	if l := s.TaskPanels().List(); len(l) != 0 {
		t.Fatalf("panel not removed after resolve: %+v", l)
	}
}

func TestShowTaskPanelRejectsEmptyID(t *testing.T) {
	s := NewSession()
	if err := s.ShowTaskPanel(wire.TaskPanelSpec{Title: "x"}); err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestCloseTaskPanelIsSilent(t *testing.T) {
	s := NewSession()
	_ = s.ShowTaskPanel(wire.TaskPanelSpec{ID: "p", Title: "P"})
	if err := s.CloseTaskPanel("p"); err != nil {
		t.Fatalf("CloseTaskPanel: %v", err)
	}
	if err := s.CloseTaskPanel("missing"); err == nil {
		t.Fatal("expected error closing unknown panel")
	}
}
