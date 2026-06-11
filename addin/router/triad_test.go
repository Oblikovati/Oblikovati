// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

func TestTriadOverWire(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "triad.show", `{"triad":{"position":[1,2,3],"allowed":[1,9]}}`, nil)
	var spec wire.TriadSpec
	call(t, r, s, "triad.get", "{}", &spec)
	if !spec.Visible || spec.Position[2] != 3 || len(spec.Allowed) != 2 {
		t.Fatalf("triad = %+v, want visible at (1,2,3) with two allowed segments", spec)
	}
	if !s.TriadAllows(types.TriadXAxis) || s.TriadAllows(types.TriadYAxis) {
		t.Error("the allowed mask did not apply")
	}

	call(t, r, s, "triad.hide", "{}", nil)
	call(t, r, s, "triad.get", "{}", &spec)
	if spec.Visible {
		t.Error("triad still visible after hide")
	}
	if _, err := r.Handle(s, "triad.show", []byte(`{"triad":{"allowed":[42]}}`)); err == nil {
		t.Error("an out-of-range segment should fail over the wire")
	}
}

func TestManipulatorsOverWire(t *testing.T) {
	r, s := seededSession(t)
	call(t, r, s, "manipulators.set",
		`{"id":"sim","handles":[{"id":"tip","position":[0,0,5],"radiusPx":10}]}`, nil)
	if handles := s.Manipulators().Handles()["sim"]; len(handles) != 1 || handles[0].ID != "tip" {
		t.Fatalf("handles = %+v, want the tip", handles)
	}
	call(t, r, s, "manipulators.remove", `{"id":"sim"}`, nil)
	if len(s.Manipulators().Handles()) != 0 {
		t.Error("gizmo survived remove")
	}
	if _, err := r.Handle(s, "manipulators.set", []byte(`{"id":"","handles":[]}`)); err == nil {
		t.Error("a gizmo without an id should fail")
	}
}
