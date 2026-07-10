// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
)

// TestConstructionRefsListsOnlyConstruction: ConstructionRefs returns live user construction datums
// (plane/axis/point) and excludes plain and origin datums (#1849).
func TestConstructionRefsListsOnlyConstruction(t *testing.T) {
	g := NewWorkGeometry()
	cp := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 5 })
	cp.SetConstruction(true)
	np := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 3 }) // plain
	cax := g.WorkAxes().AddByLine(math.P3(0, 0, 0), mustUnit(0, 0, 1))
	cax.SetConstruction(true)
	cpt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 0) })
	cpt.SetConstruction(true)
	g.Recompute(nil)

	refs := g.ConstructionRefs()
	if !contains(refs, cp.Key()) || !contains(refs, cax.Key()) || !contains(refs, cpt.Key()) {
		t.Errorf("ConstructionRefs = %v, want the construction plane, axis, and point", refs)
	}
	if contains(refs, np.Key()) {
		t.Error("a plain datum must not appear in ConstructionRefs")
	}
	if contains(refs, OriginXYPlane) {
		t.Error("an origin datum is never construction")
	}
}

// TestRefConsumedByDatum: a datum referenced by another datum is reported consumed; an unreferenced
// one is not (#1849).
func TestRefConsumedByDatum(t *testing.T) {
	g := NewWorkGeometry()
	cp := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 5 })
	cp.SetConstruction(true)
	pt := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 0) })
	g.WorkAxes().AddByPointAndPlane(pt.Key(), cp.Key()) // axis references the construction plane
	g.Recompute(nil)

	if !g.RefConsumedByDatum(cp.Key()) {
		t.Error("the construction plane is referenced by an axis, so it should read as consumed")
	}
	if g.RefConsumedByDatum(WorkRef("plane/404")) {
		t.Error("an unreferenced ref must not read as consumed")
	}
}

// TestPruneConstructionOrphan: pruning tombstones a live construction datum and is a no-op on an
// origin datum or an unknown ref (#1849).
func TestPruneConstructionOrphan(t *testing.T) {
	g := NewWorkGeometry()
	cp := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 5 })
	cp.SetConstruction(true)
	g.Recompute(nil)

	g.PruneConstructionOrphan(cp.Key())
	if !cp.Deleted() {
		t.Error("pruning should tombstone the construction plane")
	}
	g.PruneConstructionOrphan(OriginXYPlane)        // origin: no-op, must not panic
	g.PruneConstructionOrphan(WorkRef("plane/404")) // unknown: no-op
}

func contains(refs []WorkRef, ref WorkRef) bool {
	for _, r := range refs {
		if r == ref {
			return true
		}
	}
	return false
}
