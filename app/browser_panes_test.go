// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/event"
)

func simPane() wire.BrowserPaneSpec {
	return wire.BrowserPaneSpec{
		ID: "sim", Title: "Simulation",
		Nodes: []wire.BrowserNodeSpec{{
			ID: "loads", Label: "Loads", Expanded: true,
			Children: []wire.BrowserNodeSpec{{ID: "f1", Label: "Force 10N"}},
		}},
	}
}

func TestBrowserPanesSetReplaceDeleteList(t *testing.T) {
	t.Parallel()
	panes := NewAddInBrowserPanes()
	if err := panes.Set(simPane()); err != nil {
		t.Fatalf("Set: %v", err)
	}
	replaced := simPane()
	replaced.Nodes = nil
	if err := panes.Set(replaced); err != nil {
		t.Fatalf("Set(replace): %v", err)
	}
	lst := panes.List()
	if len(lst) != 1 || len(lst[0].Nodes) != 0 {
		t.Fatalf("List = %+v, want one pane with the replaced (empty) tree", lst)
	}
	if err := panes.Delete("sim"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(panes.List()) != 0 {
		t.Error("pane survived Delete")
	}
	if err := panes.Delete("sim"); err == nil {
		t.Error("Delete(unknown) should fail")
	}
}

func TestBrowserPaneRejectsMissingIdentity(t *testing.T) {
	t.Parallel()
	if err := NewAddInBrowserPanes().Set(wire.BrowserPaneSpec{ID: "x"}); err == nil {
		t.Error("Set without title should fail")
	}
}

// TestActivateBrowserPaneNodeMenuEmitsItem checks a context-menu choice reaches the add-in as a
// "menu" gesture carrying the chosen item id.
func TestActivateBrowserPaneNodeMenuEmitsItem(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := s.BrowserPanes().Set(simPane()); err != nil {
		t.Fatalf("Set: %v", err)
	}
	var got []BrowserPaneNodeActivated
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e BrowserPaneNodeActivated) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})

	if err := s.ActivateBrowserPaneNodeMenu("sim", "f1", "edit"); err != nil {
		t.Fatalf("ActivateBrowserPaneNodeMenu: %v", err)
	}
	if len(got) != 1 || got[0].Gesture != BrowserGestureMenu || got[0].MenuItem != "edit" {
		t.Fatalf("events = %+v, want one menu gesture with item edit", got)
	}
	if err := s.ActivateBrowserPaneNodeMenu("ghost", "f1", "edit"); err == nil {
		t.Error("unknown pane should error")
	}
}

func TestActivateBrowserPaneNodeEmitsEvent(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := s.BrowserPanes().Set(simPane()); err != nil {
		t.Fatalf("Set: %v", err)
	}
	var got []BrowserPaneNodeActivated
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e BrowserPaneNodeActivated) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})

	if err := s.ActivateBrowserPaneNode("sim", "f1", BrowserGestureDouble); err != nil {
		t.Fatalf("ActivateBrowserPaneNode: %v", err)
	}
	if len(got) != 1 || got[0].Node != "f1" || got[0].Gesture != "double" {
		t.Fatalf("events = %+v, want one double-gesture on f1", got)
	}

	if err := s.ActivateBrowserPaneNode("ghost", "f1", BrowserGestureSelect); err == nil {
		t.Error("unknown pane should error")
	}
	if err := s.ActivateBrowserPaneNode("sim", "f1", "hover"); err == nil {
		t.Error("unknown gesture should error")
	}
}
