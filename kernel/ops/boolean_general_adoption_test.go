// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// EPIC #1403 adoption guard. The general curved∩curved drivers must produce a solid that validBooleanSolid
// ACCEPTS — i.e. one ops.Boolean actually adopts instead of silently discarding for the bespoke fallback. This
// guard exists because every general migration shipped while the result was orientation-inconsistent:
// validBooleanSolid rejected it and the bespoke handler did all the work, yet the brep tests (edge-USE-COUNT
// "watertight" only) and the OCC oracle (run through the adopted = bespoke result) all passed. The imprint weld
// fix (emitImprintRun, #1403) is what makes the INTERSECT adopt; the OUTSIDE-keep wrapping-band emission
// (Oblikovati#1476) makes the crossing-cylinder CUT/JOIN adopt. This test fails if any silently falls back.
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

// TestGeneralCrossingCutJoinIsAdopted guards the crossing-cylinder CUT and JOIN general drivers (the
// OUTSIDE-keep wrapping-band emission, Oblikovati#1476): each must produce a validBooleanSolid result so
// ops.Boolean adopts it rather than falling back to the bespoke handler.
func TestGeneralCrossingCutJoinIsAdopted(t *testing.T) {
	fat := func() *topo.Body { b, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12); return b }
	rod := func() *topo.Body { b, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12); return b }
	cases := []struct {
		name    string
		general func() (*topo.Body, bool)
	}{
		{"crossing cylinders − (drill)", func() (*topo.Body, bool) { return brep.CrossingCylinderCutGeneral(fat(), rod(), nil) }},
		{"crossing cylinders ∪", func() (*topo.Body, bool) { return brep.CrossingCylinderJoinGeneral(fat(), rod(), nil) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, ok := c.general()
			if !ok {
				t.Fatalf("%s: general driver declined; want the general path taken", c.name)
			}
			if r := Validate(res); !r.Valid {
				t.Fatalf("%s: general result NOT adopted by validBooleanSolid (silent fallback): %+v", c.name, r)
			}
		})
	}
}
