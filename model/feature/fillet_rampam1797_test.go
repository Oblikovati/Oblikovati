// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// verticalEdgeKeysF returns the keys of a prism's vertical edges (Z varies, X/Y fixed).
func verticalEdgeKeysF(b *topo.Body) [][]byte {
	var keys [][]byte
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if a.X == c.X && a.Y == c.Y {
			keys = append(keys, e.ReferenceKey())
		}
	}
	return keys
}

// topPerimeterKeysF returns the keys of every edge lying entirely on the body's top plane.
func topPerimeterKeysF(b *topo.Body) [][]byte {
	maxZ := 0.0
	for _, v := range b.Vertices() {
		if v.Point().Z > maxZ {
			maxZ = v.Point().Z
		}
	}
	var keys [][]byte
	for _, e := range b.Edges() {
		if e.StartVertex().Point().Z >= maxZ-1e-9 && e.EndVertex().Point().Z >= maxZ-1e-9 {
			keys = append(keys, e.ReferenceKey())
		}
	}
	return keys
}

func countTorusF(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Torus); ok {
			n++
		}
	}
	return n
}

// TestFilletAllAroundTopRimThroughFeaturePath is the MISSING regression for Discord item #1 (#1797):
// rampam's exact interactive workflow, driven through the real FEATURE path (AddFillet + Recompute),
// NOT the raw ops.FilletEdges call the kernel test uses. Round the 4 vertical edges of a cube, then
// fillet the whole top perimeter (a closed tangent chain of 4 straight + 4 arc edges) — it must build
// ONE continuous rounded stripe (4 torus + straight-blend cylinders), a valid closed manifold solid,
// not a faceted/distorted cage. If this fails, the feature layer diverges from the green kernel test.
func TestFilletAllAroundTopRimThroughFeaturePath(t *testing.T) {
	t.Parallel()
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 4}, {X: 0, Y: 4}},
		sketch.XYPlane(), span{near: 0, far: 4}, 0, "box")

	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)

	vert := verticalEdgeKeysF(box)
	if len(vert) != 4 {
		t.Fatalf("expected 4 vertical edges on the cube, got %d", len(vert))
	}
	f1 := NewDressUpFeatures(fs).AddFillet(vert, func() float64 { return 0.5 })
	fs.Recompute()
	if !f1.Health().OK() {
		t.Fatalf("vertical-edge fillet sick: %+v", f1.Health())
	}

	top := topPerimeterKeysF(fs.Result()[0])
	if len(top) != 8 {
		t.Fatalf("expected 8 top-perimeter edges after vertical fillet, got %d", len(top))
	}
	f2 := NewDressUpFeatures(fs).AddFillet(top, func() float64 { return 0.25 })
	fs.Recompute()
	if !f2.Health().OK() {
		t.Fatalf("all-around top-rim fillet sick (rampam #1): %+v", f2.Health())
	}

	res := fs.Result()[0]
	rep := ops.Validate(res)
	if !rep.Valid || !res.IsSolid() || !rep.Manifold || !rep.Closed {
		t.Fatalf("top-rim fillet result invalid: valid=%v solid=%v manifold=%v closed=%v issues=%v",
			rep.Valid, res.IsSolid(), rep.Manifold, rep.Closed, rep.Issues)
	}
	if tori := countTorusF(res); tori != 4 {
		t.Errorf("torus faces = %d, want 4 (one per rounded corner) — a faceted/distorted result", tori)
	}
}
