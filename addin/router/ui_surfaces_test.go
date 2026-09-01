// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

func TestBrowserPanesOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "browser.setPane",
		`{"pane":{"id":"sim","title":"Simulation","nodes":[{"id":"loads","label":"Loads","children":[{"id":"f1","label":"Force 10N"}]}]}}`, nil)

	var lst wire.ListBrowserPanesResult
	call(t, r, s, "browser.listPanes", "{}", &lst)
	if len(lst.Panes) != 1 || lst.Panes[0].Nodes[0].Children[0].ID != "f1" {
		t.Fatalf("listPanes = %+v, want the declared tree intact", lst.Panes)
	}

	call(t, r, s, "browser.deletePane", `{"id":"sim"}`, nil)
	call(t, r, s, "browser.listPanes", "{}", &lst)
	if len(lst.Panes) != 0 {
		t.Fatalf("panes after delete = %+v, want none", lst.Panes)
	}
	if _, err := r.Handle(s, "browser.deletePane", []byte(`{"id":"sim"}`)); err == nil {
		t.Error("deleting an unknown pane should fail")
	}
}

func TestDockableWindowsOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "dockableWindows.set",
		`{"window":{"id":"sim.panel","title":"Simulation","dock":2,"visible":true,"controls":[{"kind":1,"text":"Run","commandId":"Sim.Run"}]}}`, nil)

	var lst wire.ListDockableWindowsResult
	call(t, r, s, "dockableWindows.list", "{}", &lst)
	if len(lst.Windows) != 1 || lst.Windows[0].Dock != types.DockRight || !lst.Windows[0].Visible {
		t.Fatalf("list = %+v, want one visible dock-right window", lst.Windows)
	}
	if lst.Windows[0].Controls[0].Kind != types.PanelButton {
		t.Errorf("control kind = %v, want button", lst.Windows[0].Controls[0].Kind)
	}

	call(t, r, s, "dockableWindows.setVisible", `{"id":"sim.panel","visible":false}`, nil)
	call(t, r, s, "dockableWindows.list", "{}", &lst)
	if lst.Windows[0].Visible {
		t.Error("window still visible after setVisible(false)")
	}

	call(t, r, s, "dockableWindows.delete", `{"id":"sim.panel"}`, nil)
	call(t, r, s, "dockableWindows.list", "{}", &lst)
	if len(lst.Windows) != 0 {
		t.Fatalf("windows after delete = %+v, want none", lst.Windows)
	}
}

// TestDockableWindowSetValueOverWire checks setValue drives an editable control to a new value (as a
// user edit would), updating the stored window so a re-read reflects it.
func TestDockableWindowSetValueOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "dockableWindows.set",
		`{"window":{"id":"sim.panel","title":"Sim","dock":2,"visible":true,"controls":[{"kind":6,"id":"sim_view","value":"Material","options":["Material","Path"]}]}}`, nil)

	call(t, r, s, "dockableWindows.setValue", `{"windowId":"sim.panel","controlId":"sim_view","value":"Path"}`, nil)

	var lst wire.ListDockableWindowsResult
	call(t, r, s, "dockableWindows.list", "{}", &lst)
	if len(lst.Windows) != 1 || lst.Windows[0].Controls[0].Value != "Path" {
		t.Errorf("control value = %+v, want Path after setValue", lst.Windows)
	}
}

func TestListEnvironmentsOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	var res wire.ListEnvironmentsResult
	call(t, r, s, "ui.listEnvironments", "{}", &res)
	if len(res.Environments) != 2 {
		t.Fatalf("environments = %+v, want Base and Sketch", res.Environments)
	}
	if !res.Environments[0].Active || res.Environments[0].Name != "Base" {
		t.Errorf("first = %+v, want active Base (no sketch open)", res.Environments[0])
	}
}

// TestRibbonListCarriesTypedControlModel checks ribbon.list serves the M05-F03 typed
// model: control kind/style/state, popup items, and the selector of a combo panel.
func TestRibbonListCarriesTypedControlModel(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	call(t, r, s, "commands.create",
		`{"id":"x.a","displayName":"Alpha","category":"Demo"}`, nil)
	call(t, r, s, "commands.create",
		`{"id":"x.menu","displayName":"Menu","category":"Demo","kind":4,"items":["x.a"]}`, nil)

	var res wire.ListRibbonResult
	call(t, r, s, "ribbon.list", "{}", &res)

	menu := findControl(t, res, "x.menu")
	if menu.Kind != types.PopupControl {
		t.Errorf("menu kind = %v, want popup", menu.Kind)
	}
	if len(menu.Items) != 1 || menu.Items[0].CommandID != "x.a" || menu.Items[0].Label != "Alpha" {
		t.Errorf("menu items = %+v, want the resolved x.a entry", menu.Items)
	}

	if !hasSelectorPanel(res) {
		t.Error("no panel serialized as a selector — the Visual Style combo panel should")
	}
}

// findControl locates a control by command id across the ribbon.
func findControl(t *testing.T, res wire.ListRibbonResult, id string) wire.RibbonControlInfo {
	t.Helper()
	for _, tab := range res.Tabs {
		for _, p := range tab.Panels {
			for _, c := range p.Controls {
				if c.CommandID == id {
					return c
				}
			}
		}
	}
	t.Fatalf("control %q not found in ribbon", id)
	return wire.RibbonControlInfo{}
}

// hasSelectorPanel reports whether any panel rendered as a selection box.
func hasSelectorPanel(res wire.ListRibbonResult) bool {
	for _, tab := range res.Tabs {
		for _, p := range tab.Panels {
			if p.Selector != nil && len(p.Selector.Options) > 0 {
				return true
			}
		}
	}
	return false
}

// TestDockableWindowSetReferencesOverWire checks setReferences drives a referenceList control to a
// new row set and the stored window reflects it on a subsequent list.
func TestDockableWindowSetReferencesOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	// kind 12 = types.PanelReferenceList
	call(t, r, s, "dockableWindows.set",
		`{"window":{"id":"w","title":"T","dock":2,"visible":true,"controls":[{"kind":12,"id":"faces","text":"Faces"}]}}`, nil)

	call(t, r, s, "dockableWindows.setReferences",
		`{"windowId":"w","controlId":"faces","refs":["face/a"]}`, nil)

	var lst wire.ListDockableWindowsResult
	call(t, r, s, "dockableWindows.list", "{}", &lst)
	if len(lst.Windows) != 1 || len(lst.Windows[0].Controls) != 1 || len(lst.Windows[0].Controls[0].Rows) != 1 {
		t.Fatalf("after setReferences: windows=%+v, want one window with one control with one row", lst.Windows)
	}
	if got := lst.Windows[0].Controls[0].Rows[0].Ref; got != "face/a" {
		t.Errorf("Rows[0].Ref = %q, want face/a", got)
	}
}

// TestTaskPanelShowCloseOverWire checks taskPanel.show stores a modal panel in the session,
// and taskPanel.close removes it.
func TestTaskPanelShowCloseOverWire(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)

	call(t, r, s, "taskPanel.show",
		`{"panel":{"id":"fix","title":"Fixed"}}`, nil)
	if got := s.TaskPanels().List(); len(got) != 1 {
		t.Fatalf("after show: %d panels, want 1", len(got))
	}

	call(t, r, s, "taskPanel.close", `{"id":"fix"}`, nil)
	if got := s.TaskPanels().List(); len(got) != 0 {
		t.Fatalf("after close: %d panels, want 0", len(got))
	}
}
