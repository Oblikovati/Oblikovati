// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"testing"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// lastFeatureID returns the id of the most recently added feature.
func lastFeatureID(t *testing.T, r *Router, s *app.Session) uint64 {
	t.Helper()
	tree := modelTreeOf(t, r, s)
	if len(tree.Features) == 0 {
		t.Fatal("no features in the part")
	}
	return tree.Features[len(tree.Features)-1].ID
}

// partVol measures the active part's first body volume (for before/after re-pick comparisons).
func partVol(t *testing.T, s *app.Session) float64 {
	t.Helper()
	def, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	return bodyVolume(def)
}

// addFilletOnEdge adds a single-edge constant-radius fillet (the simple edgeRefs+radius form,
// which drives def.EdgeKeys — the slot the re-pick mutates) and returns its feature id.
func addFilletOnEdge(t *testing.T, r *Router, s *app.Session, edge string) uint64 {
	t.Helper()
	args, err := json.Marshal(map[string]any{
		"kind": "fillet",
		"args": map[string]any{"edgeRefs": []string{edge}, "radius": "4 mm"},
	})
	if err != nil {
		t.Fatal(err)
	}
	call(t, r, s, "features.add", string(args), &struct {
		Bodies int `json:"bodies"`
	}{})
	return lastFeatureID(t, r, s)
}

// TestFeatureSlotsReported: features.get reports a feature's re-pickable reference slots (#163).
func TestFeatureSlotsReported(t *testing.T) {
	r, s, vertical := filletBoxFixture(t)
	id := addFilletOnEdge(t, r, s, vertical[0])

	var det wire.FeatureDetailResult
	call(t, r, s, "features.get", fmt.Sprintf(`{"id":%d}`, id), &det)
	if len(det.Feature.Slots) != 1 {
		t.Fatalf("fillet slots = %+v, want one", det.Feature.Slots)
	}
	sl := det.Feature.Slots[0]
	if sl.Kind != "edges" || !sl.Multi || sl.Count != 1 {
		t.Errorf("fillet slot = %+v, want kind=edges multi=true count=1", sl)
	}
}

// TestRepickFilletAddAndClearEdges drives the #163 acceptance: add an edge to a placed fillet by
// reference key (more geometry, slot count up), then clear it (back to the bare box).
func TestRepickFilletAddAndClearEdges(t *testing.T) {
	r, s, vertical := filletBoxFixture(t)
	id := addFilletOnEdge(t, r, s, vertical[0])
	faces1 := partBodyFaces(t, s) // box (6) + one rounded edge = 7

	// Add a second edge by reference key.
	var det wire.FeatureDetailResult
	addArgs, _ := json.Marshal(wire.EditFeatureArgs{ID: id, Repick: []wire.FeatureRepick{{Slot: 0, Ref: vertical[1]}}})
	call(t, r, s, "features.edit", string(addArgs), &det)
	if det.Feature.Slots[0].Count != 2 {
		t.Fatalf("after adding an edge, slot count = %d, want 2", det.Feature.Slots[0].Count)
	}
	if faces2 := partBodyFaces(t, s); faces2 <= faces1 {
		t.Errorf("adding a filleted edge gave %d faces, want > %d", faces2, faces1)
	}

	// Clear the slot → the fillet rounds nothing → back to the bare box (6 faces).
	clearArgs, _ := json.Marshal(wire.EditFeatureArgs{ID: id, Repick: []wire.FeatureRepick{{Slot: 0, Clear: true}}})
	call(t, r, s, "features.edit", string(clearArgs), &det)
	if det.Feature.Slots[0].Count != 0 {
		t.Errorf("after clear, slot count = %d, want 0", det.Feature.Slots[0].Count)
	}
	if faces := partBodyFaces(t, s); faces != 6 {
		t.Errorf("after clearing the fillet edges, %d faces, want 6 (bare box)", faces)
	}
}

// TestRepickExtrudeProfile drives the #163 acceptance: re-point an extrude at a different sketch
// profile and recompute to different geometry.
func TestRepickExtrudeProfile(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &struct{}{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","variant":"center","points":[[0,0]],"radius":"2 cm"}`, &struct{}{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","variant":"center","points":[[6,0]],"radius":"1 cm"}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"20 mm"}}`, &struct{}{})
	id := lastFeatureID(t, r, s)
	v0 := partVol(t, s)

	var det wire.FeatureDetailResult
	call(t, r, s, "features.get", fmt.Sprintf(`{"id":%d}`, id), &det)
	if len(det.Feature.Slots) != 1 || det.Feature.Slots[0].Kind != "profile" {
		t.Fatalf("extrude slots = %+v, want one profile slot", det.Feature.Slots)
	}

	args, _ := json.Marshal(wire.EditFeatureArgs{ID: id, Repick: []wire.FeatureRepick{{Slot: 0, SketchIndex: 0, ProfileIndex: 1}}})
	call(t, r, s, "features.edit", string(args), &det)
	if v1 := partVol(t, s); v1 == v0 {
		t.Errorf("re-picking the extrude profile did not change the volume (%.3f)", v1)
	}
}

// TestRepickInvalidProfileLeavesUntouched: an out-of-range profile index fails the edit and
// leaves the definition untouched (the #163 atomic-batch acceptance — nothing applies until the
// whole batch validates).
func TestRepickInvalidProfileLeavesUntouched(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &struct{}{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","variant":"center","points":[[0,0]],"radius":"2 cm"}`, &struct{}{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"20 mm"}}`, &struct{}{})
	id := lastFeatureID(t, r, s)
	v0 := partVol(t, s)

	bad, _ := json.Marshal(wire.EditFeatureArgs{ID: id, Repick: []wire.FeatureRepick{{Slot: 0, SketchIndex: 0, ProfileIndex: 9}}})
	if _, err := r.Handle(s, "features.edit", bad); err == nil {
		t.Fatal("features.edit with an out-of-range profile index should fail")
	}
	if v1 := partVol(t, s); v1 != v0 {
		t.Errorf("a failed re-pick changed the geometry (%.3f -> %.3f); want untouched", v0, v1)
	}
}

// TestRepickSlotOutOfRange: a slot index past the feature's slots is a clear rejection.
func TestRepickSlotOutOfRange(t *testing.T) {
	r, s, vertical := filletBoxFixture(t)
	id := addFilletOnEdge(t, r, s, vertical[0])
	bad, _ := json.Marshal(wire.EditFeatureArgs{ID: id, Repick: []wire.FeatureRepick{{Slot: 9, Ref: vertical[1]}}})
	if _, err := r.Handle(s, "features.edit", bad); err == nil {
		t.Error("features.edit with an out-of-range slot should fail")
	}
}
