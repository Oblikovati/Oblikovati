// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestFlipWorkPlaneNormal flips a user plane's normal over the wire and confirms it reverses (and a
// second flip restores it); an origin plane and a bad index are clean errors (#1851).
func TestFlipWorkPlaneNormal(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	var pl wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`, &pl)

	var flip1, flip2 wire.FlipWorkPlaneResult
	call(t, r, s, "workPlanes.flipNormal", `{"index":`+itoaInt(pl.Index)+`}`, &flip1)
	call(t, r, s, "workPlanes.flipNormal", `{"index":`+itoaInt(pl.Index)+`}`, &flip2)
	if dot3(flip1.Plane.Normal, flip2.Plane.Normal) > -0.999 {
		t.Errorf("two flips should give opposite normals, got %v then %v", flip1.Plane.Normal, flip2.Plane.Normal)
	}

	if _, err := r.Handle(s, "workPlanes.flipNormal", []byte(`{"index":0}`)); err == nil {
		t.Error("flipping an origin plane should error")
	}
	if _, err := r.Handle(s, "workPlanes.flipNormal", []byte(`{"index":99}`)); err == nil {
		t.Error("a bad index should error")
	}
}

// TestRedefineWorkPlaneDisplay sets grounded / auto-resize / explicit size via redefine and confirms
// the list reports them back (#1851).
func TestRedefineWorkPlaneDisplay(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	var pl wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`, &pl)

	var res wire.RedefineWorkPlaneResult
	call(t, r, s, "workPlanes.redefine",
		`{"index":`+itoaInt(pl.Index)+`,"grounded":true,"autoResize":true,"size":[[1,2,0],[3,4,0]]}`, &res)
	// SetSize turns auto-resize back off; grounded stays on.
	if !res.Plane.Grounded {
		t.Error("plane should report grounded=true")
	}
	if res.Plane.AutoResize {
		t.Error("explicit size should turn off auto-resize")
	}
	if len(res.Plane.Size) != 2 || res.Plane.Size[0][0] != 1 || res.Plane.Size[1][1] != 4 {
		t.Errorf("plane should report the explicit size corners, got %v", res.Plane.Size)
	}
}

// TestRedefineWorkPlaneSizeBadArity: a size that is not two corners is a clean error (#1851).
func TestRedefineWorkPlaneSizeBadArity(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	var pl wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`, &pl)
	if _, err := r.Handle(s, "workPlanes.redefine", []byte(`{"index":`+itoaInt(pl.Index)+`,"size":[[1,2,0]]}`)); err == nil {
		t.Error("a one-corner size should error")
	}
}

// TestCreateTwoPlanesWithQuadrant: the quadrant point routes to the ...Toward constructor, and
// opposite quadrant points give different (perpendicular) bisector normals (#1844).
func TestCreateTwoPlanesWithQuadrant(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	var a, b wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"two-planes","refs":["origin/plane/xy","origin/plane/xz"],"quadrant":[0,-1,1]}`, &a)
	call(t, r, s, "workPlanes.create", `{"kind":"two-planes","refs":["origin/plane/xy","origin/plane/xz"],"quadrant":[0,1,1]}`, &b)
	if !a.Healthy || !b.Healthy {
		t.Fatalf("both bisectors should be healthy: %+v %+v", a, b)
	}
	var list wire.ListWorkPlanesResult
	call(t, r, s, "workPlanes.list", "{}", &list)
	na, nb := findPlane(list.Planes, a.Ref).Normal, findPlane(list.Planes, b.Ref).Normal
	if dot3(na, nb) > 1e-6 || dot3(na, nb) < -1e-6 {
		t.Errorf("opposite quadrant points should give perpendicular bisectors, dot = %v", dot3(na, nb))
	}
}

// TestCreateSolutionPlaneWrongRefCount: a quadrant point with the wrong reference count errors from
// the solution dispatcher (#1844).
func TestCreateSolutionPlaneWrongRefCount(t *testing.T) {
	t.Parallel()
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workPlanes.create", []byte(`{"kind":"two-planes","refs":["origin/plane/xy"],"quadrant":[0,1,1]}`)); err == nil {
		t.Error("two-planes with one reference should error")
	}
}

func dot3(a, b []float64) float64 {
	if len(a) != 3 || len(b) != 3 {
		return 0
	}
	return a[0]*b[0] + a[1]*b[1] + a[2]*b[2]
}
