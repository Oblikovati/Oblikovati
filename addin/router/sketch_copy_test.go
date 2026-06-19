// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestSketchCopyToCopiesGeometry (#151): copying a sketch's contents to another sketch
// re-instantiates its geometry and the profile closes in the target.
func TestSketchCopyToCopiesGeometry(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "sketch.create", `{"plane":"XZ"}`, &wire.CreateSketchResult{}) // sketch 1, empty

	var srcEnts wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &srcEnts)

	var res wire.CopySketchResult
	call(t, r, s, "sketch.copyTo", `{"sourceIndex":0,"targetIndex":1}`, &res)
	if res.Count != len(srcEnts.Entities) || res.Count == 0 {
		t.Errorf("copyTo created %d entities, want %d (every source entity)", res.Count, len(srcEnts.Entities))
	}

	var prof wire.ListProfilesResult
	call(t, r, s, "sketch.profiles", `{"sketchIndex":1}`, &prof)
	if len(prof.Profiles) != 1 {
		t.Errorf("target sketch profiles = %d, want 1 (the copied rectangle closes)", len(prof.Profiles))
	}
	// The source is untouched.
	call(t, r, s, "sketch.profiles", `{"sketchIndex":0}`, &prof)
	if len(prof.Profiles) != 1 {
		t.Errorf("source sketch profiles = %d after copy, want 1 (unchanged)", len(prof.Profiles))
	}
}

// TestSketchCopyToOffsetAndSubset (#151): a placement offset and an entity-id subset.
func TestSketchCopyToOffsetAndSubset(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	var line wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,0]]}`, &line)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","variant":"center","points":[[10,0]],"radius":"1 cm"}`, &wire.AddSketchEntityResult{})
	call(t, r, s, "sketch.create", `{"plane":"XZ"}`, &wire.CreateSketchResult{})

	// Copy only the line, offset by (0,5).
	args := mustJSON(t, wire.CopySketchArgs{SourceIndex: 0, TargetIndex: 1, EntityIDs: []uint64{line.EntityID}, Position: []float64{0, 5}})
	var res wire.CopySketchResult
	call(t, r, s, "sketch.copyTo", args, &res)
	if res.Count != 1 {
		t.Errorf("subset copy created = %d, want 1 (just the line)", res.Count)
	}

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":1}`, &ents)
	if len(ents.Entities) != 1 || ents.Entities[0].Kind != "line" {
		t.Errorf("target entities = %+v, want one line", ents.Entities)
	}
}

// TestSketchCopyToSameSketchFails (#151): copying a sketch onto itself is rejected.
func TestSketchCopyToSameSketchFails(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	if _, err := r.Handle(s, "sketch.copyTo", []byte(`{"sourceIndex":0,"targetIndex":0}`)); err == nil {
		t.Error("copyTo onto the same sketch should fail")
	}
}
