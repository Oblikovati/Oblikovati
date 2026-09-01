// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Rule fillet (#486, M20-F10): round a whole dihedral class of edges (all rounds / all fillets) in
// one feature, instead of picking edges individually.

// TestRuleFilletAllRoundsRoundsWholeBox rounds every convex edge of a 4×4×4 box in one feature: the
// result is a valid solid with cylindrical fillet faces and less material than the box.
func TestRuleFilletAllRoundsRoundsWholeBox(t *testing.T) {
	t.Parallel()
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}}, sketch.XYPlane(), span{near: 0, far: 4}, 0, "box")
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	pf := NewDressUpFeatures(fs).AddRuleFillet(RuleFilletAllRounds, func() float64 { return 0.5 })
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("rule fillet (all rounds) sick: %+v", pf.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("rule fillet result not a valid solid: %+v", r)
	}
	if !hasCylinderFace(res) {
		t.Error("rule fillet (all rounds) produced no cylindrical face")
	}
	if v := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-3}).Volume; v <= 0 || v >= 64 {
		t.Errorf("rounded box volume = %g, want 0 < v < 64 (material removed at every edge)", v)
	}
}

// TestRuleFilletAllFilletsFillsConcaveEdge fills the one concave edge of an L-prism, leaving the
// convex edges sharp: a valid solid with a cylindrical fillet face.
func TestRuleFilletAllFilletsFillsConcaveEdge(t *testing.T) {
	t.Parallel()
	l := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 2}, {X: 2, Y: 2}, {X: 2, Y: 4}, {X: 0, Y: 4}},
		sketch.XYPlane(), span{near: 0, far: 3}, 0, "L")
	if n := concaveEdgeCount(l); n != 1 {
		t.Fatalf("L-prism has %d concave edges, want 1", n)
	}
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(l)
	pf := NewDressUpFeatures(fs).AddRuleFillet(RuleFilletAllFillets, func() float64 { return 0.5 })
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("rule fillet (all fillets) sick: %+v", pf.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("rule fillet result not a valid solid: %+v", r)
	}
	if !hasCylinderFace(res) {
		t.Error("rule fillet (all fillets) produced no cylindrical face")
	}
}

// TestRuleFilletNoMatchNoOp: all-fillets on a plain box (no concave edge) changes nothing.
func TestRuleFilletNoMatchNoOp(t *testing.T) {
	t.Parallel()
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	pf := NewDressUpFeatures(fs).AddRuleFillet(RuleFilletAllFillets, func() float64 { return 0.3 })
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("no-match rule fillet sick: %+v", pf.Health())
	}
	if v := ops.BodyGeometryProperties(fs.Result()[0], ops.Quality{ChordTolerance: 1e-3}).Volume; relErr(v, 8) > 1e-6 {
		t.Errorf("box volume after no-match all-fillets = %g, want 8 (unchanged)", v)
	}
}

func concaveEdgeCount(b *topo.Body) int {
	n := 0
	for _, e := range b.Edges() {
		if ops.ClassifyEdgeConvexity(e) == ops.EdgeConcave {
			n++
		}
	}
	return n
}
