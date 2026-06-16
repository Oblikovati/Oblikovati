// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestRepresentationLifecycle exercises the representation handlers the end-to-end test
// does not: activate / list / delete for each rep kind, plus the appearance/section/
// flexible setters and modelStates.delete.
func TestRepresentationLifecycle(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	a, b := occs[0], occs[1]

	// --- design views: capture -> activate -> list -> appearance/section -> delete ---
	var dv wire.DesignViewResult
	call(t, r, s, "designReps.capture", mustJSON(t, wire.CaptureRepArgs{Name: "v1"}), &dv)
	call(t, r, s, "designReps.activate", mustJSON(t, wire.RepRef{ID: dv.Representation.ID}), &wire.DesignViewResult{})
	var dvl wire.DesignViewsResult
	call(t, r, s, "designReps.list", `{}`, &dvl)
	if len(dvl.Representations) == 0 {
		t.Error("designReps.list returned none after a capture")
	}
	_ = tryCall(t, r, s, "designReps.setAppearance", mustJSON(t, wire.SetAppearanceArgs{Rep: dv.Representation.ID, Occurrence: a.ID(), AppearanceID: "Steel"}))
	_ = tryCall(t, r, s, "designReps.addSection", mustJSON(t, wire.AddSectionArgs{
		Rep: dv.Representation.ID, Plane: types.SectionPlane{Normal: types.Vector{X: 0, Y: 0, Z: 1}},
	}))
	call(t, r, s, "designReps.delete", mustJSON(t, wire.RepRef{ID: dv.Representation.ID}), nil)

	// --- positional: capture -> list -> setFlexible -> delete ---
	var p wire.PositionalResult
	call(t, r, s, "positionalReps.capture", mustJSON(t, wire.CaptureRepArgs{Name: "p1"}), &p)
	var pl wire.PositionalsResult
	call(t, r, s, "positionalReps.list", `{}`, &pl)
	if len(pl.Representations) == 0 {
		t.Error("positionalReps.list returned none after a capture")
	}
	_ = tryCall(t, r, s, "positionalReps.setFlexible", mustJSON(t, wire.SetFlexibleArgs{Rep: p.Representation.ID, Occurrence: b.ID(), Flexible: true}))
	call(t, r, s, "positionalReps.delete", mustJSON(t, wire.RepRef{ID: p.Representation.ID}), nil)

	// --- LOD: capture -> activate -> list -> delete ---
	var lod wire.LODResult
	call(t, r, s, "lodReps.capture", mustJSON(t, wire.CaptureRepArgs{Name: "l1"}), &lod)
	call(t, r, s, "lodReps.activate", mustJSON(t, wire.RepRef{ID: lod.Representation.ID}), &wire.LODResult{})
	var ll wire.LODsResult
	call(t, r, s, "lodReps.list", `{}`, &ll)
	if len(ll.Representations) == 0 {
		t.Error("lodReps.list returned none after a capture")
	}
	call(t, r, s, "lodReps.delete", mustJSON(t, wire.RepRef{ID: lod.Representation.ID}), nil)

	// --- model states: create -> delete ---
	var ms wire.ModelStateResult
	call(t, r, s, "modelStates.create", mustJSON(t, wire.CreateModelStateArgs{Name: "st"}), &ms)
	call(t, r, s, "modelStates.delete", mustJSON(t, wire.RepRef{ID: ms.ModelState.ID}), nil)
}

// TestRepresentationErrors covers the no-assembly and bad-id error paths.
func TestRepresentationErrors(t *testing.T) {
	r, s := seededSession(t) // a PART, not an assembly
	for _, m := range []string{"designReps.capture", "positionalReps.capture", "lodReps.capture"} {
		if err := tryCall(t, r, s, m, `{"name":"x"}`); err == nil {
			t.Errorf("%s on a non-assembly should error", m)
		}
	}
	// Bad id against a real assembly.
	ra, sa, _, _ := assemblySessionWithBoxes(t, 0, 5)
	if err := tryCall(t, ra, sa, "lodReps.activate", `{"id":99999}`); err == nil {
		t.Error("activating an unknown LOD rep should error")
	}
}
