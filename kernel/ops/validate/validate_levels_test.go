// SPDX-License-Identifier: GPL-2.0-only

package validate

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// protrudingHoleFace builds the one shape that separates the two levels: a planar SHEET face whose
// inner loop lies OUTSIDE its outer loop — a hole that is not a hole. Level 1 cannot see it (every
// edge is used once on an open sheet, and chi is unconstrained); the hole-containment level flags it.
func protrudingHoleFace(t *testing.T) *topo.Body {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "protrude", 0))
	bld := topo.NewBuilder(false, lin)
	plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	outer := squareLoop(bld, lin, 0, 0, 4)
	away := squareLoop(bld, lin, 20, 20, 1) // a "hole" nowhere near the outer loop
	bld.AddFace(plane, lin, topo.OuterLoop(outer...), topo.InnerLoop(away...))
	return bld.Build()
}

// squareLoop wires an axis-aligned square of the given side at (x,y) and returns its edge uses.
func squareLoop(bld *topo.Builder, lin topo.Lineage, x, y, side float64) []topo.Use {
	pts := []math.Point3{
		math.P3(x, y, 0), math.P3(x+side, y, 0), math.P3(x+side, y+side, 0), math.P3(x, y+side, 0),
	}
	vs := make([]*topo.Vertex, len(pts))
	for i, p := range pts {
		vs[i] = bld.AddVertex(p, lin)
	}
	uses := make([]topo.Use, len(pts))
	for i := range pts {
		j := (i + 1) % len(pts)
		uses[i] = topo.Fwd(bld.AddEdge(geom.NewLineSegment(pts[i], pts[j]), vs[i], vs[j], lin))
	}
	return uses
}

// TestValidateTopologyIsTheLevelBelowValidate pins what each level answers. The feature engine's
// post-condition runs level 1 on every result of every recompute, and the reason recorded for that
// used to say level 1 was all Validate did — it also runs hole containment, whose verdict is not part
// of Valid, so the post-condition was paying for an answer it discarded (finding 5 of the stage-6
// review).
func TestValidateTopologyIsTheLevelBelowValidate(t *testing.T) {
	t.Parallel()
	b := protrudingHoleFace(t)

	lvl1 := ValidateTopology(b)
	if lvl1.HoleContainmentChecked {
		t.Error("level 1 must not claim to have checked hole containment")
	}

	full := Validate(b)
	if !full.HoleContainmentChecked {
		t.Error("Validate must record that the hole-containment level ran")
	}
	if full.HolesContained {
		t.Fatal("the fixture's inner loop lies outside its outer loop and must be flagged")
	}
	// The two levels agree on everything level 1 measures — Validate IS level 1 plus the extra level.
	if lvl1.Valid != full.Valid || lvl1.Closed != full.Closed || lvl1.Manifold != full.Manifold ||
		lvl1.OrientationOK != full.OrientationOK || lvl1.EulerCharacteristic != full.EulerCharacteristic {
		t.Errorf("the levels disagree on level-1 verdicts: level1=%+v full=%+v", lvl1, full)
	}
	// And the extra level is what costs: it found a defect level 1 reported nothing about.
	if len(full.Issues) <= len(lvl1.Issues) {
		t.Errorf("Validate must add the containment issue level 1 cannot see: level1=%d full=%d issues",
			len(lvl1.Issues), len(full.Issues))
	}
}

// The level split must not change any existing verdict: Valid never depended on HolesContained.
func TestTheLevelSplitLeavesValidUnchanged(t *testing.T) {
	t.Parallel()
	b := protrudingHoleFace(t)
	if got, want := ValidateTopology(b).Valid, Validate(b).Valid; got != want {
		t.Errorf("ValidateTopology(b).Valid = %v, Validate(b).Valid = %v; the containment level is not part of Valid", got, want)
	}
}
