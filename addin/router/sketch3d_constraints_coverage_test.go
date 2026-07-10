// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// The declared-kind coverage guard of issue #142: every Geometric3DConstraintKind
// declared in api/types must be accepted end-to-end by sketch3d.addConstraint, so the
// contract can never again advertise constraint kinds the solver cannot build. A new
// kind added to api/types without a fixture here fails the completeness check below.

// constraint3DFixture builds the minimal geometry for one kind and returns the
// addConstraint entity refs.
type constraint3DFixture func(t *testing.T, r *Router, s *app.Session) []uint64

// twoLines3D builds two non-parallel lines and returns their entity ids.
func twoLines3D(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	var l1, l2 wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[1,0,0]]}`, &l1)
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,1,0],[1,1,1]]}`, &l2)
	return []uint64{l1.EntityID, l2.EntityID}
}

// oneLine3D builds a single line and returns its entity id.
func oneLine3D(t *testing.T, r *Router, s *app.Session) []uint64 {
	return twoLines3D(t, r, s)[:1]
}

// nPoints3D builds n standalone points and returns their ids.
func nPoints3D(n int) constraint3DFixture {
	return func(t *testing.T, r *Router, s *app.Session) []uint64 {
		t.Helper()
		out := make([]uint64, n)
		for i := range out {
			var p wire.AddSketch3DEntityResult
			call(t, r, s, "sketch3d.addEntity",
				fmt.Sprintf(`{"sketchIndex":0,"kind":"point","points":[[%d,%d,0]]}`, i, i*2), &p)
			out[i] = p.EntityID
		}
		return out
	}
}

// lineAndArc3D builds a line meeting an arc at (1,0,0) and returns their ids.
func lineAndArc3D(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	var l, a wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[1,0,0]]}`, &l)
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"arc","points":[[1,1,0],[1,0,0],[2,1,0]],"ccw":false}`, &a)
	return []uint64{l.EntityID, a.EntityID}
}

// lineAndSpline3D builds a line and a fit spline starting near the line's end.
func lineAndSpline3D(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	var l, sp wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[-1,0,0],[0,0,0]]}`, &l)
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"spline","points":[[0.2,0.1,0],[1,1,0],[2,0,1]]}`, &sp)
	return []uint64{l.EntityID, sp.EntityID}
}

// splineAndPoint3D builds a fit spline and a standalone point near a fit point.
func splineAndPoint3D(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	var sp, p wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"spline","points":[[0,0,0],[1,1,0],[2,0,1]]}`, &sp)
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"point","points":[[1.1,0.9,0]]}`, &p)
	return []uint64{sp.EntityID, p.EntityID}
}

