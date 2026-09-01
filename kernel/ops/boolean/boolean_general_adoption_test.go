// SPDX-License-Identifier: GPL-2.0-only

package boolean

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
	t.Parallel()
	cases := []struct {
		name    string
		general func(a, b *topo.Body) (*topo.Body, bool)
		a, b    func() *topo.Body
	}{
		{"crossing cylinders ∩",
			func(a, b *topo.Body) (*topo.Body, bool) { return brep.RuledCrossingIntersectGeneral(a, b, nil) },
			func() *topo.Body { b, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12); return b },
			func() *topo.Body { b, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12); return b }},
		{"cone ∩ cone",
			func(a, b *topo.Body) (*topo.Body, bool) { return brep.RuledCrossingIntersectGeneral(a, b, nil) },
			func() *topo.Body {
				b, _ := brep.SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
				return b
			},
			func() *topo.Body {
				b, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")
				return b
			}},
		{"cone ∩ cylinder",
			func(a, b *topo.Body) (*topo.Body, bool) { return brep.RuledCrossingIntersectGeneral(a, b, nil) },
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
	t.Parallel()
	fat := func() *topo.Body { b, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12); return b }
	rod := func() *topo.Body { b, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12); return b }
	cases := []struct {
		name    string
		general func() (*topo.Body, bool)
	}{
		{"crossing cylinders − (drill)", func() (*topo.Body, bool) { return brep.RuledCrossingCutGeneral(fat(), rod(), nil) }},
		{"crossing cylinders ∪", func() (*topo.Body, bool) { return brep.RuledCrossingJoinGeneral(fat(), rod(), nil) }},
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

// TestGeneralConeCutJoinIsAdopted guards the cone-pair CUT and JOIN general drivers (#1403, on the #1476
// wrapping-band emission): each must produce a validBooleanSolid result so ops.Boolean adopts it instead of
// falling back to the bespoke handler.
func TestGeneralConeCutJoinIsAdopted(t *testing.T) {
	t.Parallel()
	fatCone := func() *topo.Body {
		b, _ := brep.SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
		return b
	}
	rodCone := func() *topo.Body {
		b, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "thin")
		return b
	}
	cyl := func() *topo.Body { b, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12); return b }
	cone := func() *topo.Body {
		b, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
		return b
	}
	cases := []struct {
		name    string
		general func() (*topo.Body, bool)
	}{
		{"cone − cone (drill)", func() (*topo.Body, bool) { return brep.RuledCrossingCutGeneral(fatCone(), rodCone(), nil) }},
		{"cone ∪ cone", func() (*topo.Body, bool) { return brep.RuledCrossingJoinGeneral(fatCone(), rodCone(), nil) }},
		{"cone − cylinder (drill)", func() (*topo.Body, bool) { return brep.RuledCrossingCutGeneral(cyl(), cone(), nil) }},
		{"cone ∪ cylinder", func() (*topo.Body, bool) { return brep.RuledCrossingJoinGeneral(cyl(), cone(), nil) }},
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

// TestGeneralPartialIsAdopted guards the partial-penetration general drivers (a thin rod ending inside a
// fatter cylinder, #1403 on the #1476 wrapping-band + cap generalisation): each must produce a
// validBooleanSolid result so ops.Boolean adopts it instead of falling back to the bespoke handler.
func TestGeneralPartialIsAdopted(t *testing.T) {
	t.Parallel()
	fat := func() *topo.Body { b, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12); return b }
	stub := func() *topo.Body { b, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 6); return b }
	cases := []struct {
		name    string
		general func() (*topo.Body, bool)
	}{
		{"partial ∩ (plug)", func() (*topo.Body, bool) { return brep.PartialPenetrationIntersectGeneral(fat(), stub(), nil) }},
		{"partial − (blind hole)", func() (*topo.Body, bool) { return brep.PartialPenetrationCutGeneral(fat(), stub(), nil) }},
		{"partial ∪ (entry stub)", func() (*topo.Body, bool) { return brep.PartialPenetrationJoinGeneral(fat(), stub(), nil) }},
		{"partial − (rod stub lump)", func() (*topo.Body, bool) { return brep.PartialPenetrationCutGeneral(stub(), fat(), nil) }},
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
