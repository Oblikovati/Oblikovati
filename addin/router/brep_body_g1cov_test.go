// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// M40-G1 typed-router coverage (Oblikovati/Oblikovati refactor/m40-g1-typed-router):
// the still-uncovered error/edge branches of the transient B-rep factory (brep.go),
// the body point/edge/face queries (body_query.go), the face facet/stroke handlers
// (body_facets.go) and the planar wire-offset (bodies.go). These cases target
// branches NOT already reached by brep_g1cov_test.go / body_g1cov_test.go.

// bbcovBlock creates a transient block from the given min/max args and returns its handle.
func bbcovBlock(t *testing.T, r *Router, s *app.Session, min, max string) int {
	t.Helper()
	var res wire.BrepHandleResult
	call(t, r, s, "brep.createPrimitive",
		fmt.Sprintf(`{"kind":"block","min":%s,"max":%s}`, min, max), &res)
	return res.Handle
}

// bbcovSectionWireHandle creates a block and sections it at z=1, returning the transient
// wire body's handle (a closed rectangular loop) for the wire-offset / ruled-surface tests.
func bbcovSectionWireHandle(t *testing.T, r *Router, s *app.Session) int {
	t.Helper()
	block := bbcovBlock(t, r, s, `[0,0,0]`, `[4,3,2]`)
	var sec wire.BrepWiresResult
	call(t, r, s, "brep.sectionWithPlane",
		fmt.Sprintf(`{"source":{"handle":%d},"planeOrigin":[0,0,1],"planeNormal":[0,0,1]}`, block), &sec)
	return sec.Handle
}

// TestBbcovBooleanUnionInPlace drives brep.boolean's happy path: the blank grows to the
// union volume of two overlapping unit-cornered blocks (8+8-1 = 15 cm³), modified in place.
func TestBbcovBooleanUnionInPlace(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	blank := bbcovBlock(t, r, s, `[0,0,0]`, `[2,2,2]`)
	tool := bbcovBlock(t, r, s, `[1,1,1]`, `[3,3,3]`)
	var res wire.BrepHandleResult
	call(t, r, s, "brep.boolean",
		fmt.Sprintf(`{"blankHandle":%d,"operation":"union","tool":{"handle":%d}}`, blank, tool), &res)
	if res.Handle != blank || !res.Stats.Solid {
		t.Fatalf("union result = %+v, want the blank handle %d as a solid", res, blank)
	}
	if res.Stats.Volume < 14.9 || res.Stats.Volume > 15.1 {
		t.Fatalf("union volume = %g, want ~15 (8+8-1)", res.Stats.Volume)
	}
}

// TestBbcovBooleanRejectsBadOperation drives brepBoolean's operation-parse branch: the blank
// and tool both resolve, but an unknown operation spelling is rejected before any combine.
func TestBbcovBooleanRejectsBadOperation(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	blank := bbcovBlock(t, r, s, `[0,0,0]`, `[2,2,2]`)
	tool := bbcovBlock(t, r, s, `[1,1,1]`, `[3,3,3]`)
	bad := fmt.Sprintf(`{"blankHandle":%d,"operation":"merge","tool":{"handle":%d}}`, blank, tool)
	if err := tryCall(t, r, s, "brep.boolean", bad); err == nil {
		t.Fatal("brep.boolean with an unknown operation should error")
	}
}

// TestBbcovSectionRejectsBadArgs drives brepSectionWithPlane's three input-validation branches:
// an unresolvable source ref, a malformed plane origin, and a malformed plane normal.
func TestBbcovSectionRejectsBadArgs(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	h := bbcovBlock(t, r, s, `[0,0,0]`, `[2,2,2]`)
	for _, args := range bbcovBadSectionArgs(h) {
		if err := tryCall(t, r, s, "brep.sectionWithPlane", args); err == nil {
			t.Errorf("brep.sectionWithPlane(%s) returned nil error, want rejection", args)
		}
	}
}

// bbcovBadSectionArgs enumerates one malformed section request per validation branch.
func bbcovBadSectionArgs(handle int) []string {
	src := fmt.Sprintf(`"source":{"handle":%d}`, handle)
	return []string{
		`{"source":{},"planeOrigin":[0,0,0],"planeNormal":[0,0,1]}`,
		`{` + src + `,"planeOrigin":[0,0],"planeNormal":[0,0,1]}`,
		`{` + src + `,"planeOrigin":[0,0,0],"planeNormal":[0,0]}`,
	}
}

