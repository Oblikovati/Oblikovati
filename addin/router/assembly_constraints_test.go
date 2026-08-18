// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/occurrence"
)

// bottomBoxFaceKey returns the reference key of the box component's bottom face (lowest z),
// whose −z normal mates opposed to another box's +z top face.
func bottomBoxFaceKey(t *testing.T, occ *occurrence.Occurrence) string {
	t.Helper()
	def := occ.Definition().(interface{ SurfaceBodies() *topo.SurfaceBodies })
	var bottom *topo.Face
	for _, f := range def.SurfaceBodies().All()[0].Faces() {
		if bottom == nil || f.RangeBox().Center().Z < bottom.RangeBox().Center().Z {
			bottom = f
		}
	}
	return string(bottom.ReferenceKey())
}

// TestAssemblyMateOverWire mates a free box's bottom face onto a grounded box's top face
// over the wire and checks the component is repositioned, listed, and deletable, and that
// the health report shows the remaining planar DOF (#358/#363).
func TestAssemblyMateOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	occs[0].SetGrounded(true)
	topKey := topBoxFaceKey(t, occs[0])
	botKey := bottomBoxFaceKey(t, occs[1])

	var added wire.ConstraintResult
	args := mustJSON(t, wire.AddMateArgs{
		A:        wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: topKey},
		B:        wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: botKey},
		Solution: "opposed",
	})
	call(t, r, s, "assemblyConstraints.addMate", args, &added)
	if added.Constraint.Type != "mate" || added.Constraint.A.Occurrence != occs[0].ID() {
		t.Fatalf("added = %+v, want a mate referencing the grounded box", added.Constraint)
	}
	// The grounded box's top face sits at z=1; the free box's bottom face mates onto it.
	if z := occs[1].Transform().Translation().Z; stdmath.Abs(z-1) > 1e-6 {
		t.Errorf("free box z = %v, want 1 (bottom face on the grounded top face)", z)
	}

	var list wire.ConstraintsResult
	call(t, r, s, "assemblyConstraints.list", `{}`, &list)
	if len(list.Constraints) != 1 || list.Constraints[0].ID != added.Constraint.ID {
		t.Fatalf("list = %+v, want the one mate", list.Constraints)
	}

	var health wire.AssemblyHealthResult
	call(t, r, s, "assemblyConstraints.health", `{}`, &health)
	if health.Status != "under-constrained" || health.DegreesOfFreedom != 3 {
		t.Errorf("health = %+v, want under-constrained with 3 DOF", health)
	}

	var afterDel wire.ConstraintsResult
	call(t, r, s, "assemblyConstraints.delete", mustJSON(t, wire.DeleteAssemblyConstraintArgs{ID: added.Constraint.ID}), &afterDel)
	if len(afterDel.Constraints) != 0 {
		t.Errorf("after delete = %+v, want empty set", afterDel.Constraints)
	}
}

// TestAssemblySnapConstrainOverWire grip-snaps a free box's bottom face onto a grounded box's top
// face over the wire WITHOUT naming the constraint: the host infers a mate (two opposed planar faces)
// and repositions the component, exactly as the explicit mate does (#794).
func TestAssemblySnapConstrainOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	occs[0].SetGrounded(true)
	topKey := topBoxFaceKey(t, occs[0])
	botKey := bottomBoxFaceKey(t, occs[1])

	var added wire.ConstraintResult
	args := mustJSON(t, wire.SnapConstraintArgs{
		A: wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: topKey},
		B: wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: botKey},
	})
	call(t, r, s, "assemblyConstraints.snap", args, &added)
	if added.Constraint.Type != "mate" {
		t.Fatalf("snap inferred %q, want mate (two opposed planar faces)", added.Constraint.Type)
	}
	if z := occs[1].Transform().Translation().Z; stdmath.Abs(z-1) > 1e-6 {
		t.Errorf("snapped box z = %v, want 1 (bottom face on the grounded top face)", z)
	}
	// The prefer override (snapping the same faces as a flush) is covered by the model
	// TestGripSnapPreferOverrides and the bridge grip-snap e2e.
}

