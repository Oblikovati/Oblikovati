// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

func TestUISearchOverWire(t *testing.T) {
	r, s := seededSession(t)
	var res wire.SearchCommandsResult
	call(t, r, s, "ui.search", `{"query":"extrude"}`, &res)
	if len(res.Commands) == 0 {
		t.Fatal("ui.search(extrude) found nothing — the standard Extrude command should match")
	}
	call(t, r, s, "ui.search", `{"query":""}`, &res)
	if len(res.Commands) != 0 {
		t.Error("a blank query should return nothing")
	}
}

func TestUIMarkingMenuOverWire(t *testing.T) {
	r, s := seededSession(t)
	var menu wire.MarkingMenuView
	call(t, r, s, "ui.getMarkingMenu", `{"environment":0}`, &menu)
	if len(menu.Quadrants) == 0 {
		t.Fatal("the base default radial menu should not be empty")
	}

	call(t, r, s, "ui.setMarkingMenu",
		`{"menu":{"environment":1,"quadrants":[{"quadrant":0,"commandId":"Sketch.Line"}],"overflow":["Sketch.Finish"]}}`, nil)
	call(t, r, s, "ui.getMarkingMenu", `{"environment":1}`, &menu)
	if len(menu.Quadrants) != 1 || menu.Quadrants[0].CommandID != "Sketch.Line" {
		t.Fatalf("customized menu = %+v, want the single Line slot", menu)
	}
	if _, err := r.Handle(s, "ui.setMarkingMenu",
		[]byte(`{"menu":{"quadrants":[{"quadrant":11,"commandId":"X"}]}}`)); err == nil {
		t.Error("an out-of-range quadrant should fail over the wire")
	}
}

func TestUIContextMenuAndVisibilityOverWire(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "ui.setContextMenu",
		`{"addin":"com.x.sim","kind":"feature","items":[{"label":"Analyze","commandId":"Sim.Analyze"}]}`, nil)
	if _, err := r.Handle(s, "ui.setContextMenu",
		[]byte(`{"kind":"feature","items":[]}`)); err == nil {
		t.Error("an injection without the owning add-in should fail")
	}

	call(t, r, s, "ui.setObjectVisibility",
		`{"visibility":{"workPlanes":false,"workAxes":true,"workPoints":true,"sketches":true}}`, nil)
	var vis wire.ObjectVisibilityView
	call(t, r, s, "ui.getObjectVisibility", "{}", &vis)
	if vis.WorkPlanes || !vis.WorkAxes {
		t.Fatalf("visibility = %+v, want planes hidden, axes shown", vis)
	}
	if got := len(s.PickableWorkPlanes()); got != 0 {
		t.Errorf("hidden planes still pickable (%d)", got)
	}
}
