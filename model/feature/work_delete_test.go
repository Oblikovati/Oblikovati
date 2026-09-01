// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
)

// TestDeleteWorkPlaneTombstones: deleting a user plane flags it deleted and makes its reference
// stop resolving, while its slot (and every other datum's slot) stays put (#1855).
func TestDeleteWorkPlaneTombstones(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	pl := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 1 })
	before := g.WorkPlanes().Count()

	removed, err := g.DeleteWork(pl.Key(), true)
	if err != nil {
		t.Fatalf("DeleteWork: %v", err)
	}
	if len(removed) != 1 || removed[0] != pl.Key() {
		t.Errorf("removed = %v, want just %q", removed, pl.Key())
	}
	if !pl.Deleted() {
		t.Error("plane should be flagged deleted")
	}
	if g.WorkPlanes().Count() != before {
		t.Errorf("slot count changed to %d, want %d (tombstone keeps the slot)", g.WorkPlanes().Count(), before)
	}
	if _, err := g.WorkPlaneByRef(pl.Key()); err == nil {
		t.Error("a deleted plane should not resolve by ref")
	}
	if _, err := g.plane(pl.Key()); err == nil {
		t.Error("a deleted plane ref should not resolve as a plane input")
	}
}

// TestDeleteCascadesToDependents: with retainDependents=false, deleting a point deletes the axis
// built through it (#1855).
func TestDeleteCascadesToDependents(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	pt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 5) })
	ax := g.WorkAxes().AddByTwoPoints(pt.Key(), OriginCenter)

	removed, err := g.DeleteWork(pt.Key(), false)
	if err != nil {
		t.Fatalf("DeleteWork: %v", err)
	}
	if !pt.Deleted() || !ax.Deleted() {
		t.Errorf("both point and its dependent axis should be deleted (point=%v axis=%v)", pt.Deleted(), ax.Deleted())
	}
	if len(removed) != 2 {
		t.Errorf("removed = %v, want the point and the axis", removed)
	}
}

// TestDeleteRetainDependentsSickensDependent: with retainDependents=true, only the named datum is
// deleted; a dependent is left in place and goes unhealthy on recompute (its ref no longer
// resolves) (#1855).
func TestDeleteRetainDependentsSickensDependent(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	pt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 5) })
	ax := g.WorkAxes().AddByTwoPoints(pt.Key(), OriginCenter)

	if _, err := g.DeleteWork(pt.Key(), true); err != nil {
		t.Fatalf("DeleteWork: %v", err)
	}
	if ax.Deleted() {
		t.Error("with retainDependents the axis should be kept, not deleted")
	}
	g.Recompute(nil)
	if ax.Health().OK() {
		t.Error("the retained dependent should be unhealthy after its reference was deleted")
	}
}

// TestDeleteOriginRejected: neither an origin ref nor a positional ref that lands on an origin slot
// can be deleted (#1855).
func TestDeleteOriginRejected(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	if _, err := g.DeleteWork(OriginXYPlane, false); err == nil {
		t.Error("deleting an origin plane by its well-known ref should fail")
	}
	// "plane/0" resolves positionally onto the origin XY plane (the first slot).
	if _, err := g.DeleteWork(WorkRef("plane/0"), false); err == nil {
		t.Error("deleting a coordinate-system datum by its positional ref should fail")
	}
}

// TestDeleteUnknownAndAlreadyDeleted: an unknown ref and a second delete of the same datum both
// fail cleanly (#1855).
func TestDeleteUnknownAndAlreadyDeleted(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	if _, err := g.DeleteWork(WorkRef("plane/999"), false); err == nil {
		t.Error("an out-of-range ref should fail")
	}
	if _, err := g.DeleteWork(WorkRef("axis/nope"), false); err == nil {
		t.Error("a non-user ref should fail")
	}
	pl := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 1 })
	if _, err := g.DeleteWork(pl.Key(), true); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	if _, err := g.DeleteWork(pl.Key(), true); err == nil {
		t.Error("deleting an already-deleted datum should fail")
	}
}

