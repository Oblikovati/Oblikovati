// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/model/material"
)

// TestAppearancesFullLobeRoundTrip covers the full OpenPBR lobe set beyond
// TestAppearancesAndMaterialsListAndCreate's base-color-only coverage: list sees the
// built-in, create duplicates it into a project-scoped copy, update writes a lobe
// group (Coat) and the display name, and both survive the round trip.
func TestAppearancesFullLobeRoundTrip(t *testing.T) {
	r, s := seededSession(t)
	var apprs wire.ListAppearancesResult
	call(t, r, s, "appearances.list", "{}", &apprs)
	if len(apprs.Appearances) == 0 {
		t.Fatal("appearances.list returned none")
	}

	var made wire.AppearanceInfo
	call(t, r, s, "appearances.create",
		`{"baseId":"`+material.DefaultAppearanceID+`","name":"Shop Coat"}`, &made)
	if made.Source != "project" || made.DisplayName != "Shop Coat" {
		t.Fatalf("created appearance = %+v, want a project copy named Shop Coat", made)
	}

	made.DisplayName = "Shop Coat v2"
	made.Coat.Weight = 1
	made.Coat.Roughness = 0.4
	var updated wire.AppearanceInfo
	call(t, r, s, "appearances.update", mustJSON(t, wire.UpdateAppearanceArgs{
		ID: made.ID, DisplayName: made.DisplayName, Base: made.Base, Specular: made.Specular,
		Transmission: made.Transmission, Subsurface: made.Subsurface, Coat: made.Coat,
		Fuzz: made.Fuzz, ThinFilm: made.ThinFilm, Emission: made.Emission, Geometry: made.Geometry,
	}), &updated)
	if updated.DisplayName != "Shop Coat v2" || updated.Coat.Weight != 1 || updated.Coat.Roughness != 0.4 {
		t.Errorf("updated appearance = %+v, want Shop Coat v2 with coat weight 1 / roughness 0.4", updated)
	}

	var got wire.AppearanceInfo
	call(t, r, s, "appearances.get", `{"id":"`+made.ID+`"}`, &got)
	if got.DisplayName != "Shop Coat v2" || got.Coat.Weight != 1 {
		t.Errorf("get after update = %+v, want the persisted edit", got)
	}
}

// TestModelAssignAppearanceUnknownID covers model.assignAppearance's error path —
// TestMaterialGetUpdateAssign (material_coverage_test.go) already covers the success
// path via a real appearance id.
func TestModelAssignAppearanceUnknownID(t *testing.T) {
	r, s := seededSession(t)

	var res wire.OKResult
	call(t, r, s, "model.assignAppearance", mustJSON(t, wire.AssignAppearanceArgs{
		Scope: "part", AppearanceID: material.DefaultAppearanceID,
	}), &res)
	if !res.OK {
		t.Error("model.assignAppearance did not report OK")
	}

	if err := tryCall(t, r, s, "model.assignAppearance", mustJSON(t, wire.AssignAppearanceArgs{
		Scope: "part", AppearanceID: "nope",
	})); err == nil {
		t.Error("model.assignAppearance with an unknown id should error")
	}
}
