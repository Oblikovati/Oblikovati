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
