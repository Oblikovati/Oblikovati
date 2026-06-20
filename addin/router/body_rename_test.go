// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
)

// TestBodyRenameSetsAndReverts drives the #1078 rename acceptance: rename a body, see it
// reflected in body.list, then clear it (empty name) to revert to the "Solid{N}" default.
func TestBodyRenameSetsAndReverts(t *testing.T) {
	r, s := boxBodySession(t)

	var res wire.BodyInfoResult
	call(t, r, s, "body.rename", `{"bodyIndex":0,"name":"Housing"}`, &res)
	if res.Body.Name != "Housing" {
		t.Errorf("after rename, body.name = %q, want Housing", res.Body.Name)
	}

	var list wire.BodyListResult
	call(t, r, s, "body.list", `{}`, &list)
	if list.Bodies[0].Name != "Housing" {
		t.Errorf("body.list after rename = %q, want Housing", list.Bodies[0].Name)
	}

	call(t, r, s, "body.rename", `{"bodyIndex":0,"name":""}`, &res)
	if res.Body.Name != "Solid1" {
		t.Errorf("after clearing the name, body.name = %q, want the Solid1 default", res.Body.Name)
	}
}

// TestBodyRenameSurvivesRecompute: a stored body name is keyed by the persistent reference key,
// so it survives a part recompute (the key is stable across rebuilds).
func TestBodyRenameSurvivesRecompute(t *testing.T) {
	r, s := boxBodySession(t)
	call(t, r, s, "body.rename", `{"bodyIndex":0,"name":"Bracket"}`, &wire.BodyInfoResult{})

	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	part.Recompute()

	var list wire.BodyListResult
	call(t, r, s, "body.list", `{}`, &list)
	if list.Bodies[0].Name != "Bracket" {
		t.Errorf("body name after recompute = %q, want Bracket (the name is refkey-anchored)", list.Bodies[0].Name)
	}
}

// TestBodyRenameBadIndexFails: an out-of-range body index is a rejection.
func TestBodyRenameBadIndexFails(t *testing.T) {
	r, s := boxBodySession(t)
	if _, err := r.Handle(s, "body.rename", []byte(`{"bodyIndex":9,"name":"X"}`)); err == nil {
		t.Error("body.rename with an out-of-range index should fail")
	}
}
