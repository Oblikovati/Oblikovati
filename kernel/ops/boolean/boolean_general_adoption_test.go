// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The ruled-crossing and partial-penetration corpus at the ops entry (EPIC #1403, ADR-0061 stage 4).
//
// This file used to guard ADOPTION: each general brep driver was called directly and its result had to
// pass validBooleanSolid, because the boolean would otherwise discard it and let a bespoke recognizer do
// the work — a silent substitution that the brep tests (edge-use-count "watertight" only) and the OCC
// oracle (run through the adopted, i.e. bespoke, result) both missed. The recognizers are deleted, so
// there is nothing left to fall back TO and adoption is no longer a question. What the rows are worth is
// the corpus itself: thirteen ruled pairs, each of which must come out of ops.Boolean as a valid solid.
// Certification (the per-face membership rule and the Requicha bracket) runs inside that entry, so a wrong
// body is refused by name rather than quietly replaced.

// ruledPair is one corpus row: an operation over two ruled operands, built fresh per subtest. wantCyl and
// wantPlan, when either is set, pin the analytic face census the pipeline must emit — the structure the
// deleted drivers' own tests asserted, carried over so the shape is checked and not only the validity.
type ruledPair struct {
	name              string
	op                PartFeatureOperation
	a, b              func() *topo.Body
	wantCyl, wantPlan int
}

func cylZ12(r float64) func() *topo.Body {
	return func() *topo.Body {
		b, _ := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), math.Scalar(r), 12)
		return b
	}
}

func cylX(r, h float64) func() *topo.Body {
	return func() *topo.Body {
		b, _ := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), math.Scalar(r), math.Scalar(h))
		return b
	}
}

func coneZ() *topo.Body {
	b, _ := brep.SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
	return b
}

func coneX(r0, r1 float64) func() *topo.Body {
	return func() *topo.Body {
		b, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), math.Scalar(r0), math.Scalar(r1), "thin")
		return b
	}
}

// assertRuledCorpus runs each pair through ops.Boolean and requires a valid solid.
func assertRuledCorpus(t *testing.T, rows []ruledPair) {
	t.Helper()
	for _, c := range rows {
		t.Run(c.name, func(t *testing.T) {
			res, err := Boolean(c.op, c.a(), c.b())
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			if r := Validate(res); !r.ValidSolid() {
				t.Fatalf("%s: the general pipeline produced a non-solid: %+v", c.name, r)
			}
			if c.wantCyl == 0 && c.wantPlan == 0 {
				return
			}
			cyls, planes := 0, 0
			for _, f := range res.Faces() {
				switch f.Geometry().(type) {
				case geom.Cylinder:
					cyls++
				case geom.Plane:
					planes++
				}
			}
			if cyls != c.wantCyl || planes != c.wantPlan {
				t.Errorf("%s: %d cylinder + %d plane faces, want %d + %d", c.name, cyls, planes, c.wantCyl, c.wantPlan)
			}
		})
	}
}

// TestRuledCrossingIntersectCorpus: every ruled∩ruled crossing — cylinder, cone, and the mixed pair.
func TestRuledCrossingIntersectCorpus(t *testing.T) {
	t.Parallel()
	assertRuledCorpus(t, []ruledPair{
		{"crossing cylinders ∩", Intersect, cylZ12(3), cylX(1.5, 12), 0, 0},
		{"cone ∩ cone", Intersect, coneZ, coneX(0.8, 1.5), 0, 0},
		{"cone ∩ cylinder", Intersect, cylZ12(3), coneX(1, 2.5), 0, 0},
	})
}

// TestRuledCrossingCutJoinCorpus: the crossing-cylinder CUT and JOIN, the OUTSIDE-keep wrapping-band
// emission (Oblikovati#1476) that makes a side-breached wall a single holed tube.
func TestRuledCrossingCutJoinCorpus(t *testing.T) {
	t.Parallel()
	assertRuledCorpus(t, []ruledPair{
		{"crossing cylinders − (drill)", Cut, cylZ12(3), cylX(1.5, 12), 0, 0},
		{"crossing cylinders ∪", Join, cylZ12(3), cylX(1.5, 12), 0, 0},
	})
}

// TestRuledConeCutJoinCorpus: the same two operations over cone pairs and the cone/cylinder mix (#1403).
func TestRuledConeCutJoinCorpus(t *testing.T) {
	t.Parallel()
	assertRuledCorpus(t, []ruledPair{
		{"cone − cone (drill)", Cut, coneZ, coneX(0.8, 1.5), 0, 0},
		{"cone ∪ cone", Join, coneZ, coneX(0.8, 1.5), 0, 0},
		{"cone − cylinder (drill)", Cut, cylZ12(3), coneX(1, 2.5), 0, 0},
		{"cone ∪ cylinder", Join, cylZ12(3), coneX(1, 2.5), 0, 0},
	})
}

// TestPartialPenetrationCorpus: a thin rod ending INSIDE a fatter cylinder — the plug, the blind hole, the
// entry stub, and the lump left when the rod is the target (#1403 on the #1476 wrapping-band + cap work).
func TestPartialPenetrationCorpus(t *testing.T) {
	t.Parallel()
	fat, stub := cylZ12(3), cylX(1.5, 6)
	assertRuledCorpus(t, []ruledPair{
		// 2 cyl: the fat-wall lens cap + the rod-wall band. 1 plane: the rod's blind cap.
		{"partial ∩ (plug)", Intersect, fat, stub, 2, 1},
		// 2 cyl: the holed fat wall + the rod tunnel. 3 plane: the fat's 2 caps + the blind cap as the
		// pocket bottom.
		{"partial − (blind hole)", Cut, fat, stub, 2, 3},
		// 2 cyl: the holed fat wall + the single rod stub. 3 plane: the fat's 2 caps + the rod's ENTRY
		// cap; its blind cap, inside the fat, is dropped.
		{"partial ∪ (entry stub)", Join, fat, stub, 2, 3},
		{"partial − (rod stub lump)", Cut, stub, fat, 0, 0},
	})
}
