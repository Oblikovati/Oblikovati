// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// M07 router coverage (Oblikovati/Oblikovati#293/#629/#630): the body
// topology/query/facet surface over a 40×30×50 mm extruded box.

// boxPartSession is the canonical box part (4×3×5 cm) used across the suite.
func boxPartSession(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &wire.CreateSketchResult{})
	call(t, r, s, "sketch.rectangle", `{"sketchIndex":0,"width":"40 mm","height":"30 mm"}`, &wire.SketchRectangleResult{})
	call(t, r, s, "features.add", `{"kind":"extrude","args":{"sketchIndex":0,"profileIndex":0,"distance":"50 mm"}}`, &struct{}{})
	return r, s
}

// TestBodyListAndShells: enumeration reports the box and its single closed,
// non-void shell with keys.
func TestBodyListAndShells(t *testing.T) {
	r, s := boxPartSession(t)
	var list wire.BodyListResult
	call(t, r, s, "body.list", `{}`, &list)
	if len(list.Bodies) != 1 || list.Bodies[0].Faces != 6 || !list.Bodies[0].Solid || list.Bodies[0].Shells != 1 {
		t.Fatalf("body list = %+v, want one solid 6-face single-shell box", list.Bodies)
	}
	var shells wire.BodyShellsResult
	call(t, r, s, "body.shells", `{"bodyIndex":0}`, &shells)
	if len(shells.Shells) != 1 {
		t.Fatalf("shells = %d, want 1", len(shells.Shells))
	}
	sh := shells.Shells[0]
	if !sh.Closed || sh.Void || sh.Key == "" || sh.TransientKey == 0 {
		t.Errorf("shell = %+v, want closed non-void with keys", sh)
	}
	if stdmath.Abs(sh.Volume-60) > 0.01 { // 4×3×5 cm
		t.Errorf("shell volume = %g, want 60", sh.Volume)
	}
}

// TestBodyMinimumDistance: a transient travel polyline measured against the box — clearance above
// the top face, tool-radius widening, and a piercing move clamped to 0; plus the malformed-input
// guard. Distances are database units (cm); the box top is at z=5.
func TestBodyMinimumDistance(t *testing.T) {
	r, s := boxPartSession(t)
	var md wire.MinimumDistanceResult

	call(t, r, s, "body.minimumDistance", `{"bodyIndex":0,"points":[0,1.5,10,4,1.5,10]}`, &md)
	if stdmath.Abs(md.Distance-5) > 1e-6 {
		t.Errorf("above-box distance = %g cm, want 5", md.Distance)
	}
	// A 1 cm tool radius shrinks the clearance to 4 cm.
	call(t, r, s, "body.minimumDistance", `{"bodyIndex":0,"points":[0,1.5,10,4,1.5,10],"radius":1}`, &md)
	if stdmath.Abs(md.Distance-4) > 1e-6 {
		t.Errorf("radius-widened distance = %g cm, want 4", md.Distance)
	}
	// A move through the block clamps at 0.
	call(t, r, s, "body.minimumDistance", `{"bodyIndex":0,"points":[2,1.5,2,2,1.5,3]}`, &md)
	if md.Distance != 0 {
		t.Errorf("through-box distance = %g, want 0", md.Distance)
	}
	// A points list that is not a multiple of three is rejected, not truncated.
	if err := tryCall(t, r, s, "body.minimumDistance", `{"bodyIndex":0,"points":[0,1.5]}`); err == nil {
		t.Error("malformed probe points should error")
	}
}

// TestBodyPointAndRayQueries: locate, ray, containment.
func TestBodyPointAndRayQueries(t *testing.T) {
	r, s := boxPartSession(t)
	var loc wire.LocateUsingPointResult
	call(t, r, s, "body.locateUsingPoint", `{"bodyIndex":0,"point":[2,1.5,5.05],"entityKind":"face","proximityTolerance":0.1}`, &loc)
	if !loc.Found || loc.Entity.Kind != "face" {
		t.Fatalf("locate = %+v, want the top face", loc)
	}
	var ray wire.FindUsingRayResult
	call(t, r, s, "body.findUsingRay", `{"bodyIndex":0,"origin":[2,1.5,-5],"direction":[0,0,1]}`, &ray)
	if len(ray.Hits) != 2 || ray.Hits[0].Distance >= ray.Hits[1].Distance {
		t.Fatalf("ray hits = %+v, want entry+exit sorted", ray.Hits)
	}
	var inside wire.IsPointInsideResult
	call(t, r, s, "body.isPointInside", `{"bodyIndex":0,"point":[2,1.5,2.5]}`, &inside)
	if inside.Containment != "inside" {
		t.Errorf("center containment = %q, want inside", inside.Containment)
	}
	call(t, r, s, "body.isPointInside", `{"bodyIndex":0,"point":[20,0,0]}`, &inside)
	if inside.Containment != "outside" {
		t.Errorf("far containment = %q, want outside", inside.Containment)
	}
}

