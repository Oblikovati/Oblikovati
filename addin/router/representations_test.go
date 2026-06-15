// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestRepresentationsOverWire drives the M12-F04 surface end to end: capture an LOD
// representation, edit its suppression override, switch a model state that selects it, and
// confirm the occurrence's suppression follows — over the router (#361/#367).
func TestRepresentationsOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	a, b := occs[0], occs[1]

	// Capture an LOD rep, then mark b suppressed within it.
	var lod wire.LODResult
	call(t, r, s, "lodReps.capture", mustJSON(t, wire.CaptureRepArgs{Name: "simplified"}), &lod)
	var lod2 wire.LODResult
	call(t, r, s, "lodReps.setSuppressed", mustJSON(t, wire.SetSuppressedArgs{Rep: lod.Representation.ID, Occurrence: b.ID(), Suppressed: true}), &lod2)
	if lod2.Representation.SuppressedCount != 1 {
		t.Fatalf("LOD suppressed count = %d, want 1", lod2.Representation.SuppressedCount)
	}

	// Capture a design-view rep that hides a (after making it hidden live).
	var dv wire.DesignViewResult
	call(t, r, s, "designReps.capture", mustJSON(t, wire.CaptureRepArgs{Name: "view"}), &dv)
	call(t, r, s, "designReps.setVisibility", mustJSON(t, wire.SetVisibilityArgs{Rep: dv.Representation.ID, Occurrence: a.ID(), Visible: false}), &wire.DesignViewResult{})

	// A model state selecting both, activated, applies suppression + visibility together.
	var ms wire.ModelStateResult
	call(t, r, s, "modelStates.create", mustJSON(t, wire.CreateModelStateArgs{
		Name: "state", DesignView: dv.Representation.Name, LevelOfDetail: lod.Representation.Name,
	}), &ms)
	call(t, r, s, "modelStates.activate", mustJSON(t, wire.RepRef{ID: ms.ModelState.ID}), &wire.ModelStateResult{})

	if !b.Suppressed() {
		t.Error("activating the model state did not suppress b (LOD not applied)")
	}
	if a.Visible() {
		t.Error("activating the model state did not hide a (design-view not applied)")
	}

	var list wire.ModelStatesResult
	call(t, r, s, "modelStates.list", `{}`, &list)
	if len(list.ModelStates) != 1 || !list.ModelStates[0].Active {
		t.Errorf("model state list = %+v, want one active", list.ModelStates)
	}
}

// TestPositionalRepOverWire captures a positional rep, overrides a constraint value, activates
// it, and confirms the re-solve moves the component.
func TestPositionalRepOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	occs[0].SetGrounded(true)
	topKey := topBoxFaceKey(t, occs[0])
	botKey := bottomBoxFaceKey(t, occs[1])

	var added wire.ConstraintResult
	call(t, r, s, "assemblyConstraints.addMate", mustJSON(t, wire.AddMateArgs{
		A: wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: topKey},
		B: wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: botKey},
	}), &added)

	var p wire.PositionalResult
	call(t, r, s, "positionalReps.capture", mustJSON(t, wire.CaptureRepArgs{Name: "open"}), &p)
	var p2 wire.PositionalResult
	call(t, r, s, "positionalReps.setOverride", mustJSON(t, wire.SetPositionalOverrideArgs{
		Rep: p.Representation.ID, Relationship: added.Constraint.ID, Value: 4,
	}), &p2)
	if p2.Representation.OverrideCount != 1 {
		t.Fatalf("positional override count = %d, want 1", p2.Representation.OverrideCount)
	}
	call(t, r, s, "positionalReps.activate", mustJSON(t, wire.RepRef{ID: p.Representation.ID}), &wire.PositionalResult{})
	// The override re-solved the assembly; the free box is no longer face-coincident.
	if z := occs[1].Transform().Translation().Z; z == 0 {
		t.Errorf("activating the positional override did not re-solve (z still 0)")
	}
}
