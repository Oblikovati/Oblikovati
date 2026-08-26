// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// fakeComponent is a minimal occurrence.Definition for tests — the constraint engine
// never reads component geometry (constraints carry their own Primitives), so an empty
// range box suffices.
type fakeComponent struct{}

func (fakeComponent) RangeBox() math.Box { return math.Box{} }

// recordingListener is a named fake capturing the engine's notifications.
type recordingListener struct {
	added, deleted int
	resolved       int
}

func (r *recordingListener) ConstraintAdded(contract.AssemblyConstraint)   { r.added++ }
func (r *recordingListener) ConstraintDeleted(contract.AssemblyConstraint) { r.deleted++ }
func (r *recordingListener) AssemblyResolved()                             { r.resolved++ }

func unit(t *testing.T, x, y, z float64) math.UnitVector3 {
	t.Helper()
	u, err := math.NewUnitVector3(x, y, z)
	if err != nil {
		t.Fatalf("NewUnitVector3(%v,%v,%v): %v", x, y, z, err)
	}
	return u
}

func place(occs *occurrence.Occurrences, name string, m math.Matrix4) *occurrence.Occurrence {
	return occs.AddByComponentDefinition(name, fakeComponent{}, m)
}

// ref bundles an occurrence and a primitive into a constraint geometry input (tests do not
// exercise the reference-key round-trip, so Entity is left empty).
func ref(o *occurrence.Occurrence, p Primitive) Ref {
	return Ref{Occurrence: o, Primitive: p}
}

// TestMatePositionsComponent is the headline acceptance: mating two planar faces moves the
// free component so the faces become coincident (PBI-125).
func TestMatePositionsComponent(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(0, 0, 10)))

	set := NewConstraintSet(occs, nil)
	topFace := PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))     // base's top face, +Z
	bottomFace := PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, -1)) // moving's bottom face, -Z
	set.AddMate(ref(base, topFace), ref(moving, bottomFace), 0, types.MateSolutionOpposed)

	rep := set.Solve()
	if !rep.Converged {
		t.Fatalf("solve did not converge: %+v", rep)
	}
	if z := moving.Transform().Translation().Z; stdmath.Abs(z) > 1e-6 {
		t.Errorf("moving component z = %v, want ~0 (faces coincident)", z)
	}
}

// TestMateLeavesPlanarDOF checks a single planar mate removes three DOF, leaving the free
// component three (slide in two directions + spin about the normal).
func TestMateLeavesPlanarDOF(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(0, 0, 4)))

	set := NewConstraintSet(occs, nil)
	set.AddMate(ref(base, PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))), ref(moving, PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, -1))), 0, types.MateSolutionOpposed)
	rep := set.Solve()

	if rep.DegreesOfFreedom != 3 {
		t.Errorf("total DOF = %d, want 3", rep.DegreesOfFreedom)
	}
	if got := dofOf(rep, moving.ID()); got != 3 {
		t.Errorf("moving occurrence DOF = %d, want 3", got)
	}
	if got := dofOf(rep, base.ID()); got != 0 {
		t.Errorf("grounded base DOF = %d, want 0", got)
	}
}

// TestOverConstraintFlagged checks two coincident mates over the same faces are reported
// as redundant (over-constrained → warning health), not a crash.
func TestOverConstraintFlagged(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(0, 0, 4)))

	set := NewConstraintSet(occs, nil)
	for range 2 {
		set.AddMate(ref(base, PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))), ref(moving, PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, -1))), 0, types.MateSolutionOpposed)
	}
	rep := set.Solve()

	if rep.Redundant == 0 {
		t.Errorf("redundant rows = 0, want > 0 for duplicate mates")
	}
	if rep.Status != types.HealthWarning {
		t.Errorf("status = %v, want warning (over-constrained but solvable)", rep.Status)
	}
}

// TestPointMateCoincides checks a point-point mate makes two vertices coincide.
func TestPointMateCoincides(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(7, -3, 9)))

	set := NewConstraintSet(occs, nil)
	set.AddMate(ref(base, PointPrimitive(math.P3(1, 2, 3))), ref(moving, PointPrimitive(math.P3(0, 0, 0))), 0, types.MateSolutionOpposed)
	set.Solve()

	if got := moving.Transform().TransformPoint(math.P3(0, 0, 0)); !got.IsEqualTo(math.P3(1, 2, 3), 1e-6) {
		t.Errorf("moving vertex = %+v, want (1,2,3)", got)
	}
}

