// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/occurrence"
)

// G1 router-coverage tests: exercise the previously-untested query/list/status/remove
// sibling handlers of the converted assembly surfaces (contact, replication, joints,
// derive). Setup helpers are prefixed asmcov to stay cluster-unique.

// asmcovContactSet creates a named contact set on the active assembly and returns its id.
func asmcovContactSet(t *testing.T, r *Router, s *app.Session, name string) uint64 {
	t.Helper()
	var cs wire.ContactSetResult
	call(t, r, s, "contactSets.create", mustJSON(t, wire.CreateContactSetArgs{Name: name}), &cs)
	return cs.ContactSet.ID
}

// TestAsmcovContactSetsList lists the created contact sets in creation order (contactSets.list).
func TestAsmcovContactSetsList(t *testing.T) {
	r, s, _, _ := assemblySessionWithBoxes(t, 0, 2)
	asmcovContactSet(t, r, s, "alpha")
	asmcovContactSet(t, r, s, "beta")

	var list wire.ContactSetsResult
	call(t, r, s, "contactSets.list", `{}`, &list)
	if len(list.ContactSets) != 2 || list.ContactSets[0].Name != "alpha" || list.ContactSets[1].Name != "beta" {
		t.Fatalf("contact sets = %+v, want [alpha beta]", list.ContactSets)
	}
}

// TestAsmcovContactSetsRemoveMember drops one member and leaves the other (contactSets.removeMember).
func TestAsmcovContactSetsRemoveMember(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 2)
	id := asmcovContactSet(t, r, s, "g")
	call(t, r, s, "contactSets.addMember", mustJSON(t, wire.ContactMemberArgs{Set: id, Occurrence: occs[0].ID()}), nil)
	call(t, r, s, "contactSets.addMember", mustJSON(t, wire.ContactMemberArgs{Set: id, Occurrence: occs[1].ID()}), nil)

	var removed wire.ContactSetResult
	call(t, r, s, "contactSets.removeMember", mustJSON(t, wire.ContactMemberArgs{Set: id, Occurrence: occs[0].ID()}), &removed)
	if len(removed.ContactSet.Members) != 1 || removed.ContactSet.Members[0] != occs[1].ID() {
		t.Fatalf("members after remove = %v, want just occurrence %d", removed.ContactSet.Members, occs[1].ID())
	}
}

// TestAsmcovContactRemoveMemberUnknownSet pins the removeMember error path (unknown set id).
func TestAsmcovContactRemoveMemberUnknownSet(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 2)
	args := mustJSON(t, wire.ContactMemberArgs{Set: 99999, Occurrence: occs[0].ID()})
	if _, err := r.Handle(s, "contactSets.removeMember", []byte(args)); err == nil {
		t.Error("removeMember from an unknown set should fail")
	}
}

// TestAsmcovContactSolverStatus reads the solver status before (disabled/0) and after
// creating a set and toggling it on (enabled/1) — contactSolver.status.
func TestAsmcovContactSolverStatus(t *testing.T) {
	r, s, _, _ := assemblySessionWithBoxes(t, 0, 2)
	var initial wire.ContactSolverResult
	call(t, r, s, "contactSolver.status", `{}`, &initial)
	if initial.Solver.Enabled || initial.Solver.SetCount != 0 {
		t.Fatalf("default solver = %+v, want disabled with 0 sets", initial.Solver)
	}

	asmcovContactSet(t, r, s, "g")
	call(t, r, s, "contactSolver.setEnabled", mustJSON(t, wire.ContactSolverEnableArgs{Enabled: true}), nil)
	var on wire.ContactSolverResult
	call(t, r, s, "contactSolver.status", `{}`, &on)
	if !on.Solver.Enabled || on.Solver.SetCount != 1 {
		t.Errorf("solver status = %+v, want enabled with 1 set", on.Solver)
	}
}

