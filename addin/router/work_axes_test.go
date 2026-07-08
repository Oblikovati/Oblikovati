// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

func TestWorkAxesListIncludesOriginFrame(t *testing.T) {
	r, s := emptyPartSession(t)
	var list wire.ListWorkAxesResult
	call(t, r, s, "workAxes.list", "{}", &list)
	if len(list.Axes) != 3 {
		t.Fatalf("fresh part has %d work axes, want 3 origin axes", len(list.Axes))
	}
	for _, a := range list.Axes {
		if !a.IsOrigin || !a.Healthy || a.Kind != "line" {
			t.Errorf("origin axis %q: isOrigin=%v healthy=%v kind=%q, want origin+healthy+line", a.Name, a.IsOrigin, a.Healthy, a.Kind)
		}
	}
}

func TestWorkAxesCreateLineThenList(t *testing.T) {
	r, s := emptyPartSession(t)
	var res wire.CreateWorkAxisResult
	call(t, r, s, "workAxes.create", `{"kind":"line","origin":[1,2,3],"direction":[0,0,1]}`, &res)
	if !res.Healthy {
		t.Fatalf("line axis not healthy: %+v", res)
	}
	if res.Index != 3 { // after the 3 origin axes
		t.Errorf("new axis index = %d, want 3", res.Index)
	}
	var list wire.ListWorkAxesResult
	call(t, r, s, "workAxes.list", "{}", &list)
	if len(list.Axes) != 4 {
		t.Fatalf("after create, %d axes, want 4", len(list.Axes))
	}
	created := list.Axes[3]
	if created.IsOrigin || created.Kind != "line" {
		t.Errorf("created axis = %+v, want a user line axis", created)
	}
	if created.Origin[0] != 1 || created.Origin[1] != 2 || created.Origin[2] != 3 {
		t.Errorf("created axis origin = %v, want (1,2,3)", created.Origin)
	}
	if created.Direction[2] != 1 || created.Direction[0] != 0 || created.Direction[1] != 0 {
		t.Errorf("created axis direction = %v, want +Z", created.Direction)
	}
}

func TestWorkAxesCreateTwoPoints(t *testing.T) {
	r, s := emptyPartSession(t)
	var p0, p1 wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"at":[0,0,0]}`, &p0)
	call(t, r, s, "workPoints.create", `{"at":[0,0,5]}`, &p1)
	var res wire.CreateWorkAxisResult
	call(t, r, s, "workAxes.create",
		`{"kind":"two-points","refs":["`+p0.Ref+`","`+p1.Ref+`"]}`, &res)
	if !res.Healthy {
		t.Fatalf("two-points axis not healthy (user point refs must resolve): %+v", res)
	}
	var list wire.ListWorkAxesResult
	call(t, r, s, "workAxes.list", "{}", &list)
	created := list.Axes[res.Index]
	if created.Kind != "two-points" || created.Direction[2] != 1 {
		t.Errorf("two-points axis = %+v, want kind two-points along +Z", created)
	}
}

func TestWorkAxesCreatePlaneIntersection(t *testing.T) {
	r, s := emptyPartSession(t)
	// XY ∩ XZ = the X axis.
	var res wire.CreateWorkAxisResult
	call(t, r, s, "workAxes.create",
		`{"kind":"plane-intersection","refs":["origin/plane/xy","origin/plane/xz"]}`, &res)
	if !res.Healthy {
		t.Fatalf("plane-intersection axis not healthy: %+v", res)
	}
	var list wire.ListWorkAxesResult
	call(t, r, s, "workAxes.list", "{}", &list)
	created := list.Axes[res.Index]
	if created.Kind != "plane-intersection" {
		t.Errorf("kind = %q, want plane-intersection", created.Kind)
	}
	// The X axis: direction has no Y/Z component.
	if created.Direction[1] != 0 || created.Direction[2] != 0 {
		t.Errorf("XY∩XZ axis direction = %v, want parallel to X", created.Direction)
	}
}

func TestWorkAxesCreateUnknownKind(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workAxes.create", []byte(`{"kind":"no-such-kind"}`)); err == nil {
		t.Error("expected error for unknown work-axis kind")
	}
}

func TestWorkAxesCreateWrongReferenceCount(t *testing.T) {
	r, s := emptyPartSession(t)
	// two-points needs 2 references; supplying 1 is a bad request.
	if _, err := r.Handle(s, "workAxes.create",
		[]byte(`{"kind":"two-points","refs":["origin/point/center"]}`)); err == nil {
		t.Error("expected error for wrong reference count")
	}
}

func TestWorkAxesCreateLineBadDirection(t *testing.T) {
	r, s := emptyPartSession(t)
	// A zero direction vector cannot be normalized to a unit axis direction.
	if _, err := r.Handle(s, "workAxes.create",
		[]byte(`{"kind":"line","origin":[0,0,0],"direction":[0,0,0]}`)); err == nil {
		t.Error("expected error for a zero direction vector")
	}
}
