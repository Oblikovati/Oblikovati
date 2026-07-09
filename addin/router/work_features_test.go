// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/model/compdef"
)

// TestWorkPlaneLineAndPoint creates the line-and-point plane over the wire: the origin X axis and a
// point at (0,5,0) define the XY plane — a healthy datum (#1843).
func TestWorkPlaneLineAndPoint(t *testing.T) {
	r, s := emptyPartSession(t)
	var p wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"at":[0,5,0]}`, &p)

	var res wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"line-point","refs":["origin/axis/x","`+p.Ref+`"]}`, &res)
	if !res.Healthy {
		t.Fatalf("line-point plane not healthy: %+v", res)
	}
}

// TestWorkPlaneLineAndPointWrongRefCount: line-point needs exactly two references.
func TestWorkPlaneLineAndPointWrongRefCount(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workPlanes.create", []byte(`{"kind":"line-point","refs":["origin/axis/x"]}`)); err == nil {
		t.Error("line-point with one reference should error")
	}
}

// TestSetWorkFeatureVisible hides a datum plane, axis, and point by ref and confirms each — the
// plane via its list Visible flag, the axis and point via the model (#1856).
func TestSetWorkFeatureVisible(t *testing.T) {
	r, s := emptyPartSession(t)
	var pl wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`, &pl)
	var res wire.OKResult
	call(t, r, s, "workFeatures.setVisible", `{"ref":"`+pl.Ref+`","visible":false}`, &res)
	if !res.OK {
		t.Fatal("setVisible did not report OK")
	}
	var list wire.ListWorkPlanesResult
	call(t, r, s, "workPlanes.list", "{}", &list)
	for _, p := range list.Planes {
		if p.Ref == pl.Ref && p.Visible {
			t.Error("plane should be hidden after setVisible false")
		}
	}

	var ax wire.CreateWorkAxisResult
	call(t, r, s, "workAxes.create", `{"kind":"line","origin":[0,0,0],"direction":[0,0,1]}`, &ax)
	call(t, r, s, "workFeatures.setVisible", `{"ref":"`+ax.Ref+`","visible":false}`, &res)

	var pt wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"at":[1,2,3]}`, &pt)
	call(t, r, s, "workFeatures.setVisible", `{"ref":"`+pt.Ref+`","visible":false}`, &res)

	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.WorkAxes().Item(ax.Index).Visible() {
		t.Error("axis should be hidden")
	}
	if def.WorkPoints().Item(pt.Index).Visible() {
		t.Error("point should be hidden")
	}
}

// TestSetWorkFeatureVisibleUnknownRef: a ref naming no datum is a clean error.
func TestSetWorkFeatureVisibleUnknownRef(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workFeatures.setVisible", []byte(`{"ref":"plane/99","visible":false}`)); err == nil {
		t.Error("an unknown ref should error")
	}
}

// TestDeleteWorkFeature removes a user plane over the wire: the delete reports it removed, and
// workPlanes.list no longer includes it (#1855).
func TestDeleteWorkFeature(t *testing.T) {
	r, s := emptyPartSession(t)
	var pl wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"10 mm"}`, &pl)

	var del wire.DeleteWorkFeatureResult
	call(t, r, s, "workFeatures.delete", `{"ref":"`+pl.Ref+`","retainDependents":true}`, &del)
	if len(del.Deleted) != 1 || del.Deleted[0] != pl.Ref {
		t.Errorf("deleted = %v, want just %q", del.Deleted, pl.Ref)
	}
	var list wire.ListWorkPlanesResult
	call(t, r, s, "workPlanes.list", "{}", &list)
	for _, p := range list.Planes {
		if p.Ref == pl.Ref {
			t.Error("a deleted plane should not appear in workPlanes.list")
		}
	}
}

// TestDeleteWorkFeatureCascades: deleting a point (retainDependents=false) also removes the axis
// built through it, and both drop out of their lists (#1855).
func TestDeleteWorkFeatureCascades(t *testing.T) {
	r, s := emptyPartSession(t)
	var pt wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"at":[0,0,5]}`, &pt)
	var ax wire.CreateWorkAxisResult
	call(t, r, s, "workAxes.create", `{"kind":"two-points","refs":["`+pt.Ref+`","origin/point/center"]}`, &ax)

	var del wire.DeleteWorkFeatureResult
	call(t, r, s, "workFeatures.delete", `{"ref":"`+pt.Ref+`"}`, &del)
	if len(del.Deleted) != 2 {
		t.Errorf("deleted = %v, want the point and its dependent axis", del.Deleted)
	}
	var axes wire.ListWorkAxesResult
	call(t, r, s, "workAxes.list", "{}", &axes)
	for _, a := range axes.Axes {
		if a.Ref == ax.Ref {
			t.Error("the cascaded axis should be gone from workAxes.list")
		}
	}
}

// TestDeleteWorkFeatureErrors: an origin ref and an unknown ref both fail cleanly (#1855).
func TestDeleteWorkFeatureErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	if _, err := r.Handle(s, "workFeatures.delete", []byte(`{"ref":"origin/plane/xy"}`)); err == nil {
		t.Error("deleting an origin datum should error")
	}
	if _, err := r.Handle(s, "workFeatures.delete", []byte(`{"ref":"point/404"}`)); err == nil {
		t.Error("deleting an unknown ref should error")
	}
}

// TestCreateConstructionDatum: the construction flag on create is reported back on each list DTO,
// independent of visibility (#1849).
func TestCreateConstructionDatum(t *testing.T) {
	r, s := emptyPartSession(t)
	var pl wire.CreateWorkPlaneResult
	call(t, r, s, "workPlanes.create", `{"kind":"plane-offset","refs":["origin/plane/xy"],"offset":"5 mm","construction":true,"visible":false}`, &pl)
	var pt wire.CreateWorkPointResult
	call(t, r, s, "workPoints.create", `{"at":[1,2,3],"construction":true}`, &pt)
	var ax wire.CreateWorkAxisResult
	call(t, r, s, "workAxes.create", `{"kind":"line","origin":[0,0,0],"direction":[0,0,1],"construction":true}`, &ax)

	var planes wire.ListWorkPlanesResult
	call(t, r, s, "workPlanes.list", "{}", &planes)
	if !findPlane(planes.Planes, pl.Ref).Construction {
		t.Error("plane should report construction=true")
	}
	if findPlane(planes.Planes, pl.Ref).Visible {
		t.Error("construction and visible are independent; this plane was created hidden")
	}
	var points wire.ListWorkPointsResult
	call(t, r, s, "workPoints.list", "{}", &points)
	if !findPointConstruction(points.Points, pt.Ref) {
		t.Error("point should report construction=true")
	}
	var axes wire.ListWorkAxesResult
	call(t, r, s, "workAxes.list", "{}", &axes)
	if !findAxisConstruction(axes.Axes, ax.Ref) {
		t.Error("axis should report construction=true")
	}
}

func findPlane(planes []wire.WorkPlaneInfo, ref string) wire.WorkPlaneInfo {
	for _, p := range planes {
		if p.Ref == ref {
			return p
		}
	}
	return wire.WorkPlaneInfo{}
}

func findPointConstruction(points []wire.WorkPointInfo, ref string) bool {
	for _, p := range points {
		if p.Ref == ref {
			return p.Construction
		}
	}
	return false
}

func findAxisConstruction(axes []wire.WorkAxisInfo, ref string) bool {
	for _, a := range axes {
		if a.Ref == ref {
			return a.Construction
		}
	}
	return false
}
