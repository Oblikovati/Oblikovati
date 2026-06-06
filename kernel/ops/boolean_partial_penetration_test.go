// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/subd"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

func ppLineage(l topo.Lineage) topo.Lineage { return l }

// TestPartialPenetrationConcaveWallIsNonManifold is the FAST minimal reproduction of the
// curved/partial-penetration boolean defect (PBI-199, see the partial-penetration memory).
// The heavy spec is the full fan (bridge e2e_fan_validate_test.go); this isolates the same bug
// in milliseconds with two boxes:
//
//	L-block (a 6×6×3 box with a 3×3 corner cut → a CONCAVE inner corner) JOIN a small blade box
//	rotated 45° poking diagonally INTO that concave corner.
//
// The JOIN comes out NON-MANIFOLD (an edge shared by 3 faces). Root cause (instrumented): the
// blade only PARTIALLY penetrates, so on a blade face the imprint of a crossing L-face is a
// FLOATING segment — both endpoints strictly INSIDE the blade face (the segment is clipped to
// the L-face's finite extent, not to the blade face's edges). The per-face 2D arrangement only
// splits a face on boundary-to-boundary chords, so a floating segment never cuts it; the overlap
// region survives and stitches as a coincident shell.
//
// A contained fix (extend the dangling endpoint to the face boundary along its line) was tried
// and REJECTED: it over-splits the face and the extra split has no matching vertex on the
// adjacent face → a T-junction → still non-manifold. The real fix is cross-face arrangement
// consistency (assemble imprint loops across shared edges / propagate T-vertices), the multi-day
// arrangement-robustness work. Unskip when that lands.
func TestPartialPenetrationConcaveWallIsNonManifold(t *testing.T) {
	t.Skip("DEFECT (PBI-199): partial penetration into a concave wall yields a floating imprint " +
		"segment the per-face arrangement can't split → non-manifold. Fast spec for the cross-face " +
		"arrangement-robustness fix; unskip when it lands. See the heavy full-fan spec in " +
		"oblikovati/mcp-bridge bridge/e2e_fan_validate_test.go (TestBladeJoinBooleanIsTheDefect).")

	box := subd.ToBody(subd.Box(6, 6, 3), "box")
	corner := subd.ToBody(subd.Box(3, 3, 4), "cut")
	cornerT, err := ops.TransformBody(corner, math.Translation4(math.V3(1.5, 1.5, 0)), ppLineage)
	if err != nil {
		t.Fatalf("position corner: %v", err)
	}
	lblock, err := ops.Boolean(ops.Cut, box, cornerT) // an L-prism with a concave inner corner
	if err != nil {
		t.Fatalf("make L-block: %v", err)
	}
	if r := ops.Validate(lblock); !r.Valid {
		t.Fatalf("L-block already invalid: %v", r.Issues)
	}

	zAxis, _ := math.UnitVector3FromVector(math.V3(0, 0, 1))
	blade := subd.ToBody(subd.Box(0.6, 0.6, 1.0), "blade")
	m := math.Translation4(math.V3(-0.4, -0.4, 0)).Mul(math.Rotation4(45*stdmath.Pi/180, zAxis, math.P3(0, 0, 0)))
	bladeT, err := ops.TransformBody(blade, m, ppLineage)
	if err != nil {
		t.Fatalf("position blade: %v", err)
	}

	joined, err := ops.Boolean(ops.Join, lblock, bladeT)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	if r := ops.Validate(joined); !r.Valid {
		t.Fatalf("partial-penetration JOIN is non-manifold: %v", r.Issues)
	}
}
