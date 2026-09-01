// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestMaterialGetUpdateAssign covers the material/appearance get, update, and assign
// handlers (the existing test only lists and creates).
func TestMaterialGetUpdateAssign(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)

	var mats wire.ListMaterialsResult
	call(t, r, s, "materials.list", "{}", &mats)
	if len(mats.Materials) == 0 {
		t.Fatal("materials.list returned none")
	}
	call(t, r, s, "materials.get", mustJSON(t, wire.AssetRefArgs{ID: mats.Materials[0].ID}), nil)
	upd := mats.Materials[0]
	upd.DisplayName += " (edited)"
	_ = tryCall(t, r, s, "materials.update", mustJSON(t, upd))

	var apprs wire.ListAppearancesResult
	call(t, r, s, "appearances.list", "{}", &apprs)
	if len(apprs.Appearances) == 0 {
		t.Fatal("appearances.list returned none")
	}
	aid := apprs.Appearances[0].ID
	call(t, r, s, "appearances.get", mustJSON(t, wire.AssetRefArgs{ID: aid}), nil)
	_ = tryCall(t, r, s, "model.assignAppearance", mustJSON(t, wire.AssignAppearanceArgs{Scope: "part", AppearanceID: aid}))

	// Unknown ids are clean errors.
	if err := tryCall(t, r, s, "materials.get", mustJSON(t, wire.AssetRefArgs{ID: "nope"})); err == nil {
		t.Error("materials.get on an unknown id should error")
	}
	if err := tryCall(t, r, s, "appearances.get", mustJSON(t, wire.AssetRefArgs{ID: "nope"})); err == nil {
		t.Error("appearances.get on an unknown id should error")
	}
}
