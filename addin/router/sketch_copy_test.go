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

// TestSketchCopyToCarriesConstraintsAndDimensions (#1083): a rectangle is fully constrained
// (coincidences/perpendiculars + width/height dimensions); copying the whole sketch carries
// every relation whose operands are all in the copied set onto the target, with fresh
// parameter names, and the target rectangle stays solvable.
func TestSketchCopyToCarriesConstraintsAndDimensions(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var line1, line2, circle wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,0]]}`, &line1)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,2],[4,2]]}`, &line2)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","variant":"center","points":[[10,0]],"radius":"1 cm"}`, &circle)

	parallel := mustJSON(t, wire.AddConstraintArgs{SketchIndex: 0, Kind: "parallel", Entities: []uint64{line1.EntityID, line2.EntityID}})
	call(t, r, s, "sketch.addConstraint", parallel, &wire.AddConstraintResult{})
	radius := mustJSON(t, wire.AddDimensionArgs{SketchIndex: 0, Kind: "radius", Entities: []uint64{circle.EntityID}, Expression: "1 cm"})
	call(t, r, s, "sketch.addDimension", radius, &wire.AddDimensionResult{})

	call(t, r, s, "sketch.create", `{"plane":"XZ"}`, &wire.CreateSketchResult{}) // sketch 1, empty

	var srcCons wire.ListConstraintsResult
	call(t, r, s, "sketch.constraints", `{"sketchIndex":0}`, &srcCons)
	var srcDims wire.ListDimensionsResult
	call(t, r, s, "sketch.dimensions", `{"sketchIndex":0}`, &srcDims)

	call(t, r, s, "sketch.copyTo", `{"sourceIndex":0,"targetIndex":1,"position":[100,0]}`, &wire.CopySketchResult{})

	var dstCons wire.ListConstraintsResult
	call(t, r, s, "sketch.constraints", `{"sketchIndex":1}`, &dstCons)
	if len(dstCons.Constraints) != len(srcCons.Constraints) || !hasConstraintKind(dstCons.Constraints, "parallel") {
		t.Errorf("carried constraints = %+v, want %d including a parallel (every operand is in the copied set)", dstCons.Constraints, len(srcCons.Constraints))
	}
	var dstDims wire.ListDimensionsResult
	call(t, r, s, "sketch.dimensions", `{"sketchIndex":1}`, &dstDims)
	if len(dstDims.Dimensions) != 1 || dstDims.Dimensions[0].Kind != "radius" {
		t.Errorf("carried dimensions = %+v, want one radius", dstDims.Dimensions)
	}
	// The copied dimension's parameter is freshly minted in the shared part store — no collision.
	if len(srcDims.Dimensions) == 1 && len(dstDims.Dimensions) == 1 && dstDims.Dimensions[0].Name == srcDims.Dimensions[0].Name {
		t.Errorf("copied dimension reused source parameter name %q; want a fresh one", srcDims.Dimensions[0].Name)
	}
}

// TestSketchCopyToDropsExternalConstraint (#1083): a parallel constraint relates two lines;
// copying only one of them leaves its partner behind, so the constraint is silently dropped.
func TestSketchCopyToDropsExternalConstraint(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	var line1, line2 wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,1]]}`, &line1)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,2],[4,3]]}`, &line2)
	parallel := mustJSON(t, wire.AddConstraintArgs{SketchIndex: 0, Kind: "parallel", Entities: []uint64{line1.EntityID, line2.EntityID}})
	call(t, r, s, "sketch.addConstraint", parallel, &wire.AddConstraintResult{})
	call(t, r, s, "sketch.create", `{"plane":"XZ"}`, &wire.CreateSketchResult{})

	args := mustJSON(t, wire.CopySketchArgs{SourceIndex: 0, TargetIndex: 1, EntityIDs: []uint64{line1.EntityID}})
	call(t, r, s, "sketch.copyTo", args, &wire.CopySketchResult{})

	var dstCons wire.ListConstraintsResult
	call(t, r, s, "sketch.constraints", `{"sketchIndex":1}`, &dstCons)
	if hasConstraintKind(dstCons.Constraints, "parallel") {
		t.Errorf("carried constraints = %+v, want the parallel dropped (its partner line was not copied)", dstCons.Constraints)
	}
}

// hasConstraintKind reports whether any enumerated constraint has the given kind.
func hasConstraintKind(cons []wire.ConstraintInfo, kind string) bool {
	for _, c := range cons {
		if c.Kind == kind {
			return true
		}
	}
	return false
}

// TestSketchCopyToSameSketchFails (#151): copying a sketch onto itself is rejected.
func TestSketchCopyToSameSketchFails(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	if _, err := r.Handle(s, "sketch.copyTo", []byte(`{"sourceIndex":0,"targetIndex":0}`)); err == nil {
		t.Error("copyTo onto the same sketch should fail")
	}
}
