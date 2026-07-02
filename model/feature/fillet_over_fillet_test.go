// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// adjacentTopEdgeKeys returns the two top edges of a box (both endpoints at z==top) that meet at
// the (hx,hy) corner — the pair whose miter fillet leaves a cylinder∩cylinder seam.
func adjacentTopEdgeKeys(t *testing.T, b *topo.Body, hx, hy, top float64) [][]byte {
	t.Helper()
	var keys [][]byte
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if float64(a.Z) != top || float64(c.Z) != top {
			continue
		}
		onY := float64(a.Y) == hy && float64(c.Y) == hy
		onX := float64(a.X) == hx && float64(c.X) == hx
		if onX || onY {
			keys = append(keys, e.ReferenceKey())
		}
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 adjacent top edges at corner (%g,%g), got %d", hx, hy, len(keys))
	}
	return keys
}

// cylinderSeamKey returns the key of an edge bounded by two cylinder faces (a miter fillet seam).
func cylinderSeamKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, e := range b.Edges() {
		fs := e.Faces()
		if len(fs) != 2 {
			continue
		}
		_, c0 := fs[0].Geometry().(geom.Cylinder)
		_, c1 := fs[1].Geometry().(geom.Cylinder)
		if c0 && c1 {
			return e.ReferenceKey()
		}
	}
	t.Fatal("no cylinder∩cylinder seam edge on the filleted body")
	return nil
}

// TestFilletOverFilletSeamGivesHonestReason reproduces live scenario 07: after filleting two
// adjacent top edges (a miter, leaving a cylinder∩cylinder seam), a second fillet on that seam edge
// is an unsupported fillet-over-fillet. It must go Sick with a reason that NAMES the curved cause —
// not the misleading "result is not a valid solid" the whole-body facet path produced (the second
// fillet was faceting the first fillet's cylinder into a triangle cage that could not close).
func TestFilletOverFilletSeamGivesHonestReason(t *testing.T) {
	box := buildPrism([]math.Point2{{X: 0, Y: 0}, {X: 4, Y: 0}, {X: 4, Y: 3}, {X: 0, Y: 3}},
		sketch.XYPlane(), span{near: 0, far: 2}, 0, "box")

	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(box)

	top := adjacentTopEdgeKeys(t, box, 4, 3, 2)
	f1 := NewDressUpFeatures(fs).AddFilletCorner(top, func() float64 { return 0.5 }, types.FilletCornerMiter)
	fs.Recompute()
	if !f1.Health().OK() {
		t.Fatalf("first (miter) fillet sick: %+v", f1.Health())
	}

	seam := cylinderSeamKey(t, fs.Result()[0])
	f2 := NewDressUpFeatures(fs).AddFillet([][]byte{seam}, func() float64 { return 0.2 })
	fs.Recompute()

	if f2.Health().OK() {
		t.Fatal("fillet on a curved (cylinder∩cylinder) seam should go Sick, but it reported healthy")
	}
	reason := f2.Health().Reason
	if !strings.Contains(reason, "curved") {
		t.Errorf("Sick reason should name the curved cause, got: %q", reason)
	}
	if strings.Contains(reason, "not a valid solid") {
		t.Errorf("Sick reason should be the honest curved-adjacent message, not the facet-cage failure: %q", reason)
	}
	if strings.Contains(reason, "fillet: fillet:") {
		t.Errorf("Sick reason should not double the feature-kind prefix: %q", reason)
	}
}

// TestPlanarizeCylinderForEdgesPrismifiesSimpleCylinder covers the one curved body the fillet's
// planarise DOES re-facet: a simple extrude-circle cylinder becomes a prism and its rim maps onto
// the prism's faceted segments (the #127/#129 path). Any other curved body is left analytic (the
// pass-through path is exercised by TestFilletOverFilletSeamGivesHonestReason).
func TestPlanarizeCylinderForEdgesPrismifiesSimpleCylinder(t *testing.T) {
	fs, rim := extrudedCylinderTopRim(t, 2, 5)
	body := fs.Result()[0]
	edges, _, err := resolveEdges(body, [][]byte{rim}, nil)
	if err != nil {
		t.Fatalf("resolve rim: %v", err)
	}
	pb, mapped := planarizeCylinderForEdges(body, edges, "fillet")
	if pb == body {
		t.Fatal("a simple cylinder must be prism-ified for the rim fillet, got the body unchanged")
	}
	if len(mapped) == 0 {
		t.Fatal("the rim edge must map onto the prism's faceted segments")
	}
}
