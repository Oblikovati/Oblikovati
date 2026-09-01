// SPDX-License-Identifier: GPL-2.0-only

package validate_test

import (
	"testing"

	"oblikovati.org/kernel/ops"

	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/ops/heal"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestSelfIntersectionsCleanSolid: a valid solid has none — adjacent faces
// touching along shared edges must not count.
func TestSelfIntersectionsCleanSolid(t *testing.T) {
	t.Parallel()
	body, err := heal.Stitch(brepfixture.CubeFaces(), 0, false, "clean")
	if err != nil {
		t.Fatalf("heal.Stitch: %v", err)
	}
	if hits := ops.SelfIntersections(body, ops.DefaultQuality()); len(hits) != 0 {
		t.Errorf("clean cube reports %d self-intersections, want 0: %+v", len(hits), hits)
	}
}

// TestSelfIntersectionsOverlappingShells: two interpenetrating tetra shells
// merged into one body — the import failure mode PBI-084 targets — must be
// caught with a witness inside the overlap region.
func TestSelfIntersectionsOverlappingShells(t *testing.T) {
	t.Parallel()
	a := brepfixture.Tetra(1, math.V3(0, 0, 0))
	b := brepfixture.Tetra(1, math.V3(0.2, 0.2, 0.2)) // shifted: shells pass through each other
	merged := topo.MergeBodies(topo.NewLineage(topo.Tok("imp", "body", 0)), true, a, b)
	hits := ops.SelfIntersections(merged, ops.DefaultQuality())
	if len(hits) == 0 {
		t.Fatal("interpenetrating shells must report self-intersections")
	}
	w := hits[0].Witness
	if w.X < 0 || w.X > 1.2 || w.Y < 0 || w.Y > 1.2 || w.Z < 0 || w.Z > 1.2 {
		t.Errorf("witness %v lies outside the bodies' overlap region", w)
	}
}

// TestSelfIntersectionsDisjointShells: a two-shell body whose shells do not
// touch (a legitimate multi-shell solid) is clean.
func TestSelfIntersectionsDisjointShells(t *testing.T) {
	t.Parallel()
	a := brepfixture.Tetra(1, math.V3(0, 0, 0))
	b := brepfixture.Tetra(1, math.V3(5, 0, 0))
	merged := topo.MergeBodies(topo.NewLineage(topo.Tok("imp", "body", 0)), true, a, b)
	if hits := ops.SelfIntersections(merged, ops.DefaultQuality()); len(hits) != 0 {
		t.Errorf("disjoint shells report %d self-intersections, want 0", len(hits))
	}
}

// TestSelfIntersectionsCoplanarOverlap: two coplanar overlapping faces (a
// doubled wall from a bad import) are caught by the coplanar branch.
func TestSelfIntersectionsCoplanarOverlap(t *testing.T) {
	t.Parallel()
	p := math.P3
	q1 := brepfixture.QuadBody("w1", p(0, 0, 0), p(2, 0, 0), p(2, 0, 2), p(0, 0, 2))
	q2 := brepfixture.QuadBody("w2", p(1, 0, 1), p(3, 0, 1), p(3, 0, 3), p(1, 0, 3)) // overlaps q1's corner
	merged := topo.MergeBodies(topo.NewLineage(topo.Tok("imp", "body", 0)), false, q1, q2)
	if hits := ops.SelfIntersections(merged, ops.DefaultQuality()); len(hits) == 0 {
		t.Error("coplanar overlapping faces must report a self-intersection")
	}
}
