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

// TestObliquePartialPenetrationFlushBottomStaysManifold is the FAST minimal reproduction of
// the "V2" boolean defect (PBI-199). The heavy spec is the full fan
// (bridge e2e_fan_validate_test.go); this isolates it in milliseconds with two boxes (subd.Box
// is corner-at-origin, so both sit on z=0):
//
//	A 6×6×3 box (with a 3×3 square notch) JOIN a small 0.6×0.6×1 blade rotated 45° about Z and
//	poking its tip into the box's corner — its BOTTOM face flush (coplanar) on the z=0 base.
//
// It used to come out NON-MANIFOLD (an edge shared by 3 faces). CORRECTED root cause (the old
// "concave wall / floating imprint" diagnosis was wrong — subd.Box is corner-at-origin, so the
// blade pokes a CONVEX corner, not a concave one): the trigger is the COPLANAR FLUSH BOTTOM
// combined with the oblique partial penetration — lifting the blade off z=0 (no coplanar
// contact) makes the very same penetration VALID. The coplanar seam (blade wall + blade bottom
// + box bottom meeting at the tip) doesn't stitch in the planar arrangement.
//
// Now PASSES via the guarded fallback in booleanGeneral: when the exact planar boolean returns
// an invalid (non-manifold) body, it retries with the robust triangle CSG and adopts it when
// that validates. The result is heavier but a valid solid. (The root-cause coplanar-seam
// arrangement fix is the cleaner future work.)
func TestObliquePartialPenetrationFlushBottomStaysManifold(t *testing.T) {
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
