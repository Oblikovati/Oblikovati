// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati/api/wire"
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
