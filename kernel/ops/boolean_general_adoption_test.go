// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// EPIC #1403 adoption guard. The general curved∩curved INTERSECT drivers must produce a solid that
// validBooleanSolid ACCEPTS — i.e. one ops.Boolean actually adopts instead of silently discarding for the
// bespoke fallback. This guard exists because every general migration shipped while the result was
// orientation-inconsistent: validBooleanSolid rejected it and the bespoke handler did all the work, yet the
// brep tests (edge-USE-COUNT "watertight" only) and the OCC oracle (run through the adopted = bespoke result)
// all passed. The imprint weld fix (emitImprintRun keeps the closed-loop traversal sense) is what makes these
// adopt; this test would have caught the silent fallback, and fails if it ever returns. The general CUT/JOIN
// are NOT here: they still mesh the wrong region and stay on the bespoke handlers until Oblikovati#1476.
func TestGeneralIntersectIsAdopted(t *testing.T) {
	cases := []struct {
		name    string
		general func(a, b *topo.Body) (*topo.Body, bool)
		a, b    func() *topo.Body
	}{
		{"crossing cylinders ∩",
			func(a, b *topo.Body) (*topo.Body, bool) { return brep.CrossingCylinderIntersectGeneral(a, b, nil) },
			func() *topo.Body { b, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12); return b },
			func() *topo.Body { b, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12); return b }},
		{"cone ∩ cone",
			func(a, b *topo.Body) (*topo.Body, bool) { return brep.ConeConeIntersectGeneral(a, b, nil) },
			func() *topo.Body {
				b, _ := brep.SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
				return b
			},
			func() *topo.Body {
				b, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")
				return b
			}},
		{"cone ∩ cylinder",
			func(a, b *topo.Body) (*topo.Body, bool) { return brep.ConeCylinderIntersectGeneral(a, b, nil) },
			func() *topo.Body { b, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12); return b },
			func() *topo.Body {
				b, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
				return b
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, ok := c.general(c.a(), c.b())
			if !ok {
				t.Fatalf("%s: general intersect declined; want the general path taken", c.name)
			}
			if r := Validate(res); !r.Valid {
				t.Fatalf("%s: general result NOT adopted by validBooleanSolid (silent fallback): %+v", c.name, r)
			}
		})
	}
}
