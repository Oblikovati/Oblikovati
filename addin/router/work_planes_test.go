// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
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

func TestWorkPointsCreateAndThreePointPlane(t *testing.T) {
	r, s := emptyPartSession(t)
	// Three created points define a plane tilted off the principal planes.
	var p0, p1, p2 wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"at":[0,0,0]}`, &p0)
	call(t, r, s, "workPoints.create", `{"at":[4,0,0]}`, &p1)
	call(t, r, s, "workPoints.create", `{"at":[0,4,2]}`, &p2)
	// The origin center point occupies index 0, so user points start at point/1.
	if p0.Ref != "point/1" || p1.Ref != "point/2" || p2.Ref != "point/3" {
		t.Fatalf("point refs = %q,%q,%q, want point/1..3", p0.Ref, p1.Ref, p2.Ref)
	}

	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create",
		`{"kind":"three-points","refs":["`+p0.Ref+`","`+p1.Ref+`","`+p2.Ref+`"]}`, &res)
	if !res.Healthy {
		t.Fatalf("three-point plane not healthy (user point refs must resolve): %+v", res)
	}

	// Its three slots are point slots.
	var list wire.ListWorkPlanesResult
	call(t, r, s, "workPlanes.list", "{}", &list)
	p := list.Planes[res.Index]
	if p.Kind != "three-points" || len(p.Slots) != 3 || p.Slots[2].Kind != "point" {
		t.Errorf("three-point plane slots = %+v (kind %q), want 3 point slots", p.Slots, p.Kind)
	}
}

func TestWorkPlanesRedefineThreePointRepick(t *testing.T) {
	r, s := emptyPartSession(t)
	var a, b, c wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"at":[0,0,0]}`, &a)
	call(t, r, s, "workPoints.create", `{"at":[4,0,0]}`, &b)
	call(t, r, s, "workPoints.create", `{"at":[0,4,0]}`, &c) // all in the XY plane → ±Z normal
	var plane wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create",
		`{"kind":"three-points","refs":["`+a.Ref+`","`+b.Ref+`","`+c.Ref+`"]}`, &plane)
	if !plane.Healthy {
		t.Fatalf("three-point plane from user points should be healthy: %+v", plane)
	}
	// The fresh plane lies in XY, so its normal is vertical (±Z).
	var list wire.ListWorkPlanesResult
	call(t, r, s, "workPlanes.list", "{}", &list)
	if z := list.Planes[plane.Index].Normal[2]; z != 1 && z != -1 {
		t.Fatalf("initial three-point plane normal = %v, want vertical ±Z", list.Planes[plane.Index].Normal)
	}

	// A fourth point above the XY plane; re-point slot 2 at it → the plane tilts off ±Z.
	var d wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"at":[0,4,3]}`, &d)
	var rd wire.RedefineWorkPlaneResult
	call(t, r, s, "workPlanes.redefine", fmt.Sprintf(`{"index":%d,"repick":[{"slot":2,"ref":%q}]}`, plane.Index, d.Ref), &rd)
	if !rd.Plane.Healthy {
		t.Fatalf("redefined three-point plane not healthy: %+v", rd.Plane)
	}
	if z := rd.Plane.Normal[2]; z == 1 || z == -1 {
		t.Errorf("after re-picking point 3 above the plane, normal = %v, want tilted off ±Z", rd.Plane.Normal)
	}
}

// TestWorkPlanesRedefineRepickAcceptsListRef: a repick with the Ref string workPlanes.list
// returns for a user plane ("plane/3") binds to that plane — it was once misread as a B-rep
// face key, silently sickening the plane.
func TestWorkPlanesRedefineRepickAcceptsListRef(t *testing.T) {
	r, s := emptyPartSession(t)
	var first, second wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`, &first)
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xz"],"offset":"10 mm"}`, &second)

	// Re-base the second plane onto the first (its list ref): both offsets stack along +Z.
	var rd wire.RedefineWorkPlaneResult
	call(t, r, s, "workPlanes.redefine", `{"index":4,"repick":[{"slot":0,"ref":"`+first.Ref+`"}]}`, &rd)
	if !rd.Plane.Healthy {
		t.Fatalf("repick onto a user-plane list ref produced an unhealthy plane: %+v", rd.Plane)
	}
	if rd.Plane.Origin[2] != 2 { // 1 cm (first) + 1 cm (second) above XY
		t.Errorf("re-based plane origin = %v, want z=2 (stacked offsets)", rd.Plane.Origin)
	}
}

