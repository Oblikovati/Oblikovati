// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

func TestWorkPlanesListIncludesOriginFrame(t *testing.T) {
	r, s := emptyPartSession(t)
	var list wire.ListWorkPlanesResult
	call(t, r, s, "workPlanes.list", "{}", &list)
	if len(list.Planes) != 3 {
		t.Fatalf("fresh part has %d work planes, want 3 origin planes", len(list.Planes))
	}
	for _, p := range list.Planes {
		if !p.IsOrigin || !p.Healthy {
			t.Errorf("origin plane %q: isOrigin=%v healthy=%v, want both true", p.Name, p.IsOrigin, p.Healthy)
		}
	}
}

func TestWorkPlanesCreateOffsetThenList(t *testing.T) {
	r, s := emptyPartSession(t)
	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`, &res)
	if !res.Healthy {
		t.Fatalf("offset plane not healthy: %+v", res)
	}
	if res.Index != 3 { // after the 3 origin planes
		t.Errorf("new plane index = %d, want 3", res.Index)
	}
	var list wire.ListWorkPlanesResult
	call(t, r, s, "workPlanes.list", "{}", &list)
	if len(list.Planes) != 4 {
		t.Fatalf("after create, %d planes, want 4", len(list.Planes))
	}
	created := list.Planes[3]
	// "10 mm" is 1.0 cm in model units; the XY plane offset +Z lands at z=1.
	if created.IsOrigin || len(created.Origin) != 3 || created.Origin[2] != 1 {
		t.Errorf("created plane = %+v, want a user plane at z=1", created)
	}
}

func TestWorkPlanesCreateMidplane(t *testing.T) {
	r, s := emptyPartSession(t)
	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create",
		`{"kind":"two-planes","refs":["origin/plane/xy","origin/plane/xz"]}`, &res)
	if !res.Healthy {
		t.Errorf("midplane not healthy: %+v", res)
	}
}

func TestWorkPlanesCreateUnknownKind(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workPlanes.create", []byte(`{"kind":"no-such-kind"}`)); err == nil {
		t.Error("expected error for unknown work-plane kind")
	}
}

func TestWorkPlanesCreateWrongReferenceCount(t *testing.T) {
	r, s := emptyPartSession(t)
	// three-points needs 3 references; supplying 2 is a bad request.
	if _, err := r.Handle(s, "workPlanes.create",
		[]byte(`{"kind":"three-points","refs":["origin/point/center","origin/axis/x"]}`)); err == nil {
		t.Error("expected error for wrong reference count")
	}
}

func TestWorkPlanesCreateBadOffsetExpression(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workPlanes.create",
		[]byte(`{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"ten"}`)); err == nil {
		t.Error("expected error for an unparseable offset expression")
	}
}

func TestWorkPlanesListReportsScalarsAndSlots(t *testing.T) {
	r, s := emptyPartSession(t)
	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"20 mm"}`, &res)

	var list wire.ListWorkPlanesResult
	call(t, r, s, "workPlanes.list", "{}", &list)
	p := list.Planes[res.Index]
	if p.Kind != "plane-offset" {
		t.Errorf("kind = %q, want plane-offset", p.Kind)
	}
	if len(p.Scalars) != 1 || p.Scalars[0].Label != "Offset" || p.Scalars[0].Unit != "mm" || p.Scalars[0].Value != 20 {
		t.Errorf("scalars = %+v, want one Offset = 20 mm", p.Scalars)
	}
	if len(p.Slots) != 1 || p.Slots[0].Kind != "plane" {
		t.Errorf("slots = %+v, want one plane slot", p.Slots)
	}
}

func TestWorkPlanesRedefineScalarMovesPlane(t *testing.T) {
	r, s := emptyPartSession(t)
	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`, &res)

	var rd wire.RedefineWorkPlaneResult
	call(t, r, s, "workPlanes.redefine", `{"index":3,"scalars":[{"index":0,"value":"50 mm"}]}`, &rd)
	// 50 mm = 5 cm; the XY-offset plane lands at z=5.
	if !rd.Plane.Healthy || rd.Plane.Origin[2] != 5 {
		t.Errorf("after scalar redefine, plane = %+v, want healthy at z=5", rd.Plane)
	}
	if rd.Plane.Scalars[0].Value != 50 {
		t.Errorf("redefined offset reads back %v mm, want 50", rd.Plane.Scalars[0].Value)
	}
}

func TestWorkPlanesRedefineRepickReorientsPlane(t *testing.T) {
	r, s := emptyPartSession(t)
	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`, &res)

	// Re-point the base from XY (+Z normal) to XZ (+Y normal): the plane reorients.
	var rd wire.RedefineWorkPlaneResult
	call(t, r, s, "workPlanes.redefine", `{"index":3,"repick":[{"slot":0,"ref":"origin/plane/xz"}]}`, &rd)
	if !rd.Plane.Healthy {
		t.Fatalf("repick produced an unhealthy plane: %+v", rd.Plane)
	}
	// XZ plane's normal is +Y, so the offset plane's normal is now ±Y, not ±Z.
	if rd.Plane.Normal[2] != 0 {
		t.Errorf("after re-picking base to XZ, normal = %v, want no Z component", rd.Plane.Normal)
	}
}

func TestWorkPlanesRedefineRejectsOriginAndBadIndex(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workPlanes.redefine", []byte(`{"index":0}`)); err == nil {
		t.Error("redefining an origin plane (index 0) should error")
	}
	if _, err := r.Handle(s, "workPlanes.redefine", []byte(`{"index":99}`)); err == nil {
		t.Error("redefining an out-of-range index should error")
	}
}
