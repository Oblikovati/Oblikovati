// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestCreateRevolvedFaceAxis dispatches a revolved-face axis over the wire. With no body the face
// reference does not resolve, so the axis is created but reports healthy=false — exercising the
// arity-1 dispatch without needing a B-rep body (#1840).
func TestCreateRevolvedFaceAxis(t *testing.T) {
	r, s := emptyPartSession(t)
	var res wire.CreateWorkAxisResult
	call(t, r, s, "workAxes.create", `{"kind":"revolved-face","refs":["face/cyl-1"]}`, &res)
	if res.Ref == "" {
		t.Error("a revolved-face axis should be created (with a stable ref) even when its face is unresolved")
	}
	if res.Healthy {
		t.Error("with no body the face is unresolved, so the axis should report healthy=false")
	}
}

// TestCreateRevolvedFaceAxisWrongRefCount: revolved-face needs exactly one face reference (#1840).
func TestCreateRevolvedFaceAxisWrongRefCount(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workAxes.create", []byte(`{"kind":"revolved-face","refs":["a","b"]}`)); err == nil {
		t.Error("revolved-face with two references should error")
	}
	if _, err := r.Handle(s, "workAxes.create", []byte(`{"kind":"revolved-face"}`)); err == nil {
		t.Error("revolved-face with no reference should error")
	}
}

// TestCreateFaceCenterPoint dispatches a face-center point over the wire (unresolved without a body,
// so healthy=false) — exercising the arity-1 point dispatch (#1842).
func TestCreateFaceCenterPoint(t *testing.T) {
	r, s := emptyPartSession(t)
	var res wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"kind":"face-center","refs":["face/sph-1"]}`, &res)
	if res.Ref == "" {
		t.Error("a face-center point should be created even when its face is unresolved")
	}
	if res.Healthy {
		t.Error("with no body the face is unresolved, so the point should report healthy=false")
	}
}

// TestCreateFaceCenterWrongRefCount: face-center needs exactly one face reference (#1842).
func TestCreateFaceCenterWrongRefCount(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workPoints.create", []byte(`{"kind":"face-center","refs":["a","b"]}`)); err == nil {
		t.Error("face-center with two references should error")
	}
}