// TestWorkPlanesRedefineRejectsSelfReference: re-picking a plane's base to its own ref must
// fail the call — it once stayed healthy while the plane drifted on every recompute.
func TestWorkPlanesRedefineRejectsSelfReference(t *testing.T) {
	r, s := emptyPartSession(t)
	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`, &res)
	if _, err := r.Handle(s, "workPlanes.redefine",
		[]byte(`{"index":3,"repick":[{"slot":0,"ref":"`+res.Ref+`"}]}`)); err == nil {
		t.Error("a self-referential repick must error")
	}
}

// TestWorkPlanesRedefineAppliesScalarsAndRepickTogether: the wire doc promises scalars and
// repicks are applied together; the angle edit must survive the line re-pick (value-typed
// definitions once dropped it).
func TestWorkPlanesRedefineAppliesScalarsAndRepickTogether(t *testing.T) {
	r, s := emptyPartSession(t)
	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create",
		`{"kind":"line-plane-angle","refs":["origin/axis/x","origin/plane/xy"],"angle":"0 deg"}`, &res)

	var rd wire.RedefineWorkPlaneResult
	call(t, r, s, "workPlanes.redefine",
		`{"index":3,"scalars":[{"index":0,"value":"45 deg"}],"repick":[{"slot":0,"ref":"origin/axis/y"}]}`, &rd)
	if !rd.Plane.Healthy {
		t.Fatalf("combined redefine produced an unhealthy plane: %+v", rd.Plane)
	}
	if rd.Plane.Scalars[0].Value != 45 {
		t.Errorf("angle after combined redefine = %v deg, want 45 — the scalar edit was lost", rd.Plane.Scalars[0].Value)
	}
}

// TestWorkPlanesRedefineRejectsBadEdits: every malformed edit fails the call (it does not
// silently skip or partially apply).
func TestWorkPlanesRedefineRejectsBadEdits(t *testing.T) {
	r, s := emptyPartSession(t)
	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`, &res)

	for name, req := range map[string]string{
		"scalar index out of range": `{"index":3,"scalars":[{"index":5,"value":"1 mm"}]}`,
		"unparseable scalar value":  `{"index":3,"scalars":[{"index":0,"value":"garbage"}]}`,
		"slot out of range":         `{"index":3,"repick":[{"slot":5,"ref":"origin/plane/xz"}]}`,
		"nonexistent work ref":      `{"index":3,"repick":[{"slot":0,"ref":"plane/99"}]}`,
	} {
		if _, err := r.Handle(s, "workPlanes.redefine", []byte(req)); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

// TestWorkPlanesCreateKindRoundTripsToList pins the create-kind ↔ list Kind vocabulary: the
// types.WorkPlaneKind a plane is created as must be the Kind workPlanes.list reports (the
// model's kindName strings and the api/types constants are coupled by convention only).
func TestWorkPlanesCreateKindRoundTripsToList(t *testing.T) {
	r, s := emptyPartSession(t)
	for i, req := range []struct{ kind, body string }{
		{"plane-offset", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`},
		{"three-points", `{"kind":"three-points","refs":["origin/point/center","origin/point/center","origin/point/center"]}`},
		{"plane-point", `{"kind":"plane-point","refs":["origin/plane/xy","origin/point/center"]}`},
		{"two-planes", `{"kind":"two-planes","refs":["origin/plane/xy","origin/plane/xz"]}`},
		{"line-plane-angle", `{"kind":"line-plane-angle","refs":["origin/axis/x","origin/plane/xy"],"angle":"45 deg"}`},
		{"two-lines", `{"kind":"two-lines","refs":["origin/axis/x","origin/axis/y"]}`},
		{"normal-to-curve", `{"kind":"normal-to-curve","refs":["origin/axis/z","origin/point/center"]}`},
	} {
		var res wire.CreateWorkPlaneResult
		call(t, r, s, "workPlanes.create", req.body, &res)
		var list wire.ListWorkPlanesResult
		call(t, r, s, "workPlanes.list", "{}", &list)
		if got := list.Planes[3+i].Kind; got != req.kind {
			t.Errorf("plane created as %q lists Kind %q", req.kind, got)
		}
	}
}

// TestWorkPlanesRedefineReportsSickReason: an unsatisfiable (but well-formed) redefine reports
// healthy=false with the reason, so a remote client can diagnose what went wrong.
func TestWorkPlanesRedefineReportsSickReason(t *testing.T) {
	r, s := emptyPartSession(t)
	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`, &res)

	var rd wire.RedefineWorkPlaneResult
	call(t, r, s, "workPlanes.redefine", `{"index":3,"repick":[{"slot":0,"ref":"bogus-face-key"}]}`, &rd)
	if rd.Plane.Healthy {
		t.Fatal("a repick to a dangling face key must report healthy=false")
	}
	if rd.Plane.Reason == "" {
		t.Error("an unhealthy plane must report the reason it is sick")
	}
}

// TestWorkPlanesRedefineRollsBackOnError: when one edit of a batch fails, earlier edits of
// the same call must not stick (the definition is snapshotted and restored).
func TestWorkPlanesRedefineRollsBackOnError(t *testing.T) {
	r, s := emptyPartSession(t)
	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`, &res)

	if _, err := r.Handle(s, "workPlanes.redefine",
		[]byte(`{"index":3,"scalars":[{"index":0,"value":"50 mm"}],"repick":[{"slot":9,"ref":"origin/plane/xz"}]}`)); err == nil {
		t.Fatal("an out-of-range slot must fail the call")
	}
	var list wire.ListWorkPlanesResult
	call(t, r, s, "workPlanes.list", "{}", &list)
	if got := list.Planes[3].Scalars[0].Value; got != 10 {
		t.Errorf("offset after a failed batch = %v mm, want 10 (rolled back)", got)
	}
}
