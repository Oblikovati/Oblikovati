// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// M40-G1 typed-router coverage (Oblikovati/Oblikovati refactor/m40-g1-typed-router):
// the previously-untested body facet/stroke, wire, face-evaluate, containment and
// transient-key handlers, driven against the canonical 4×3×5 cm extruded box
// (boxPartSession, defined in body_m07_test.go).

// TestBodcovStrokeCacheAndTolerances exercises body.calculateStrokes → body.existingStrokes →
// body.strokeTolerances (the last was 0% before this): the box's 12 edges stroke into 12
// polylines, the cache serves them back, and the tolerance list reports the one cached set.
func TestBodcovStrokeCacheAndTolerances(t *testing.T) {
	t.Parallel()
	r, s := boxPartSession(t)
	var calc wire.StrokeSetResult
	call(t, r, s, "body.calculateStrokes", `{"bodyIndex":0,"tolerance":0.02}`, &calc)
	if calc.PolylineCount != 12 || calc.VertexCount == 0 {
		t.Fatalf("strokes = %d polylines, %d verts, want 12 non-empty", calc.PolylineCount, calc.VertexCount)
	}
	var existing wire.StrokeSetResult
	call(t, r, s, "body.existingStrokes", `{"bodyIndex":0,"tolerance":0.02}`, &existing)
	if existing.PolylineCount != 12 {
		t.Errorf("existing strokes = %d, want 12", existing.PolylineCount)
	}
	var tols wire.FacetTolerancesResult
	call(t, r, s, "body.strokeTolerances", `{"bodyIndex":0}`, &tols)
	if len(tols.Tolerances) != 1 || tols.Tolerances[0] != 0.02 {
		t.Errorf("stroke tolerances = %v, want [0.02]", tols.Tolerances)
	}
}

// TestBodcovBodyWires drives body.wires (0% before this): a solid box has no free wires, and an
// out-of-range body index is named in the error rather than panicking.
func TestBodcovBodyWires(t *testing.T) {
	t.Parallel()
	r, s := boxPartSession(t)
	var res wire.BodyWiresResult
	call(t, r, s, "body.wires", `{"bodyIndex":0}`, &res)
	if len(res.Wires) != 0 {
		t.Errorf("solid box wires = %d, want 0 (a closed solid has no free wires)", len(res.Wires))
	}
	if err := tryCall(t, r, s, "body.wires", `{"bodyIndex":7}`); err == nil {
		t.Error("body.wires on out-of-range index 7 should error")
	}
}

// TestBodcovFaceEvaluateModes runs body.faceEvaluate through all four surface-evaluation modes so
// each branch of evaluateFaceSurface executes; every mode returns one sampled point and the
// [uMin,vMin,uMax,vMax] param range.
func TestBodcovFaceEvaluateModes(t *testing.T) {
	t.Parallel()
	r, s := boxPartSession(t)
	key := bodcovFirstFaceKey(t, r, s)
	for _, mode := range []string{
		wire.FaceEvalPointAtParam, wire.FaceEvalNormalAtParam,
		wire.FaceEvalTangents, wire.FaceEvalClosestPoint,
	} {
		var res wire.FaceEvaluateResult
		call(t, r, s, "body.faceEvaluate", mustJSON(t, wire.FaceEvaluateArgs{
			BodyIndex: 0, FaceKey: key, Mode: mode, Inputs: bodcovEvalInputs(mode)}), &res)
		if len(res.Points) != 3 || len(res.ParamRange) != 4 {
			t.Fatalf("%s: points=%v range=%v, want one point + 4-cell range", mode, res.Points, res.ParamRange)
		}
	}
}

// TestBodcovPointContainmentShell covers the shell-scoped branch of pointContainmentOf: a point on
// the top face reports "on" when tested against shell 0, and an out-of-range shell index errors.
func TestBodcovPointContainmentShell(t *testing.T) {
	t.Parallel()
	r, s := boxPartSession(t)
	var on wire.IsPointInsideResult
	call(t, r, s, "body.isPointInside",
		`{"bodyIndex":0,"point":[2,1.5,5],"shellIndex":0,"onTolerance":0.001}`, &on)
	if on.Containment != "on" {
		t.Errorf("on-surface via shell 0 = %q, want on", on.Containment)
	}
	if err := tryCall(t, r, s, "body.isPointInside",
		`{"bodyIndex":0,"point":[0,0,0],"shellIndex":9}`); err == nil {
		t.Error("out-of-range shell index 9 should error")
	}
}

// TestBodcovBindEntityKinds locates a face, an edge and a vertex on the box, then binds each
// transient key back — exercising the vertex/edge/face arms of transientRefReferenceKey.
func TestBodcovBindEntityKinds(t *testing.T) {
	t.Parallel()
	r, s := boxPartSession(t)
	for _, tc := range bodcovLocateCases() {
		var loc wire.LocateUsingPointResult
		call(t, r, s, "body.locateUsingPoint", mustJSON(t, wire.LocateUsingPointArgs{
			BodyIndex: 0, Point: tc.point, EntityKind: tc.kind, ProximityTolerance: tc.tol}), &loc)
		if !loc.Found {
			t.Fatalf("locate %s at %v not found", tc.kind, tc.point)
		}
		var bind wire.BindTransientKeyResult
		call(t, r, s, "body.bindTransientKey",
			fmt.Sprintf(`{"bodyIndex":0,"transientKey":%d}`, loc.Entity.TransientKey), &bind)
		if !bind.Found || bind.Kind != tc.kind || bind.Key == "" {
			t.Errorf("bind %s = %+v, want found %s with a key", tc.kind, bind, tc.kind)
		}
	}
}

// bodcovLocateCase names a probe point and the entity kind it should resolve to on the box.
type bodcovLocateCase struct {
	kind  string
	point []float64
	tol   float64
}

// bodcovLocateCases picks a point near the top face, a bottom edge and a corner of the 4×3×5 box.
func bodcovLocateCases() []bodcovLocateCase {
	return []bodcovLocateCase{
		{"face", []float64{2, 1.5, 5.05}, 0.1},
		{"edge", []float64{2, 0.05, 0.05}, 0.3},
		{"vertex", []float64{0.1, 0.1, 0.1}, 0.4},
	}
}

// bodcovFirstFaceKey returns the persistent reference key of the box's first face.
func bodcovFirstFaceKey(t *testing.T, r *Router, s *app.Session) string {
	t.Helper()
	var keys wire.ReferenceKeysResult
	call(t, r, s, "model.referenceKeys", `{}`, &keys)
	if len(keys.Bodies) == 0 || len(keys.Bodies[0].Faces) == 0 {
		t.Fatal("model.referenceKeys returned no face keys")
	}
	return keys.Bodies[0].Faces[0].Key
}

// bodcovEvalInputs supplies the flat input array for a face-evaluate mode: an (x,y,z) point for
// closestPoint, a single (u,v) parameter pair for the parametric modes.
func bodcovEvalInputs(mode string) []float64 {
	if mode == wire.FaceEvalClosestPoint {
		return []float64{2, 1.5, 5}
	}
	return []float64{0.5, 0.5}
}