// TestBodyConvexityValidateRangeBoxBind: the remaining query handlers.
func TestBodyConvexityValidateRangeBoxBind(t *testing.T) {
	r, s := boxPartSession(t)
	var conv wire.ConvexityEdgesResult
	call(t, r, s, "body.convexityEdges", `{"bodyIndex":0,"collection":"allConvex"}`, &conv)
	if len(conv.Edges) != 12 {
		t.Errorf("convex edges = %d, want 12", len(conv.Edges))
	}
	var valid wire.ValidateBodyResult
	call(t, r, s, "body.validate", `{"bodyIndex":0,"checkLevel":2}`, &valid)
	if !valid.Valid || len(valid.Problems) != 0 {
		t.Errorf("validate = %+v, want clean", valid)
	}
	var box wire.BodyRangeBoxResult
	call(t, r, s, "body.rangeBox", `{"bodyIndex":0,"precise":true}`, &box)
	if len(box.Min) != 3 || len(box.Max) != 3 {
		t.Fatalf("precise box = %+v", box)
	}
	var obb wire.BodyRangeBoxResult
	call(t, r, s, "body.rangeBox", `{"bodyIndex":0,"oriented":true}`, &obb)
	vol := vecLen(obb.DirectionOne) * vecLen(obb.DirectionTwo) * vecLen(obb.DirectionThree)
	if stdmath.Abs(vol-60) > 0.01 {
		t.Errorf("OBB volume = %g, want 60", vol)
	}
	var shells wire.BodyShellsResult
	call(t, r, s, "body.shells", `{"bodyIndex":0}`, &shells)
	var bind wire.BindTransientKeyResult
	call(t, r, s, "body.bindTransientKey",
		fmt.Sprintf(`{"bodyIndex":0,"transientKey":%d}`, shells.Shells[0].TransientKey), &bind)
	if !bind.Found || bind.Kind != "shell" {
		t.Errorf("bind = %+v, want the shell back", bind)
	}
}

func vecLen(v []float64) float64 {
	return stdmath.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

// TestFacetAndStrokeCacheFlow: calculate caches, existing retrieves, the
// tolerance lists track, and the face-level calls work by key.
func TestFacetAndStrokeCacheFlow(t *testing.T) {
	r, s := boxPartSession(t)
	var fs wire.FacetSetResult
	call(t, r, s, "body.calculateFacets", `{"bodyIndex":0,"tolerance":0.01,"includeTextureMap":true}`, &fs)
	if fs.FacetCount != 12 || len(fs.IndexCountPerFace) != 6 || len(fs.TextureCoordinates) != 2*fs.VertexCount {
		t.Fatalf("facets = %d tris, %d faces, %d uv floats", fs.FacetCount, len(fs.IndexCountPerFace), len(fs.TextureCoordinates))
	}
	var existing wire.FacetSetResult
	call(t, r, s, "body.existingFacets", `{"bodyIndex":0,"tolerance":0.01}`, &existing)
	if existing.FacetCount != 12 {
		t.Errorf("existing facets = %d tris, want 12", existing.FacetCount)
	}
	if _, err := r.Handle(s, "body.existingFacets", []byte(`{"bodyIndex":0,"tolerance":0.5}`)); err == nil {
		t.Error("existingFacets at an uncached tolerance must error")
	}
	var tols wire.FacetTolerancesResult
	call(t, r, s, "body.facetTolerances", `{"bodyIndex":0}`, &tols)
	if len(tols.Tolerances) != 1 || tols.Tolerances[0] != 0.01 {
		t.Errorf("facet tolerances = %v, want [0.01]", tols.Tolerances)
	}
	var ss wire.StrokeSetResult
	call(t, r, s, "body.calculateStrokes", `{"bodyIndex":0,"tolerance":0.01}`, &ss)
	if ss.PolylineCount != 12 {
		t.Errorf("strokes = %d polylines, want 12 edges", ss.PolylineCount)
	}
	call(t, r, s, "body.existingStrokes", `{"bodyIndex":0,"tolerance":0.01}`, &ss)
	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	// Reference keys carry a raw kind byte — marshal the args as real JSON.
	faceArgs, err := json.Marshal(wire.FaceFacetsArgs{BodyIndex: 0, FaceKey: keys.Bodies[0].Faces[0].Key, Tolerance: 0.01})
	if err != nil {
		t.Fatal(err)
	}
	var faceFs wire.FacetSetResult
	call(t, r, s, "face.calculateFacets", string(faceArgs), &faceFs)
	if faceFs.FacetCount != 2 {
		t.Errorf("face facets = %d tris, want 2 (a planar quad)", faceFs.FacetCount)
	}
	var faceSs wire.StrokeSetResult
	call(t, r, s, "face.calculateStrokes", string(faceArgs), &faceSs)
	if faceSs.PolylineCount != 4 {
		t.Errorf("face strokes = %d polylines, want 4", faceSs.PolylineCount)
	}
}

// TestBodyValidateFlagsBadIndex: errors carry the offending value.
func TestBodyValidateFlagsBadIndex(t *testing.T) {
	r, s := boxPartSession(t)
	_, err := r.Handle(s, "body.shells", []byte(`{"bodyIndex":5}`))
	if err == nil || !strings.Contains(err.Error(), "5") {
		t.Errorf("bad index error = %v, want it to name index 5", err)
	}
}
