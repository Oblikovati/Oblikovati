// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

func TestMiniToolbarOverWire(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "miniToolbar.set",
		`{"toolbar":{"id":"sim.probe","visible":true,"headsUpText":"Probe","anchor":[1,2,3],"showOK":true,"controls":[{"kind":5,"id":"depth","label":"Depth","value":"10 mm"}]}}`, nil)

	var lst wire.ListMiniToolbarsResult
	call(t, r, s, "miniToolbar.list", "{}", &lst)
	if len(lst.Toolbars) != 1 || lst.Toolbars[0].Anchor == nil || lst.Toolbars[0].Anchor[2] != 3 {
		t.Fatalf("list = %+v, want the anchored toolbar", lst.Toolbars)
	}

	call(t, r, s, "miniToolbar.update",
		`{"id":"sim.probe","controls":[{"id":"depth","value":"12 mm"}]}`, nil)
	call(t, r, s, "miniToolbar.list", "{}", &lst)
	if lst.Toolbars[0].Controls[0].Value != "12 mm" {
		t.Fatalf("update did not merge: %+v", lst.Toolbars[0].Controls)
	}

	call(t, r, s, "miniToolbar.remove", `{"id":"sim.probe"}`, nil)
	call(t, r, s, "miniToolbar.list", "{}", &lst)
	if len(lst.Toolbars) != 0 {
		t.Fatalf("toolbars after remove = %+v, want none", lst.Toolbars)
	}
	if _, err := r.Handle(s, "miniToolbar.update",
		[]byte(`{"id":"sim.probe","controls":[]}`)); err == nil {
		t.Error("updating a removed toolbar should fail")
	}
}
