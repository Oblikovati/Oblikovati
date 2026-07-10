// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestSketch3DEquationCurveCylindricalOverWire creates a cylindrical equation curve and checks
// enumeration reports its coordinate system (#1846).
func TestSketch3DEquationCurveCylindricalOverWire(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	call(t, r, s, "sketch3d.addEntity",
		`{"sketchIndex":0,"kind":"equationCurve","xExpr":"2","yExpr":"t","zExpr":"1","t0":0,"t1":6.28,"coordinateSystem":"cylindrical"}`,
		&wire.AddSketch3DEntityResult{})

	var ents wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &ents)
	if len(ents.Entities) != 1 || ents.Entities[0].CoordinateSystem != "cylindrical" {
		t.Fatalf("enumerated entities = %+v, want one cylindrical equation curve", ents.Entities)
	}
}

// TestSketch3DEquationCurveDefaultOmitsCoordinateSystem: a Cartesian curve reports no coordinate
// system (#1846 AC2).
func TestSketch3DEquationCurveDefaultOmitsCoordinateSystem(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	call(t, r, s, "sketch3d.addEntity",
		`{"sketchIndex":0,"kind":"equationCurve","xExpr":"cos(t)","yExpr":"sin(t)","zExpr":"t","t0":0,"t1":6.28}`,
		&wire.AddSketch3DEntityResult{})

	var ents wire.EnumerateEntities3DResult
	call(t, r, s, "sketch3d.entities", `{"sketchIndex":0}`, &ents)
	if len(ents.Entities) != 1 || ents.Entities[0].CoordinateSystem != "" {
		t.Errorf("cartesian curve reported coordinateSystem = %q, want empty", ents.Entities[0].CoordinateSystem)
	}
}

// TestSketch3DEquationCurveUnknownCoordinateSystemErrors rejects a bad selector cleanly (#1846).
func TestSketch3DEquationCurveUnknownCoordinateSystemErrors(t *testing.T) {
	r, s := emptyPartSession(t)
	call(t, r, s, "sketch3d.create", `{}`, &wire.CreateSketch3DResult{})
	if err := tryCall(t, r, s, "sketch3d.addEntity",
		`{"sketchIndex":0,"kind":"equationCurve","xExpr":"t","yExpr":"t","zExpr":"t","t0":0,"t1":1,"coordinateSystem":"toroidal"}`); err == nil {
		t.Error("an unknown coordinate system should error")
	}
}