// TestListenerNotified checks add/delete/resolve notifications reach the listener.
func TestListenerNotified(t *testing.T) {
	occs := occurrence.NewOccurrences()
	a := place(occs, "a:1", math.Identity4())
	a.SetGrounded(true)
	b := place(occs, "b:1", math.Translation4(math.V3(0, 0, 2)))

	rec := &recordingListener{}
	set := NewConstraintSet(occs, rec)
	m := set.AddMate(ref(a, PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))), ref(b, PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, -1))), 0, types.MateSolutionOpposed)
	set.Solve()
	if !set.Delete(m.ID()) {
		t.Fatal("Delete returned false for a known constraint")
	}
	if rec.added != 1 || rec.deleted != 1 || rec.resolved != 1 {
		t.Errorf("notifications = %+v, want added=1 deleted=1 resolved=1", *rec)
	}
}

// TestSuppressedConstraintIgnored checks a suppressed constraint contributes no residual,
// so the moving component keeps all six DOF.
func TestSuppressedConstraintIgnored(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(0, 0, 4)))

	set := NewConstraintSet(occs, nil)
	m := set.AddMate(ref(base, PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))), ref(moving, PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, -1))), 0, types.MateSolutionOpposed)
	m.SetSuppressed(true)
	rep := set.Solve()

	if got := dofOf(rep, moving.ID()); got != 6 {
		t.Errorf("moving DOF with suppressed mate = %d, want 6", got)
	}
	if rep.Constraints != 0 {
		t.Errorf("active constraint count = %d, want 0", rep.Constraints)
	}
}

// TestLimitsRoundTrip checks set/clear limits and the clamp.
func TestLimitsRoundTrip(t *testing.T) {
	occs := occurrence.NewOccurrences()
	a := place(occs, "a:1", math.Identity4())
	b := place(occs, "b:1", math.Identity4())
	set := NewConstraintSet(occs, nil)
	m := set.AddMate(ref(a, PointPrimitive(math.P3(0, 0, 0))), ref(b, PointPrimitive(math.P3(0, 0, 0))), 0, types.MateSolutionOpposed)

	lim := NewLimits(-1, true, 2, true, 0, false)
	if err := set.SetLimits(m.ID(), lim); err != nil {
		t.Fatalf("SetLimits: %v", err)
	}
	if m.Limits() == nil {
		t.Fatal("Limits() = nil after SetLimits")
	}
	if got := lim.clamp(5); got != 2 {
		t.Errorf("clamp(5) = %v, want 2", got)
	}
	if got := lim.clamp(-3); got != -1 {
		t.Errorf("clamp(-3) = %v, want -1", got)
	}
	if err := set.SetLimits(99, lim); err == nil {
		t.Error("SetLimits on unknown id returned nil error")
	}
}

// TestForOccurrenceView checks the per-occurrence enumerator returns only the constraints
// touching that occurrence.
func TestForOccurrenceView(t *testing.T) {
	occs := occurrence.NewOccurrences()
	a := place(occs, "a:1", math.Identity4())
	b := place(occs, "b:1", math.Identity4())
	c := place(occs, "c:1", math.Identity4())
	set := NewConstraintSet(occs, nil)
	set.AddMate(ref(a, PointPrimitive(math.P3(0, 0, 0))), ref(b, PointPrimitive(math.P3(0, 0, 0))), 0, types.MateSolutionOpposed)

	if set.ForOccurrence(a).Count() != 1 {
		t.Errorf("ForOccurrence(a).Count() = %d, want 1", set.ForOccurrence(a).Count())
	}
	if set.ForOccurrence(c).Count() != 0 {
		t.Errorf("ForOccurrence(c).Count() = %d, want 0", set.ForOccurrence(c).Count())
	}
}

// dofOf returns the reported DOF for an occurrence id in a solve report.
func dofOf(rep SolveReport, id uint64) int {
	for _, o := range rep.Occurrences {
		if o.Occurrence == id {
			return o.DegreesOfFreedom
		}
	}
	return -1
}
