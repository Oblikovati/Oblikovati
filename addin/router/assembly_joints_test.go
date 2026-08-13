// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/occurrence"
)

// boxEdgeKey returns the reference key of the box component's first edge — a straight edge,
// hence a joint axis.
func boxEdgeKey(t *testing.T, occ *occurrence.Occurrence) string {
	t.Helper()
	def := occ.Definition().(interface{ SurfaceBodies() *topo.SurfaceBodies })
	edges := def.SurfaceBodies().All()[0].Edges()
	if len(edges) == 0 {
		t.Fatal("box body has no edges")
	}
	return string(edges[0].ReferenceKey())
}

// TestAssemblyRotationalJointOverWire adds a rotational joint between two boxes' shared edge
// axis over the wire, checks the free component is left one DOF, then lists, flips, limits,
// and deletes it (#359/#364).
func TestAssemblyRotationalJointOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	occs[0].SetGrounded(true)
	edge := boxEdgeKey(t, occs[0])

	var added wire.AssemblyJointResult
	args := mustJSON(t, wire.AddJointArgs{
		A: wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: edge},
		B: wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: edge},
	})
	call(t, r, s, "assemblyJoints.addRotational", args, &added)
	if added.Joint.Type != "rotational" || added.Joint.DegreesOfFreedom != 1 {
		t.Fatalf("added joint = %+v, want rotational with 1 DOF", added.Joint)
	}

	var list wire.AssemblyJointsResult
	call(t, r, s, "assemblyJoints.list", `{}`, &list)
	if len(list.Joints) != 1 || list.Joints[0].ID != added.Joint.ID {
		t.Fatalf("list = %+v, want the one joint", list.Joints)
	}

	var flipped wire.AssemblyJointResult
	call(t, r, s, "assemblyJoints.setFlip", mustJSON(t, wire.SetJointFlipArgs{ID: added.Joint.ID, Flip: true}), &flipped)
	if !flipped.Joint.Flip {
		t.Errorf("setFlip did not flip the joint: %+v", flipped.Joint)
	}

	var limited wire.AssemblyJointResult
	call(t, r, s, "assemblyJoints.setLimits", mustJSON(t, wire.SetJointLimitsArgs{
		ID: added.Joint.ID, Limits: wire.JointLimits{HasAngularMax: true, AngularMax: 1.5},
	}), &limited)
	if limited.Joint.Limits == nil || !limited.Joint.Limits.HasAngularMax || limited.Joint.Limits.AngularMax != 1.5 {
		t.Errorf("setLimits = %+v, want angular max 1.5", limited.Joint.Limits)
	}

	var afterDel wire.AssemblyJointsResult
	call(t, r, s, "assemblyJoints.delete", mustJSON(t, wire.DeleteJointArgs{ID: added.Joint.ID}), &afterDel)
	if len(afterDel.Joints) != 0 {
		t.Errorf("after delete = %+v, want empty set", afterDel.Joints)
	}
}

// TestDSJointOverWire drives the DS-joint surface: add, lock a DOF, and list.
func TestDSJointOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	edge := boxEdgeKey(t, occs[0])

	var added wire.DSJointResult
	call(t, r, s, "dsJoints.add", mustJSON(t, wire.AddDSJointArgs{
		A:    wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: edge},
		B:    wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: edge},
		Type: "cylindrical",
	}), &added)
	if len(added.Joint.DegreesOfFreedom) != 2 {
		t.Fatalf("DS cylindrical joint = %+v, want 2 DOF", added.Joint)
	}

	var locked wire.DSJointResult
	call(t, r, s, "dsJoints.setImposedMotion", mustJSON(t, wire.SetImposedMotionArgs{
		ID: added.Joint.ID, DOFIndex: 0, ImposedMotion: "locked",
	}), &locked)
	if locked.Joint.DegreesOfFreedom[0].ImposedMotion != "locked" {
		t.Errorf("DOF 0 imposed motion = %q, want locked", locked.Joint.DegreesOfFreedom[0].ImposedMotion)
	}

	var list wire.DSJointsResult
	call(t, r, s, "dsJoints.list", `{}`, &list)
	if len(list.Joints) != 1 {
		t.Errorf("DS joint list = %d, want 1", len(list.Joints))
	}
}

// TestAssemblyJointUnknownKeyRejected checks an unresolvable joint origin is a clean error.
func TestAssemblyJointUnknownKeyRejected(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	args := mustJSON(t, wire.AddJointArgs{
		A: wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: "bogus"},
		B: wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: "bogus"},
	})
	if _, err := r.Handle(s, "assemblyJoints.addRotational", []byte(args)); err == nil {
		t.Error("addRotational with an unknown reference key should fail")
	}
}

// TestAssemblyJointSetOriginOverWire adds a joint, offsets its first origin over the wire, and checks
// the joint info reports the offset mode and X/Y, and that an unknown mode is rejected (#1973).
func TestAssemblyJointSetOriginOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	occs[0].SetGrounded(true)
	edge := boxEdgeKey(t, occs[0])

	var added wire.AssemblyJointResult
	call(t, r, s, "assemblyJoints.addRigid", mustJSON(t, wire.AddJointArgs{
		A: wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: edge},
		B: wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: edge},
	}), &added)

	var offset wire.AssemblyJointResult
	call(t, r, s, "assemblyJoints.setOrigin", mustJSON(t, wire.SetJointOriginArgs{
		ID: added.Joint.ID, Which: 1, Mode: "offset", XOffset: 2, YOffset: 3,
	}), &offset)
	if offset.Joint.OriginOneMode != "offset" || offset.Joint.OriginOneXOffset != 2 || offset.Joint.OriginOneYOffset != 3 {
		t.Errorf("offset origin = %+v, want mode offset (2,3)", offset.Joint)
	}

	if _, err := r.Handle(s, "assemblyJoints.setOrigin", []byte(mustJSON(t, wire.SetJointOriginArgs{
		ID: added.Joint.ID, Which: 1, Mode: "midplane",
	}))); err == nil {
		t.Error("setOrigin with an unknown mode should fail")
	}
}
