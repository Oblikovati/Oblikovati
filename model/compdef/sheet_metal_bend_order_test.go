// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sheetmetal"
)

// twoBendSheet builds a sheet with two flanges (on opposite top edges) so the bend order has
// something to reorder, returning the part and the two flange feature names.
func twoBendSheet(t *testing.T) (*PartComponentDefinition, string, string) {
	t.Helper()
	d := NewPartComponentDefinition()
	if _, err := d.EnableSheetMetal(); err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	addSquareFace(d, 4)
	// Resolve each edge against the current body (the first flange's union relabels keys, so
	// the y=4 edge must be found after the y=0 flange is built).
	f1 := addFlangeOnEdge(t, d, topEdgeAt(t, d, 0).ReferenceKey())
	f2 := addFlangeOnEdge(t, d, topEdgeAt(t, d, 4).ReferenceKey())
	return d, f1, f2
}

// topEdgeAt returns the highest X-aligned edge at the given Y of the running body (the base
// boundary edge at that Y, even after another flange has raised the body elsewhere).
func topEdgeAt(t *testing.T, d *PartComponentDefinition, y float64) *topo.Edge {
	t.Helper()
	var best *topo.Edge
	bestZ := -1e18
	for _, e := range d.Features().Result()[0].Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		dx, dy := a.X-b.X, a.Y-b.Y
		alongX := (dx > 1e-6 || dx < -1e-6) && dy < 1e-6 && dy > -1e-6
		if alongX && a.Y > y-1e-6 && a.Y < y+1e-6 && a.Z > bestZ {
			best, bestZ = e, a.Z
		}
	}
	if best == nil {
		t.Fatalf("no X-edge at y=%g", y)
	}
	return best
}

// addFlangeOnEdge flanges the edge (by reference key) and returns the flange feature's name.
func addFlangeOnEdge(t *testing.T, d *PartComponentDefinition, edgeKey []byte) string {
	t.Helper()
	pf := feature.NewSheetMetalFlangeFeatures(d.Features()).Add(&feature.SheetMetalFlangeDefinition{
		EdgeKey: edgeKey, Height: func() float64 { return 1 },
	})
	d.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("flange unhealthy: %s", pf.Health().Reason)
	}
	return pf.Name()
}

// TestBendOrderReorderAndPersist the default bend order is creation order; setting a custom
// order reorders the bends and persists through the recipe.
func TestBendOrderReorderAndPersist(t *testing.T) {
	d, f1, f2 := twoBendSheet(t)

	nat := d.OrderedBends()
	if len(nat) != 2 || nat[0].Feature != f1 || nat[1].Feature != f2 {
		t.Fatalf("natural order = %v, want [%s %s]", featureNames(nat), f1, f2)
	}

	if err := d.SetBendOrder([]string{f2, f1}); err != nil {
		t.Fatalf("SetBendOrder: %v", err)
	}
	if got := d.OrderedBends(); got[0].Feature != f2 || got[1].Feature != f1 {
		t.Errorf("reordered = %v, want [%s %s]", featureNames(got), f2, f1)
	}
	if err := d.SetBendOrder([]string{"NoSuchBend"}); err == nil {
		t.Error("an unknown bend name must error")
	}

	blob, _ := d.MarshalRecipe()
	dst := NewPartComponentDefinition()
	if err := dst.ApplyRecipe(blob); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if got := dst.OrderedBends(); len(got) != 2 || got[0].Feature != f2 {
		t.Errorf("restored order = %v, want %s first", featureNames(got), f2)
	}
}

func featureNames(bends []sheetmetal.Bend) []string {
	out := make([]string, len(bends))
	for i, b := range bends {
		out[i] = b.Feature
	}
	return out
}