// bendPieces3D builds two L-corner lines and a near-fillet arc, returning
// [arc, l1, l2] — the bend constraint binds them and pulls the join tight on solve.
func bendPieces3D(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	var l1, l2, a wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[0,0,0],[0.75,0,0]]}`, &l1)
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"line","points":[[1,0.25,0],[1,1,0]]}`, &l2)
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"arc","points":[[0.75,0.25,0],[0.75,0,0],[1,0.25,0]],"ccw":true}`, &a)
	return []uint64{a.EntityID, l1.EntityID, l2.EntityID}
}

// twoCircles3D builds two circles of different radii and returns their ids.
func twoCircles3D(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	var c1, c2 wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0,0]],"radius":"10 mm"}`, &c1)
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0,5]],"radius":"7 mm"}`, &c2)
	return []uint64{c1.EntityID, c2.EntityID}
}

// helixAndCircle3D builds a coaxial (Z) helix and circle and returns their ids.
func helixAndCircle3D(t *testing.T, r *Router, s *app.Session) []uint64 {
	t.Helper()
	var h, c wire.AddSketch3DEntityResult
	call(t, r, s, "sketch3d.addEntity",
		`{"sketchIndex":0,"kind":"helical","points":[[0,0,1]],"radius":"4 mm","mode":"pitchRevolution","pitch":"10 mm","revolutions":3}`, &h)
	call(t, r, s, "sketch3d.addEntity", `{"sketchIndex":0,"kind":"circle","points":[[0,0,0]],"radius":"5 mm"}`, &c)
	return []uint64{h.EntityID, c.EntityID}
}

// declared3DConstraintFixtures maps EVERY declared kind to its minimal fixture.
var declared3DConstraintFixtures = map[types.Geometric3DConstraintKind]constraint3DFixture{
	types.Geo3DCoincident:    nPoints3D(2),
	types.Geo3DCollinear:     nPoints3D(3),
	types.Geo3DConcentric:    nPoints3D(2),
	types.Geo3DParallel:      twoLines3D,
	types.Geo3DPerpendicular: twoLines3D,
	types.Geo3DTangent:       lineAndArc3D,
	types.Geo3DSmooth:        lineAndSpline3D,
	types.Geo3DMidpoint: func(t *testing.T, r *Router, s *app.Session) []uint64 {
		pts := nPoints3D(1)(t, r, s)
		return append(pts, oneLine3D(t, r, s)...)
	},
	types.Geo3DGround:            nPoints3D(1),
	types.Geo3DParallelToXAxis:   oneLine3D,
	types.Geo3DParallelToYAxis:   oneLine3D,
	types.Geo3DParallelToZAxis:   oneLine3D,
	types.Geo3DParallelToXYPlane: oneLine3D,
	types.Geo3DParallelToXZPlane: oneLine3D,
	types.Geo3DParallelToYZPlane: oneLine3D,
	types.Geo3DSplineFitPoints:   splineAndPoint3D,
	types.Geo3DHelical:           helixAndCircle3D,
	types.Geo3DEqual:             twoCircles3D,
	types.Geo3DBend:              bendPieces3D,
}

// trackedUnimplemented3DKinds are declared kinds knowingly absent from the solver,
// each requiring an open issue. The completeness check fails when a kind is neither
// covered nor tracked here. Currently empty — every declared kind is implemented
// (#142 closed the gap; #143 added bend) — and kept so the next gap must be tracked.
var trackedUnimplemented3DKinds = map[types.Geometric3DConstraintKind]string{}

// dedicated3DConstraintKinds are implemented kinds the generic fixture harness cannot drive because
// they take more than entity ids — onFace needs a face reference key + a real solid, so it has its
// own end-to-end test (TestSketch3DOnFaceConstraintOverWire) instead of a fixture (#1839).
var dedicated3DConstraintKinds = map[types.Geometric3DConstraintKind]string{
	types.Geo3DOnFace: "TestSketch3DOnFaceConstraintOverWire",
}

// allDeclared3DConstraintKinds mirrors the const block in api/types/sketch3d.go
// (Go cannot enumerate constants); keep in sync when the API grows.
var allDeclared3DConstraintKinds = []types.Geometric3DConstraintKind{
	types.Geo3DCoincident, types.Geo3DCollinear, types.Geo3DConcentric, types.Geo3DEqual,
	types.Geo3DParallel, types.Geo3DPerpendicular, types.Geo3DTangent, types.Geo3DSmooth,
	types.Geo3DMidpoint, types.Geo3DGround,
	types.Geo3DParallelToXAxis, types.Geo3DParallelToYAxis, types.Geo3DParallelToZAxis,
	types.Geo3DParallelToXYPlane, types.Geo3DParallelToXZPlane, types.Geo3DParallelToYZPlane,
	types.Geo3DSplineFitPoints, types.Geo3DBend, types.Geo3DHelical, types.Geo3DOnFace,
}

// TestEvery3DConstraintKindAccepted drives sketch3d.addConstraint for every declared
// kind with a fixture and asserts the constraint lands with the same kind reported
// back by sketch3d.constraints.
func TestEvery3DConstraintKindAccepted(t *testing.T) {
	for kind, fixture := range declared3DConstraintFixtures {
		t.Run(string(kind), func(t *testing.T) {
			r, s := emptyPartSession(t)
			call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
			refs := fixture(t, r, s)
			var res wire.AddSketch3DConstraintResult
			call(t, r, s, "sketch3d.addConstraint",
				fmt.Sprintf(`{"sketchIndex":0,"kind":"%s","entities":%s}`, kind, jsonUints(refs)), &res)
			if res.Kind != string(kind) {
				t.Errorf("addConstraint kind = %q, want %q", res.Kind, kind)
			}
			var cons wire.ListConstraints3DResult
			call(t, r, s, "sketch3d.constraints", `{"sketchIndex":0}`, &cons)
			if len(cons.Constraints) != 1 || cons.Constraints[0].Kind != string(kind) {
				t.Errorf("enumerated constraints = %+v, want one of kind %q", cons.Constraints, kind)
			}
		})
	}
}

// TestDeclared3DConstraintKindsComplete fails when a declared kind is neither covered
// by a fixture (incl. the equal special case) nor in the tracked-unimplemented list —
// the guard that api/types can never ship ahead of the solver again (issue #142).
func TestDeclared3DConstraintKindsComplete(t *testing.T) {
	for _, kind := range allDeclared3DConstraintKinds {
		_, covered := declared3DConstraintFixtures[kind]
		_, dedicated := dedicated3DConstraintKinds[kind]
		issue, tracked := trackedUnimplemented3DKinds[kind]
		switch {
		case covered && tracked:
			t.Errorf("kind %q is both covered and tracked-unimplemented (%s) — remove one", kind, issue)
		case !covered && !dedicated && !tracked:
			t.Errorf("kind %q is declared in api/types but has no fixture, dedicated test, or tracking issue", kind)
		}
	}
}

// jsonUints renders ids as a JSON array.
func jsonUints(ids []uint64) string {
	out := "["
	for i, id := range ids {
		if i > 0 {
			out += ","
		}
		out += fmt.Sprintf("%d", id)
	}
	return out + "]"
}
