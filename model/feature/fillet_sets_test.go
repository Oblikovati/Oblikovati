// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/model/sketch"
)

// Fillet edge sets (M09-F01 PBI-099, #323): mixed constant sets and the
// variable start→end radius set through the feature engine.

// boxAndVerticalEdges builds a 2×2×2 box and returns it with the reference
// keys of its four vertical edges (no two of which share a vertex).
func boxAndVerticalEdges(t *testing.T) (*PartFeatures, [][]byte) {
	t.Helper()
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 2, Y: 2}, {X: 0, Y: 2}}, sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")
	var keys [][]byte
	for _, e := range box.Edges() {
		if a, b := e.StartVertex().Point(), e.EndVertex().Point(); a.X == b.X && a.Y == b.Y {
			keys = append(keys, e.ReferenceKey())
		}
	}
	if len(keys) != 4 {
		t.Fatalf("found %d vertical edges, want 4", len(keys))
	}
	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)
	return fs, keys
}

// TestFilletSetsMixedConstantRadii rounds two vertical edges at different
// radii via two constant sets in one feature — the cylinder-exact volume.
func TestFilletSetsMixedConstantRadii(t *testing.T) {
	fs, keys := boxAndVerticalEdges(t)
	pf := NewDressUpFeatures(fs).AddFilletSets([]FilletEdgeSet{
		{EdgeKeys: [][]byte{keys[0]}, Radius: angleConst(0.3)},
		{EdgeKeys: [][]byte{keys[1]}, Radius: angleConst(0.6)},
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("mixed-set fillet sick: %+v", pf.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("mixed-set fillet not a valid solid: %+v", r)
	}
	notch := func(r float64) float64 { return r*r - stdmath.Pi*r*r/4 }
	want := 8 - notch(0.3)*2 - notch(0.6)*2
	if got := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-4}).Volume; relErr(got, want) > 1e-3 {
		t.Errorf("mixed-set fillet volume = %g, want ≈ %g", got, want)
	}
}

// TestFilletVariableSetThroughEngine rounds one vertical edge 0.3→0.6 through the feature
// engine — the smooth-blend volume, since the variable blend is the exact rational ruled
// surface (#1606; the pre-A10 planar strips followed the chord integral instead).
func TestFilletVariableSetThroughEngine(t *testing.T) {
	fs, keys := boxAndVerticalEdges(t)
	pf := NewDressUpFeatures(fs).AddFilletSets([]FilletEdgeSet{
		{EdgeKeys: [][]byte{keys[0]}, StartRadius: angleConst(0.3), EndRadius: angleConst(0.6)},
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("variable fillet sick: %+v", pf.Health())
	}
	res := fs.Result()[0]
	removed := (1 - stdmath.Pi/4) * 2 * (0.3*0.3 + 0.3*0.6 + 0.6*0.6) / 3
	want := 8 - removed
	if got := ops.BodyGeometryProperties(res, fineFilletQuality()).Volume; stdmath.Abs(got-want) > 0.03*removed {
		t.Errorf("variable fillet volume = %g, want %g (smooth blend, 3%%-of-notch band)", got, want)
	}
	if pf.Definition().(*FilletFeature).Definition().FilletType() != 61697 {
		t.Error("fillet type discriminator != edge fillet")
	}
}

// TestFilletRadiusPointsThroughEngine rounds one vertical edge with an intermediate
// radius stop (0.3 → 0.7 at T=0.5 → 0.4) through the feature engine — the per-segment
// smooth-blend volume (#695, exact spans since #1606).
func TestFilletRadiusPointsThroughEngine(t *testing.T) {
	fs, keys := boxAndVerticalEdges(t)
	pf := NewDressUpFeatures(fs).AddFilletSets([]FilletEdgeSet{{
		EdgeKeys:     [][]byte{keys[0]},
		StartRadius:  angleConst(0.3),
		EndRadius:    angleConst(0.4),
		RadiusPoints: []FilletRadiusPoint{{T: 0.5, Radius: angleConst(0.7)}},
	}})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("radius-points fillet sick: %+v", pf.Health())
	}
	res := fs.Result()[0]
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("radius-points fillet not a valid solid: %+v", r)
	}
	seg := func(ra, rb float64) float64 { return 1.0 * (ra*ra + ra*rb + rb*rb) / 3 }
	removed := (1 - stdmath.Pi/4) * (seg(0.3, 0.7) + seg(0.7, 0.4))
	want := 8 - removed
	if got := ops.BodyGeometryProperties(res, fineFilletQuality()).Volume; stdmath.Abs(got-want) > 0.03*removed {
		t.Errorf("radius-points fillet volume = %g, want %g (smooth blend, 3%%-of-notch band)", got, want)
	}
}

// fineFilletQuality is the tight tessellation the smooth-blend volume brackets measure at (the
// exact blend converges to the smooth integral as the chord tolerance tightens, #1606).
func fineFilletQuality() ops.Quality {
	return ops.Quality{ChordTolerance: 0.002, AngleTolerance: 2 * stdmath.Pi / 180}
}

// TestFilletVariableSetNeedsOneEdge: a variable set over two edges is a
// precise Sick, not a broken body.
func TestFilletVariableSetNeedsOneEdge(t *testing.T) {
	fs, keys := boxAndVerticalEdges(t)
	pf := NewDressUpFeatures(fs).AddFilletSets([]FilletEdgeSet{
		{EdgeKeys: [][]byte{keys[0], keys[1]}, StartRadius: angleConst(0.2), EndRadius: angleConst(0.4)},
	})
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("two-edge variable set = %v, want Sick", pf.Health().Status)
	}
}

// TestFilletSetsLostEdgeSick: any lost key in any set makes the feature Sick.
func TestFilletSetsLostEdgeSick(t *testing.T) {
	fs, keys := boxAndVerticalEdges(t)
	pf := NewDressUpFeatures(fs).AddFilletSets([]FilletEdgeSet{
		{EdgeKeys: [][]byte{keys[0]}, Radius: angleConst(0.3)},
		{EdgeKeys: [][]byte{[]byte("gone")}, Radius: angleConst(0.2)},
	})
	fs.Recompute()
	if pf.Health().Status != health.Sick {
		t.Errorf("lost-edge set = %v, want Sick", pf.Health().Status)
	}
}
