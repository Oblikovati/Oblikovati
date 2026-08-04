// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// bossRootRimKey returns the reference key of the closed circle where a boss cylinder rises from a
// plate — a HOLE loop of the plate's top face, not the plate's own outer boundary. This is the #2006
// topology: the corpus cases R8/W6/W8/W9 all pick this shape of rim (a boss protruding from a bigger
// face, as opposed to I9's lone disc cap where the rim IS the cap's own outer boundary).
func bossRootRimKey(t *testing.T, b *topo.Body, bossZ float64) []byte {
	t.Helper()
	for _, e := range b.Edges() {
		if _, ok := e.Geometry().(geom.Circle); !ok {
			continue
		}
		if e.StartVertex() != e.EndVertex() {
			continue
		}
		if c := e.RangeBox().Center(); c.Z > bossZ-1e-6 && c.Z < bossZ+1e-6 {
			return e.ReferenceKey()
		}
	}
	t.Fatal("no boss-root rim found")
	return nil
}

// bossOnPlate builds a box with a cylindrical boss fused onto its top face — a plate + boss union, so
// the boss-to-plate rim sits as an INNER (hole) loop of the plate's top face, matching the corpus R8/
// W6/W8/W9 shape (see landscape-resurvey-report.md §3).
func bossOnPlate(t *testing.T) (*topo.Body, []byte) {
	t.Helper()
	box, err := brep.SolidBlock(math.P3(-10, -10, -10), math.P3(10, 10, 10), "box")
	if err != nil {
		t.Fatalf("box: %v", err)
	}
	boss, err := brep.SolidCylinder(math.P3(0, 0, 10), math.V3(0, 0, 1), 3, 15)
	if err != nil {
		t.Fatalf("boss cylinder: %v", err)
	}
	body, err := ops.Boolean(ops.Join, box, boss)
	if err != nil {
		t.Fatalf("box+boss union: %v", err)
	}
	return body, bossRootRimKey(t, body, 10)
}

// TestFilletRimBossRootOrientationConsistent is the #2006 regression: rounding a boss-root rim (a
// cylinder wall meeting a plate at a HOLE loop of the plate's own face, not the plate's outer boundary)
// used to build a manifold-but-invalid solid — "inconsistent orientation" on the cyl-tangent and
// cap-tangent replacement circles (fillet_rim_build.go's mapUse/addBandFace blindly copied the
// original rim edge's Reversed flag, which is only meaningful relative to the ORIGINAL rim curve's own
// parametrization, not the fresh replacement circles solveRim builds). It must now build a genuinely
// valid, manifold solid with every edge 2-incident and antiparallel — not just "no error returned".
func TestFilletRimBossRootOrientationConsistent(t *testing.T) {
	body, rimKey := bossOnPlate(t)
	res, err := ops.FilletEdges(body, [][]byte{rimKey}, 1.0)
	if err != nil {
		t.Fatalf("boss-root rim fillet declined: %v", err)
	}
	rep := ops.Validate(res)
	if !rep.Valid {
		t.Fatalf("boss-root rim fillet result invalid: %+v", rep)
	}
	assertEveryEdgeAntiparallel(t, res)
}

// assertEveryEdgeAntiparallel is the manifold 2-incidence invariant ops.Validate's OrientationOK
// encodes: every edge of a closed solid is used by exactly two faces, traversed in OPPOSITE senses.
// Asserted directly here (not just via ops.Validate) per #2006's regression requirement.
func assertEveryEdgeAntiparallel(t *testing.T, b *topo.Body) {
	t.Helper()
	for _, e := range b.Edges() {
		uses := e.Uses()
		if len(uses) != 2 {
			t.Fatalf("edge %d has %d uses, want exactly 2 on a closed solid", e.ID(), len(uses))
		}
		if uses[0].Reversed() == uses[1].Reversed() {
			t.Fatalf("edge %d: both uses Reversed=%v, want opposite directions", e.ID(), uses[0].Reversed())
		}
	}
}
