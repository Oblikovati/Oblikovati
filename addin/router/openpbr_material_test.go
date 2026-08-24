// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/model/material"
)

// TestOpenPBRAppearancesListAndCreate mirrors TestAppearancesAndMaterialsListAndCreate for
// the full OpenPBR lobe set (M45-F02 #2126, ADR-0053): list sees the built-in, create
// duplicates it into a project-scoped copy, update writes a lobe group and the display
// name, and both survive the round trip.
func TestOpenPBRAppearancesListAndCreate(t *testing.T) {
	r, s := seededSession(t)
	var apprs wire.ListOpenPBRAppearancesResult
	call(t, r, s, "openpbrAppearances.list", "{}", &apprs)
	if len(apprs.Appearances) == 0 {
		t.Fatal("openpbrAppearances.list returned none")
	}

	var made wire.OpenPBRAppearanceInfo
	call(t, r, s, "openpbrAppearances.create",
		`{"baseId":"`+material.DefaultOpenPBRAppearanceID+`","name":"Shop Coat"}`, &made)
	if made.Source != "project" || made.DisplayName != "Shop Coat" {
		t.Fatalf("created openpbr appearance = %+v, want a project copy named Shop Coat", made)
	}

	made.DisplayName = "Shop Coat v2"
	made.Coat.Weight = 1
	made.Coat.Roughness = 0.4
	var updated wire.OpenPBRAppearanceInfo
	call(t, r, s, "openpbrAppearances.update", mustJSON(t, wire.UpdateOpenPBRAppearanceArgs{
		ID: made.ID, DisplayName: made.DisplayName, Base: made.Base, Specular: made.Specular,
		Transmission: made.Transmission, Subsurface: made.Subsurface, Coat: made.Coat,
		Fuzz: made.Fuzz, ThinFilm: made.ThinFilm, Emission: made.Emission, Geometry: made.Geometry,
	}), &updated)
	if updated.DisplayName != "Shop Coat v2" || updated.Coat.Weight != 1 || updated.Coat.Roughness != 0.4 {
		t.Errorf("updated openpbr appearance = %+v, want Shop Coat v2 with coat weight 1 / roughness 0.4", updated)
	}

	var got wire.OpenPBRAppearanceInfo
	call(t, r, s, "openpbrAppearances.get", `{"id":"`+made.ID+`"}`, &got)
	if got.DisplayName != "Shop Coat v2" || got.Coat.Weight != 1 {
		t.Errorf("get after update = %+v, want the persisted edit", got)
	}
}

// TestModelAssignOpenPBRAppearance covers the model.assignOpenPBRAppearance handler
// (M45-F05 PBI-350, ADR-0053) — mirrors material_coverage_test.go's
// model.assignAppearance coverage for the separate OpenPBRAppearance chain.
func TestModelAssignOpenPBRAppearance(t *testing.T) {
	r, s := seededSession(t)

	var res wire.OKResult
	call(t, r, s, "model.assignOpenPBRAppearance", mustJSON(t, wire.AssignOpenPBRAppearanceArgs{
		Scope: "part", AppearanceID: material.DefaultOpenPBRAppearanceID,
	}), &res)
	if !res.OK {
		t.Error("model.assignOpenPBRAppearance did not report OK")
	}

	if err := tryCall(t, r, s, "model.assignOpenPBRAppearance", mustJSON(t, wire.AssignOpenPBRAppearanceArgs{
		Scope: "part", AppearanceID: "nope",
	})); err == nil {
		t.Error("model.assignOpenPBRAppearance with an unknown id should error")
	}
}