// TestBbcovRuledSurfaceRejectsBadWireRef drives brepRuledSurface → brepWireOf: a section-one
// wire index out of range, and a section-two whose source ref cannot resolve.
func TestBbcovRuledSurfaceRejectsBadWireRef(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	sec := bbcovSectionWireHandle(t, r, s)
	oob := fmt.Sprintf(`{"sectionOne":{"body":{"handle":%d},"wireIndex":9},"sectionTwo":{"body":{"handle":%d},"wireIndex":0}}`, sec, sec)
	if err := tryCall(t, r, s, "brep.ruledSurface", oob); err == nil {
		t.Error("ruledSurface with an out-of-range wireIndex should error")
	}
	badSrc := fmt.Sprintf(`{"sectionOne":{"body":{"handle":%d},"wireIndex":0},"sectionTwo":{"body":{},"wireIndex":0}}`, sec)
	if err := tryCall(t, r, s, "brep.ruledSurface", badSrc); err == nil {
		t.Error("ruledSurface with an unresolvable section-two source should error")
	}
}

// TestBbcovPrimitiveBuildErrors drives the primitive builders' geometric-validation branches
// (surfaced through the router): a degenerate cylinderCone and a self-intersecting torus.
func TestBbcovPrimitiveBuildErrors(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	for _, args := range bbcovBadPrimitiveGeometry() {
		if err := tryCall(t, r, s, "brep.createPrimitive", args); err == nil {
			t.Errorf("brep.createPrimitive(%s) returned nil error, want a geometry rejection", args)
		}
	}
}

// bbcovBadPrimitiveGeometry lists valid-vector requests that still fail the builder's guards.
func bbcovBadPrimitiveGeometry() []string {
	return []string{
		`{"kind":"cylinderCone","bottom":[0,0,0],"top":[0,0,0],"bottomRadius":1,"topRadius":1}`,
		`{"kind":"cylinderCone","bottom":[0,0,0],"top":[0,0,5],"bottomRadius":0,"topRadius":0}`,
		`{"kind":"torus","center":[0,0,0],"axis":[0,0,1],"majorRadius":1,"minorRadius":2}`,
	}
}

// TestBbcovLocateRejectsUnknownEntityKind drives entityKindFilter's default branch: an entity
// kind that is neither vertex, edge nor face is rejected.
func TestBbcovLocateRejectsUnknownEntityKind(t *testing.T) {
	t.Parallel()
	r, s := boxPartSession(t)
	bad := `{"bodyIndex":0,"point":[0,0,0],"entityKind":"solid","proximityTolerance":0.5}`
	if err := tryCall(t, r, s, "body.locateUsingPoint", bad); err == nil {
		t.Fatal("body.locateUsingPoint with an unknown entityKind should error")
	}
}

// TestBbcovConvexityConcaveAndTangent drives convexityClass's allConcave and
// tangentiallyConnected arms: a convex box reports no edges in either class.
func TestBbcovConvexityConcaveAndTangent(t *testing.T) {
	t.Parallel()
	r, s := boxPartSession(t)
	for _, coll := range []string{"allConcave", "tangentiallyConnected"} {
		var res wire.ConvexityEdgesResult
		call(t, r, s, "body.convexityEdges",
			fmt.Sprintf(`{"bodyIndex":0,"collection":%q}`, coll), &res)
		if len(res.Edges) != 0 {
			t.Errorf("convex box %s edges = %d, want 0", coll, len(res.Edges))
		}
	}
}

// TestBbcovBindTransientKeyBranches drives bodyBindTransientKey's not-found reply (an unknown
// transient key resolves to Found=false) and its body-index error branch.
func TestBbcovBindTransientKeyBranches(t *testing.T) {
	t.Parallel()
	r, s := boxPartSession(t)
	var miss wire.BindTransientKeyResult
	call(t, r, s, "body.bindTransientKey", `{"bodyIndex":0,"transientKey":999999999}`, &miss)
	if miss.Found {
		t.Error("binding an unknown transient key should report Found=false")
	}
	if err := tryCall(t, r, s, "body.bindTransientKey", `{"bodyIndex":9,"transientKey":1}`); err == nil {
		t.Error("body.bindTransientKey on an out-of-range body should error")
	}
}

// TestBbcovFaceFacetsAndStrokes drives faceCalculateFacets and faceCalculateStrokes happy paths:
// one box face facets into a triangle mesh and strokes into boundary polylines.
func TestBbcovFaceFacetsAndStrokes(t *testing.T) {
	t.Parallel()
	r, s := boxPartSession(t)
	args := mustJSON(t, wire.FaceFacetsArgs{BodyIndex: 0, FaceKey: bodcovFirstFaceKey(t, r, s), Tolerance: 0.02})
	var facets wire.FacetSetResult
	call(t, r, s, "face.calculateFacets", args, &facets)
	if facets.FacetCount == 0 || facets.VertexCount == 0 {
		t.Fatalf("face facets = %+v, want a non-empty mesh", facets)
	}
	var strokes wire.StrokeSetResult
	call(t, r, s, "face.calculateStrokes", args, &strokes)
	if strokes.PolylineCount == 0 {
		t.Fatalf("face strokes = %+v, want boundary polylines", strokes)
	}
}