// TestAssemblySolveReportsFreeDOF checks the solve endpoint reports per-occurrence DOF for
// a grounded + free pair with no constraints: the free box keeps six DOF, the grounded box
// none.
func TestAssemblySolveReportsFreeDOF(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	occs[0].SetGrounded(true)

	var health wire.AssemblyHealthResult
	call(t, r, s, "assemblyConstraints.solve", `{}`, &health)
	if health.Constraints != 0 {
		t.Errorf("active constraints = %d, want 0", health.Constraints)
	}
	if dofOfOccurrence(health, occs[1].ID()) != 6 {
		t.Errorf("free box DOF = %d, want 6", dofOfOccurrence(health, occs[1].ID()))
	}
	if dofOfOccurrence(health, occs[0].ID()) != 0 {
		t.Errorf("grounded box DOF = %d, want 0", dofOfOccurrence(health, occs[0].ID()))
	}
}

// TestAssemblyConstraintUnknownKeyRejected checks an unresolvable geometry reference is a
// clean error, not a panic.
func TestAssemblyConstraintUnknownKeyRejected(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	args := mustJSON(t, wire.AddMateArgs{
		A: wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: "bogus"},
		B: wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: "bogus"},
	})
	if _, err := r.Handle(s, "assemblyConstraints.addMate", []byte(args)); err == nil {
		t.Error("addMate with an unknown reference key should fail")
	}
}

// TestAssemblyReferenceVectorAngleNeedsThirdEntity: requesting the reference-vector angle solution
// without a referenceVector entity is rejected with a clear error, while supplying one succeeds
// (#1972).
func TestAssemblyReferenceVectorAngleNeedsThirdEntity(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	occs[0].SetGrounded(true)
	topKey := topBoxFaceKey(t, occs[0])
	botKey := bottomBoxFaceKey(t, occs[1])

	missing := mustJSON(t, wire.AddAngleArgs{
		A:        wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: topKey},
		B:        wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: botKey},
		Angle:    stdmath.Pi / 4,
		Solution: "reference-vector",
	})
	if _, err := r.Handle(s, "assemblyConstraints.addAngle", []byte(missing)); err == nil {
		t.Error("reference-vector angle without a referenceVector entity should fail")
	}

	withAxis := mustJSON(t, wire.AddAngleArgs{
		A:               wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: topKey},
		B:               wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: botKey},
		Angle:           stdmath.Pi / 4,
		Solution:        "reference-vector",
		ReferenceVector: wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: topKey},
	})
	var added wire.ConstraintResult
	call(t, r, s, "assemblyConstraints.addAngle", withAxis, &added)
	if added.Constraint.Type != "angle" {
		t.Errorf("added = %+v, want an angle constraint", added.Constraint)
	}
}

// dofOfOccurrence returns the reported DOF for an occurrence id in a health result.
func dofOfOccurrence(h wire.AssemblyHealthResult, id uint64) int {
	for _, o := range h.Occurrences {
		if o.Occurrence == id {
			return o.DegreesOfFreedom
		}
	}
	return -1
}

// TestAssemblyDOFSplitOverWire: the solve report splits a free occurrence's six DOF into three
// translational and three rotational, with a DOF centre (#1980).
func TestAssemblyDOFSplitOverWire(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	occs[0].SetGrounded(true)

	var health wire.AssemblyHealthResult
	call(t, r, s, "assemblyConstraints.solve", `{}`, &health)
	free := dofInfoOf(health, occs[1].ID())
	if free.TranslationCount != 3 || free.RotationCount != 3 {
		t.Fatalf("free box DOF split = %d/%d, want 3/3", free.TranslationCount, free.RotationCount)
	}
	if free.TranslationCount+free.RotationCount != free.DegreesOfFreedom {
		t.Errorf("split %d+%d ≠ scalar DOF %d", free.TranslationCount, free.RotationCount, free.DegreesOfFreedom)
	}
	grounded := dofInfoOf(health, occs[0].ID())
	if grounded.TranslationCount != 0 || grounded.RotationCount != 0 {
		t.Errorf("grounded box split = %d/%d, want 0/0", grounded.TranslationCount, grounded.RotationCount)
	}
}

// dofInfoOf returns the DOF info reported for an occurrence id.
func dofInfoOf(h wire.AssemblyHealthResult, id uint64) wire.OccurrenceDOFInfo {
	for _, o := range h.Occurrences {
		if o.Occurrence == id {
			return o
		}
	}
	return wire.OccurrenceDOFInfo{}
}
