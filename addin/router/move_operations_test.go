// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// TestMoveBodyOperationListShiftsBody drives the full wire path for an M20-F20 move
// operation list — features.add(moveBody, operations:[freeDrag]) — and checks the body's
// range box shifts by the requested offsets, proving the opregistry args resolve into a
// composed, recomputed move (#654).
func TestMoveBodyOperationListShiftsBody(t *testing.T) {
	t.Parallel()
	r, s, _ := extrudedPartViaAPI(t)

	before := rangeBoxOf(t, r, s)
	call(t, r, s, "features.add",
		`{"kind":"moveBody","args":{"bodyIndex":0,"operations":[{"type":"freeDrag","x":"10 mm","z":"5 mm"}]}}`,
		&struct{}{})
	after := rangeBoxOf(t, r, s)

	if got := after.Min[0] - before.Min[0]; math.Abs(got-1) > 1e-6 { // 10 mm = 1 cm
		t.Errorf("range box min X moved by %g cm, want 1", got)
	}
	if got := after.Min[2] - before.Min[2]; math.Abs(got-0.5) > 1e-6 { // 5 mm = 0.5 cm
		t.Errorf("range box min Z moved by %g cm, want 0.5", got)
	}
	if got := after.Min[1] - before.Min[1]; math.Abs(got) > 1e-6 {
		t.Errorf("range box min Y moved by %g cm, want 0 (no Y offset)", got)
	}
}

// TestMoveBodyOperationUnknownTypeFails rejects an unknown operation type with a precise
// error rather than silently producing a no-op move.
func TestMoveBodyOperationUnknownTypeFails(t *testing.T) {
	t.Parallel()
	r, s, _ := extrudedPartViaAPI(t)
	if _, err := r.Handle(s, "features.add",
		[]byte(`{"kind":"moveBody","args":{"bodyIndex":0,"operations":[{"type":"warp"}]}}`)); err == nil {
		t.Error("expected an error for an unknown move operation type")
	}
}

// TestMoveBodyOperationScalarIsEditable proves a composed move exposes each operation's
// scalar through features.edit: editing the along-ray distance re-shifts the body (#654).
func TestMoveBodyOperationScalarIsEditable(t *testing.T) {
	t.Parallel()
	r, s, _ := extrudedPartViaAPI(t)
	call(t, r, s, "features.add",
		`{"kind":"moveBody","args":{"bodyIndex":0,"operations":[{"type":"alongRay","dir":[1,0,0],"dist":"10 mm"}]}}`,
		&struct{}{})
	tree := modelTreeOf(t, r, s)
	moveID := tree.Features[len(tree.Features)-1].ID

	var detail wire.FeatureDetailResult
	call(t, r, s, "features.get", mustJSON(t, wire.FeatureRefArgs{ID: moveID}), &detail)
	if len(detail.Feature.Scalars) != 1 || detail.Feature.Scalars[0].Label != "Move 1 Distance" {
		t.Fatalf("move scalars = %+v, want one 'Move 1 Distance'", detail.Feature.Scalars)
	}

	before := rangeBoxOf(t, r, s)
	call(t, r, s, "features.edit",
		mustJSON(t, wire.EditFeatureArgs{ID: moveID, Scalars: []wire.ScalarEdit{{Index: 0, Value: "30 mm"}}}),
		&wire.FeatureDetailResult{})
	after := rangeBoxOf(t, r, s)
	if got := after.Min[0] - before.Min[0]; math.Abs(got-2) > 1e-6 { // 30 mm − 10 mm = 2 cm
		t.Errorf("editing distance to 30 mm shifted X by %g cm, want 2", got)
	}
}

func rangeBoxOf(t *testing.T, r *Router, s *app.Session) wire.BodyRangeBoxResult {
	t.Helper()
	var box wire.BodyRangeBoxResult
	call(t, r, s, "body.rangeBox", `{"bodyIndex":0}`, &box)
	return box
}
