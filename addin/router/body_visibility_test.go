// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// boxBodySession builds a part with one extruded box body.
func boxBodySession(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"20 mm"}}`, &struct{}{})
	return r, s
}

// TestBodyListReportsNameAndVisibility: body.list now reports each body's display name, solid
// flag, and visibility (#158).
func TestBodyListReportsNameAndVisibility(t *testing.T) {
	t.Parallel()
	r, s := boxBodySession(t)
	var list wire.BodyListResult
	call(t, r, s, "body.list", `{}`, &list)
	if len(list.Bodies) != 1 {
		t.Fatalf("body.list = %d bodies, want 1", len(list.Bodies))
	}
	b := list.Bodies[0]
	if b.Name != "Solid1" || !b.Solid || !b.Visible {
		t.Errorf("body info = %+v, want Solid1 solid and visible", b)
	}
}

// TestBodySetVisibleHidesAndShows drives the #158 acceptance: hide a body, see it reflected,
// then show it again.
func TestBodySetVisibleHidesAndShows(t *testing.T) {
	t.Parallel()
	r, s := boxBodySession(t)

	var res wire.BodyInfoResult
	call(t, r, s, "body.setVisible", `{"bodyIndex":0,"visible":false}`, &res)
	if res.Body.Visible {
		t.Errorf("after hide, body.visible = true, want false")
	}
	var list wire.BodyListResult
	call(t, r, s, "body.list", `{}`, &list)
	if list.Bodies[0].Visible {
		t.Errorf("body.list after hide shows visible=true, want false")
	}

	call(t, r, s, "body.setVisible", `{"bodyIndex":0,"visible":true}`, &res)
	if !res.Body.Visible {
		t.Errorf("after show, body.visible = false, want true")
	}
}

// TestBodySetVisibleBadIndexFails: an out-of-range body index is a rejection.
func TestBodySetVisibleBadIndexFails(t *testing.T) {
	t.Parallel()
	r, s := boxBodySession(t)
	if _, err := r.Handle(s, "body.setVisible", []byte(`{"bodyIndex":9,"visible":false}`)); err == nil {
		t.Error("body.setVisible with an out-of-range index should fail")
	}
}