// TestDeletePreservesPositionalRefs: deleting an earlier user datum does not shift a later datum's
// positional reference — the later datum still resolves to the same geometry (#1855).
func TestDeletePreservesPositionalRefs(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	first := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 5) })
	second := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(9, 0, 0) })

	if _, err := g.DeleteWork(first.Key(), true); err != nil {
		t.Fatalf("DeleteWork: %v", err)
	}
	g.Recompute(nil)
	got, err := g.point(second.Key())
	if err != nil {
		t.Fatalf("the surviving point should still resolve: %v", err)
	}
	if !got.IsEqualTo(math.P3(9, 0, 0), wtol) {
		t.Errorf("surviving point moved to %v after deleting an earlier datum; want (9,0,0)", got)
	}
}

// TestWorkFeatureFlagsRoundTrip: a construction datum and a tombstoned datum both survive a
// Marshal/Apply cycle — the construction flag and the deleted slot are preserved, and a datum
// created after the tombstone keeps its positional reference (#1849, #1855).
func TestWorkFeatureFlagsRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	con := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 2 })
	con.SetConstruction(true)
	doomed := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 1, 1) })
	survivor := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(3, 0, 0) })
	if _, err := g.DeleteWork(doomed.Key(), true); err != nil {
		t.Fatalf("DeleteWork: %v", err)
	}

	data, err := MarshalWork(g)
	if err != nil {
		t.Fatalf("MarshalWork: %v", err)
	}
	restored := NewWorkGeometry()
	if err := ApplyWork(restored, data); err != nil {
		t.Fatalf("ApplyWork: %v", err)
	}
	restored.Recompute(nil)

	rp := restored.WorkPlanes()
	if last := rp.Item(rp.Count() - 1); !last.Construction() || last.Deleted() {
		t.Errorf("restored plane: construction=%v deleted=%v, want construction only", last.Construction(), last.Deleted())
	}
	if _, ok := restored.WorkPointByRef(doomed.Key()); ok {
		t.Error("the tombstoned point should not resolve after restore")
	}
	got, err := restored.point(survivor.Key())
	if err != nil || !got.IsEqualTo(math.P3(3, 0, 0), wtol) {
		t.Errorf("survivor after restore = %v err %v, want (3,0,0) — positional ref must survive the tombstone", got, err)
	}
}

// TestDeletedAxisRefStopsResolving: a deleted axis no longer resolves as an axis input, and a
// deleted plane drops out of the viewport host-pick (#1855).
func TestDeletedAxisRefStopsResolving(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	ax := g.WorkAxes().AddByLine(math.P3(0, 0, 0), mustUnit(0, 0, 1))
	pl := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 1 })
	if !pl.ShownForHostPick(false) {
		t.Fatal("a visible user plane should be shown for host pick before deletion")
	}
	if _, err := g.DeleteWork(ax.Key(), true); err != nil {
		t.Fatalf("delete axis: %v", err)
	}
	if _, err := g.axis(ax.Key()); err == nil {
		t.Error("a deleted axis ref should not resolve")
	}
	if _, err := g.DeleteWork(pl.Key(), true); err != nil {
		t.Fatalf("delete plane: %v", err)
	}
	if pl.ShownForHostPick(true) {
		t.Error("a tombstoned plane must not be shown for host pick")
	}
}

// TestConstructionDefaultsFalse: a freshly created datum is not construction (#1849).
func TestConstructionDefaultsFalse(t *testing.T) {
	t.Parallel()
	g := NewWorkGeometry()
	pl := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 1 })
	if pl.Construction() {
		t.Error("a new plane should default to non-construction")
	}
	pl.SetConstruction(true)
	if !pl.Construction() {
		t.Error("SetConstruction(true) should stick")
	}
}
