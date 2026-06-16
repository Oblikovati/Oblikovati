// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestAssemblyJointKinds drives the joint-add handlers the existing test does not
// (rigid, slider, cylindrical, planar, ball — only rotational was covered).
func TestAssemblyJointKinds(t *testing.T) {
	for _, kind := range []string{"addRigid", "addSlider", "addCylindrical", "addPlanar", "addBall"} {
		t.Run(kind, func(t *testing.T) {
			r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
			occs[0].SetGrounded(true)
			edge := boxEdgeKey(t, occs[0])
			args := mustJSON(t, wire.AddJointArgs{
				A: wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: edge},
				B: wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: edge},
			})
			_ = tryCall(t, r, s, "assemblyJoints."+kind, args)
		})
	}
}

// TestDSJointDeleteAndMotions covers dsJoints.delete and the imposed-motion modes the
// existing test does not.
func TestDSJointDeleteAndMotions(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	occs[0].SetGrounded(true)
	edge := boxEdgeKey(t, occs[0])

	var added wire.DSJointResult
	call(t, r, s, "dsJoints.add", mustJSON(t, wire.AddDSJointArgs{
		A:    wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: edge},
		B:    wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: edge},
		Type: "rotational",
	}), &added)

	for _, m := range []string{"velocity", "position", "none"} {
		_ = tryCall(t, r, s, "dsJoints.setImposedMotion", mustJSON(t, wire.SetImposedMotionArgs{
			ID: added.Joint.ID, DOFIndex: 0, ImposedMotion: m, Value: 1,
		}))
	}
	call(t, r, s, "dsJoints.delete", mustJSON(t, wire.DeleteDSJointArgs{ID: added.Joint.ID}), nil)
}