// TestAsmcovRectangularPattern replicates a seed in a 2x2 grid: three new occurrences
// beyond the seed, growing the tree to four (rectangularArrangement).
func TestAsmcovRectangularPattern(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0)
	var created wire.NewOccurrencesResult
	args := mustJSON(t, wire.CreatePatternArgs{
		Seed: occs[0].ID(), Kind: "rectangular",
		Direction1: [3]float64{1, 0, 0}, Spacing1: 2, Count1: 2,
		Direction2: [3]float64{0, 1, 0}, Spacing2: 2, Count2: 2,
	})
	call(t, r, s, "assembly.patternCreate", args, &created)
	if len(created.Created) != 3 {
		t.Fatalf("2x2 rectangular pattern created %d occurrences, want 3", len(created.Created))
	}

	var tree wire.OccurrencesResult
	call(t, r, s, "assembly.occurrences", `{}`, &tree)
	if len(tree.Occurrences) != 4 {
		t.Errorf("tree = %d occurrences, want 4 after the 2x2 pattern", len(tree.Occurrences))
	}
}

// TestAsmcovRectangularPatternRejectsBadInput pins the three rectangularArrangement error
// branches: a zero first/second direction and a zero grid count.
func TestAsmcovRectangularPatternRejectsBadInput(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0)
	rect := func(extra string) []byte {
		return []byte(fmt.Sprintf(`{"seed":%d,"kind":"rectangular",%s}`, occs[0].ID(), extra))
	}
	if _, err := r.Handle(s, "assembly.patternCreate", rect(`"direction1":[0,0,0],"count1":2,"direction2":[0,1,0],"count2":2`)); err == nil {
		t.Error("a zero direction1 should fail")
	}
	if _, err := r.Handle(s, "assembly.patternCreate", rect(`"direction1":[1,0,0],"count1":2,"direction2":[0,0,0],"count2":2`)); err == nil {
		t.Error("a zero direction2 should fail")
	}
	if _, err := r.Handle(s, "assembly.patternCreate", rect(`"direction1":[1,0,0],"count1":0,"direction2":[0,1,0],"count2":2`)); err == nil {
		t.Error("a zero count1 should fail")
	}
}

// asmcovDSJoint adds a DS joint of the given kind between two boxes' shared edge axis.
func asmcovDSJoint(t *testing.T, r *Router, s *app.Session, occs []*occurrence.Occurrence, kind string) wire.DSJointResult {
	t.Helper()
	edge := boxEdgeKey(t, occs[0])
	var added wire.DSJointResult
	call(t, r, s, "dsJoints.add", mustJSON(t, wire.AddDSJointArgs{
		A: wire.ConstraintGeomRef{Occurrence: occs[0].ID(), Entity: edge},
		B: wire.ConstraintGeomRef{Occurrence: occs[1].ID(), Entity: edge}, Type: kind,
	}), &added)
	return added
}

// TestAsmcovDSJointTypes exercises every dsJointType mapping branch, including the
// default (an unknown kind maps to rigid).
func TestAsmcovDSJointTypes(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	want := map[string]string{
		"rotational": "rotational", "prismatic": "prismatic",
		"planar": "planar", "spherical": "spherical", "bogus": "rigid",
	}
	for in, exp := range want {
		if got := asmcovDSJoint(t, r, s, occs, in).Joint.Type; got != exp {
			t.Errorf("dsJoint kind %q rendered type %q, want %q", in, got, exp)
		}
	}
}