// TestBbcovResolveFaceBadKey drives resolveFace's no-such-face branch through both face-addressed
// handlers, and its body-index error branch.
func TestBbcovResolveFaceBadKey(t *testing.T) {
	t.Parallel()
	r, s := boxPartSession(t)
	for _, m := range []string{"face.calculateFacets", "face.calculateStrokes"} {
		if err := tryCall(t, r, s, m, `{"bodyIndex":0,"faceKey":"nope","tolerance":0.02}`); err == nil {
			t.Errorf("%s with an unknown faceKey should error", m)
		}
	}
	if err := tryCall(t, r, s, "face.calculateFacets", `{"bodyIndex":9,"faceKey":"x","tolerance":0.02}`); err == nil {
		t.Error("face.calculateFacets on an out-of-range body should error")
	}
}

// TestBbcovExistingBeforeCalculate drives the retrieval-only error branches of
// bodyExistingFacets and bodyExistingStrokes: querying a tolerance never calculated errors.
func TestBbcovExistingBeforeCalculate(t *testing.T) {
	t.Parallel()
	r, s := boxPartSession(t)
	if err := tryCall(t, r, s, "body.existingFacets", `{"bodyIndex":0,"tolerance":0.007}`); err == nil {
		t.Error("body.existingFacets before any calculate should error")
	}
	if err := tryCall(t, r, s, "body.existingStrokes", `{"bodyIndex":0,"tolerance":0.007}`); err == nil {
		t.Error("body.existingStrokes before any calculate should error")
	}
}

// TestBbcovWireOffsetRejectsBadRefs drives wireOffsetPlanar → offsetSourceBody (missing transient
// handle) and → resolveWireRef (a document body with no free wires: index out of range).
func TestBbcovWireOffsetRejectsBadRefs(t *testing.T) {
	t.Parallel()
	r, s := boxPartSession(t)
	missing := `{"handle":424242,"wireIndex":0,"normal":[0,0,1],"distance":0.5}`
	if err := tryCall(t, r, s, "wire.offsetPlanar", missing); err == nil {
		t.Error("wire.offsetPlanar on a missing transient handle should error")
	}
	noWire := `{"bodyIndex":0,"wireIndex":0,"normal":[0,0,1],"distance":0.5}`
	if err := tryCall(t, r, s, "wire.offsetPlanar", noWire); err == nil {
		t.Error("wire.offsetPlanar on a solid document body (no free wires) should error")
	}
}

// TestBbcovWireOffsetCornerBranches drives wireOffsetPlanar's remaining offsetCorner arms — extend
// and the default circular — plus the unknown-corner and malformed-normal rejections.
func TestBbcovWireOffsetCornerBranches(t *testing.T) {
	t.Parallel()
	r, s := seededSession(t)
	sec := bbcovSectionWireHandle(t, r, s)
	for _, corner := range []string{"extend", ""} {
		var off wire.OffsetPlanarWireResult
		call(t, r, s, "wire.offsetPlanar",
			fmt.Sprintf(`{"handle":%d,"wireIndex":0,"normal":[0,0,1],"distance":-0.4,"cornerClosure":%q}`, sec, corner), &off)
		if off.Handle == 0 || len(off.Wires) != 1 {
			t.Fatalf("offset (corner=%q) = %+v, want one wire on a new handle", corner, off)
		}
	}
	bbcovAssertWireOffsetRejections(t, r, s, sec)
}

// bbcovAssertWireOffsetRejections checks the unknown-corner and malformed-normal error branches.
func bbcovAssertWireOffsetRejections(t *testing.T, r *Router, s *app.Session, sec int) {
	t.Helper()
	badCorner := fmt.Sprintf(`{"handle":%d,"wireIndex":0,"normal":[0,0,1],"distance":0.4,"cornerClosure":"round"}`, sec)
	if err := tryCall(t, r, s, "wire.offsetPlanar", badCorner); err == nil {
		t.Error("wire.offsetPlanar with an unknown cornerClosure should error")
	}
	badNormal := fmt.Sprintf(`{"handle":%d,"wireIndex":0,"normal":[0,0],"distance":0.4}`, sec)
	if err := tryCall(t, r, s, "wire.offsetPlanar", badNormal); err == nil {
		t.Error("wire.offsetPlanar with a malformed normal should error")
	}
}
