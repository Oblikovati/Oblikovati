// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"testing"

	"oblikovati.org/api/wire"
)

// TestControlPointSplineKind (#150): a control-point spline is created by the first-class kind
// and enumerates distinctly from a fit spline.
func TestControlPointSplineKind(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"controlPointSpline","points":[[0,0],[2,1],[4,0]]}`, &wire.AddSketchEntityResult{})
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"spline","points":[[0,5],[2,6],[4,5]]}`, &wire.AddSketchEntityResult{})

	var ents wire.EnumerateEntitiesResult
	call(t, r, s, "sketch.entities", `{"sketchIndex":0}`, &ents)
	kinds := map[string]int{}
	for _, e := range ents.Entities {
		kinds[e.Kind]++
	}
	if kinds["controlPointSpline"] != 1 || kinds["spline"] != 1 {
		t.Errorf("spline kinds = %v, want one controlPointSpline and one spline", kinds)
	}
}

// TestSlotPlacementVariants (#149): the by-overall straight slot and the by-center-point arc
// slot (sweep angle via endAngle) create closed profiles over the wire.
func TestSlotPlacementVariants(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})

	var res wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"slot","variant":"overall","points":[[0,0],[8,0]],"width":"2 cm"}`, &res)
	if len(res.EntityIDs) != 4 {
		t.Errorf("by-overall slot = %d entities, want 4", len(res.EntityIDs))
	}

	var prof wire.ListProfilesResult
	call(t, r, s, "sketch.profiles", `{"sketchIndex":0}`, &prof)
	if len(prof.Profiles) < 1 {
		t.Errorf("by-overall slot profiles = %d, want >= 1", len(prof.Profiles))
	}

	// Arc slot by center point: center + start + a sweep angle in endAngle.
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"slot","variant":"arcCenterPoint","points":[[20,0],[25,0]],"width":"1 cm","endAngle":"90 deg"}`, &res)
	if len(res.EntityIDs) != 4 {
		t.Errorf("arc-center-point slot = %d entities, want 4", len(res.EntityIDs))
	}
}

// TestTangentDistanceDimension (#152): the distance from a line to a circle's near and far
// tangent point. With the line on y=0 and the circle centered at (0,5) r=2, near = 5-2 = 3,
// far = 5+2 = 7.
func TestTangentDistanceDimension(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	var line, circle wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,0]]}`, &line)
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"circle","variant":"center","points":[[0,5]],"radius":"2 cm"}`, &circle)

	near := mustJSON(t, wire.AddDimensionArgs{SketchIndex: 0, Kind: "tangentDistance", Entities: []uint64{line.EntityID, circle.EntityID}, Expression: "1 cm"})
	var dim wire.AddDimensionResult
	call(t, r, s, "sketch.addDimension", near, &dim)
	if dim.Kind != "tangentDistance" || math.Abs(dim.Value-3) > 1e-6 {
		t.Errorf("near tangent-distance = %+v, want kind tangentDistance value 3", dim)
	}

	far := mustJSON(t, wire.AddDimensionArgs{SketchIndex: 0, Kind: "tangentDistance", Entities: []uint64{line.EntityID, circle.EntityID}, Expression: "1 cm", FarSide: true})
	call(t, r, s, "sketch.addDimension", far, &dim)
	if math.Abs(dim.Value-7) > 1e-6 {
		t.Errorf("far tangent-distance value = %v, want 7", dim.Value)
	}
}

// TestTangentDistanceDimensionBadOperands: wrong operand count/kind is a clear rejection.
func TestTangentDistanceDimensionBadOperands(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	var line wire.AddSketchEntityResult
	call(t, r, s, "sketch.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0],[4,0]]}`, &line)
	// Only a line, no circle.
	args := mustJSON(t, wire.AddDimensionArgs{SketchIndex: 0, Kind: "tangentDistance", Entities: []uint64{line.EntityID}, Expression: "1 cm"})
	if _, err := r.Handle(s, "sketch.addDimension", []byte(args)); err == nil {
		t.Error("tangentDistance with a single operand should fail")
	}
}