// TestAsmcovDSJointImposedMotionAndDelete drives a DOF, then deletes the joint
// (dsJoints.setImposedMotion driven branch + dsJoints.delete).
func TestAsmcovDSJointImposedMotionAndDelete(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0, 5)
	added := asmcovDSJoint(t, r, s, occs, "cylindrical")

	var driven wire.DSJointResult
	call(t, r, s, "dsJoints.setImposedMotion", mustJSON(t, wire.SetImposedMotionArgs{ID: added.Joint.ID, DOFIndex: 0, ImposedMotion: "driven", Value: 2}), &driven)
	if d := driven.Joint.DegreesOfFreedom[0]; d.ImposedMotion != "driven" || d.Value != 2 {
		t.Fatalf("DOF 0 = %+v, want driven with value 2", d)
	}

	var afterDel wire.DSJointsResult
	call(t, r, s, "dsJoints.delete", mustJSON(t, wire.DeleteDSJointArgs{ID: added.Joint.ID}), &afterDel)
	if len(afterDel.Joints) != 0 {
		t.Errorf("after delete = %d DS joints, want 0", len(afterDel.Joints))
	}
}

// TestAsmcovDSJointErrors pins the DS-joint mutator error paths (unknown ids).
func TestAsmcovDSJointErrors(t *testing.T) {
	r, s, _, _ := assemblySessionWithBoxes(t, 0, 5)
	if _, err := r.Handle(s, "dsJoints.delete", []byte(mustJSON(t, wire.DeleteDSJointArgs{ID: 99999}))); err == nil {
		t.Error("deleting an unknown DS joint should fail")
	}
	if _, err := r.Handle(s, "dsJoints.setImposedMotion", []byte(mustJSON(t, wire.SetImposedMotionArgs{ID: 99999}))); err == nil {
		t.Error("imposed motion on an unknown DS joint should fail")
	}
}

// asmcovShrinkwrap makes an active part with an open two-box source assembly to shrinkwrap.
func asmcovShrinkwrap(t *testing.T) (*Router, *app.Session, uint64) {
	t.Helper()
	r, s := emptyPartSession(t)
	src := addAssemblyDoc(t, s, "src.obk", 0, 10)
	return r, s, uint64(src.ID())
}

// asmcovShrinkwrapKind shrinkwraps the source with the given styles and returns the feature kind.
func asmcovShrinkwrapKind(t *testing.T, r *Router, s *app.Session, src uint64, remove, envelope int) string {
	t.Helper()
	var detail wire.FeatureDetailResult
	args := fmt.Sprintf(`{"source":%d,"removeStyle":%d,"envelopeStyle":%d}`, src, remove, envelope)
	call(t, r, s, "assembly.shrinkwrapCreate", args, &detail)
	return detail.Feature.Kind
}

// TestAsmcovShrinkwrapStyles exercises the removeStyle/envelopeStyle mapping branches:
// remove-small + per-part envelope, and remove-internal + no envelope.
func TestAsmcovShrinkwrapStyles(t *testing.T) {
	r1, s1, src1 := asmcovShrinkwrap(t)
	if k := asmcovShrinkwrapKind(t, r1, s1, src1, 1, 1); k != "shrinkwrap" {
		t.Errorf("remove-small/per-part kind = %q, want shrinkwrap", k)
	}
	r2, s2, src2 := asmcovShrinkwrap(t)
	if k := asmcovShrinkwrapKind(t, r2, s2, src2, 2, 0); k != "shrinkwrap" {
		t.Errorf("remove-internal/no-envelope kind = %q, want shrinkwrap", k)
	}
}

// TestAsmcovShrinkwrapRejectsUnknownStyles pins the removeStyle/envelopeStyle error branches.
func TestAsmcovShrinkwrapRejectsUnknownStyles(t *testing.T) {
	r, s, src := asmcovShrinkwrap(t)
	if _, err := r.Handle(s, "assembly.shrinkwrapCreate", []byte(fmt.Sprintf(`{"source":%d,"removeStyle":99}`, src))); err == nil {
		t.Error("an unknown removeStyle should fail")
	}
	if _, err := r.Handle(s, "assembly.shrinkwrapCreate", []byte(fmt.Sprintf(`{"source":%d,"envelopeStyle":99}`, src))); err == nil {
		t.Error("an unknown envelopeStyle should fail")
	}
}
