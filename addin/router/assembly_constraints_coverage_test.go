// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestAssemblyConstraintKinds drives every assembly-constraint add handler the existing
// test does not (flush/angle/tangent/insert/symmetry + the motion constraints), each over
// two box faces. Box geometry cannot satisfy every constraint, so a clean error is fine —
// the point is exercising each handler's decode and constraint-building path.
func TestAssemblyConstraintKinds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		method string
		args   func(a, b wire.ConstraintGeomRef) any
	}{
		{"assemblyConstraints.addFlush", func(a, b wire.ConstraintGeomRef) any { return wire.AddFlushArgs{A: a, B: b, Offset: 1} }},
		{"assemblyConstraints.addAngle", func(a, b wire.ConstraintGeomRef) any { return wire.AddAngleArgs{A: a, B: b, Angle: 45} }},
		{"assemblyConstraints.addTangent", func(a, b wire.ConstraintGeomRef) any { return wire.AddTangentArgs{A: a, B: b} }},
		{"assemblyConstraints.addInsert", func(a, b wire.ConstraintGeomRef) any { return wire.AddInsertArgs{A: a, B: b, Offset: 1} }},
		{"assemblyConstraints.addSymmetry", func(a, b wire.ConstraintGeomRef) any { return wire.AddSymmetryArgs{A: a, B: b, Plane: a} }},
		{"assemblyConstraints.addRotateRotate", func(a, b wire.ConstraintGeomRef) any { return wire.AddRotateRotateArgs{A: a, B: b, Ratio: 2} }},
		{"assemblyConstraints.addRotateTranslate", func(a, b wire.ConstraintGeomRef) any { return wire.AddRotateTranslateArgs{A: a, B: b, Distance: 2} }},
		{"assemblyConstraints.addTranslateTranslate", func(a, b wire.ConstraintGeomRef) any { return wire.AddTranslateTranslateArgs{A: a, B: b, Ratio: 2} }},
		{"assemblyConstraints.addTransitional", func(a, b wire.ConstraintGeomRef) any { return wire.AddTransitionalArgs{A: a, B: b} }},
		{"assemblyConstraints.addCustom", func(a, b wire.ConstraintGeomRef) any {
			return wire.AddCustomArgs{A: a, B: b, Kind: "custom", Params: []float64{1}}
		}},
	}
	for _, c := range cases {
		t.Run(c.method, func(t *testing.T) {
			r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
			a := wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: topBoxFaceKey(t, occs[0])}
			b := wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: bottomBoxFaceKey(t, occs[1])}
			_ = tryCall(t, r, s, c.method, mustJSON(t, c.args(a, b)))
		})
	}
}

// TestAssemblyConstraintSetLimits captures a mate, then drives setLimits + solve + health
// against it.
func TestAssemblyConstraintSetLimits(t *testing.T) {
	t.Parallel()
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	occs[0].SetGrounded(true)
	a := wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: topBoxFaceKey(t, occs[0])}
	b := wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: bottomBoxFaceKey(t, occs[1])}

	var added wire.ConstraintResult
	call(t, r, s, "assemblyConstraints.addMate", mustJSON(t, wire.AddMateArgs{A: a, B: b}), &added)
	_ = tryCall(t, r, s, "assemblyConstraints.setLimits", mustJSON(t, wire.SetConstraintLimitsArgs{
		ID: added.Constraint.ID, Limits: wire.ConstraintLimits{},
	}))
	call(t, r, s, "assemblyConstraints.solve", `{}`, nil)
	call(t, r, s, "assemblyConstraints.health", `{}`, nil)
}
