// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The curved-on-planar contact, across the interior/partial boundary (#1591, ADR-0049 D-b, Slice C;
// ADR-0061 stage 4).
//
// A cylinder meeting a planar face in a circle has two shapes: the circle lies WHOLLY INSIDE the face, or
// it PIERCES the face's edge. Four bespoke recognizers used to divide that pair of shapes between them —
// DrillThroughHole and JoinCylindricalBoss took the interior contact, CutEdgeScallop and JoinPartialBoss
// the pierced one — and this file existed to pin the gate between them: each partial recognizer had to
// DECLINE the interior case, or it would steal the other's work and build the wrong body.
//
// All four are deleted. There is one pipeline and the gate is gone with them, so what the file pins now is
// that the boundary itself is not a discontinuity: the four contacts differ only in where the circle sits,
// and each comes out as an exact analytic solid whose face count grows by exactly the splits the contact
// implies — the pierced cases carry one more planar face than their interior twin, because the circle cuts
// the face's edge instead of nesting inside it.

// curvedOnPlanarCase is one contact: an operation over a plate and a cylinder, with the exact face census
// the general pipeline must produce.
type curvedOnPlanarCase struct {
	name              string
	op                ops.PartFeatureOperation
	tool              func() *topo.Body
	wantCyl, wantPlan int
}

func plate10x10x2(t *testing.T) *topo.Body {
	t.Helper()
	p, err := brep.SolidBlock(math.P3(-5, -5, 0), math.P3(5, 5, 2), "plate")
	if err != nil {
		t.Fatalf("plate: %v", err)
	}
	return p
}

// TestCurvedOnPlanarContactsAreAnalyticSolids drives both contacts of both operations through ops.Boolean.
// The interior drill leaves the plate's six faces plus the bore wall; moving the same drill out to clip the
// +x edge splits that side face, so the scallop carries one more. The bosses add a wall and a cap on top of
// their seat, and straddle the same way.
func TestCurvedOnPlanarContactsAreAnalyticSolids(t *testing.T) {
	t.Parallel()
	cyl := func(x, z, h float64) func() *topo.Body {
		return func() *topo.Body {
			b, _ := brep.SolidCylinder(math.P3(math.Scalar(x), 0, math.Scalar(z)), math.V3(0, 0, 1), 2, math.Scalar(h))
			return b
		}
	}
	for _, c := range []curvedOnPlanarCase{
		{"interior drill", ops.Cut, cyl(0, -1, 4), 1, 6},  // circle inside both caps: 6 plate faces + bore
		{"edge scallop", ops.Cut, cyl(4, -1, 4), 1, 7},    // circle clips the +x edge: that side face splits
		{"interior boss", ops.Join, cyl(0, 2, 3), 1, 7},   // 6 plate faces + boss cap, seat face holed
		{"straddling boss", ops.Join, cyl(4, 2, 3), 1, 8}, // the base circle overhangs: the +x side splits
		{"downward boss", ops.Join, cyl(0, -3, 3), 1, 7},  // seated on the BOTTOM face: the seat's sense must not matter
	} {
		t.Run(c.name, func(t *testing.T) {
			res, err := ops.Boolean(c.op, plate10x10x2(t), c.tool())
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			assertAnalyticCylinderSolid(t, res, c.name, c.wantCyl, c.wantPlan)
		})
	}
}

// assertAnalyticCylinderSolid checks a curved-on-planar result is one watertight shell whose faces are
// exactly the wanted analytic census — no extra surface kind, and the cylinder wall kept analytic.
func assertAnalyticCylinderSolid(t *testing.T, res *topo.Body, label string, wantCyl, wantPlan int) {
	t.Helper()
	if !res.IsSolid() {
		t.Errorf("%s: result is not a solid", label)
	}
	if n := len(res.Shells()); n != 1 {
		t.Errorf("%s: result has %d shells, want 1", label, n)
	}
	cyls, planes, other := 0, 0, 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder:
			cyls++
		case geom.Plane:
			planes++
		default:
			other++
		}
	}
	if cyls != wantCyl || planes != wantPlan || other != 0 {
		t.Errorf("%s: %d cylinder + %d plane + %d other faces, want %d + %d + 0",
			label, cyls, planes, other, wantCyl, wantPlan)
	}
}
