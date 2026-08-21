// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"math"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/feature"
)

// topXEdges returns every X-aligned edge on the sheet's top face (the two opposite sides), so a test
// can flange more than one edge in one feature.
func topXEdges(t *testing.T, body *topo.Body) []*topo.Edge {
	t.Helper()
	maxZ := math.Inf(-1)
	for _, e := range body.Edges() {
		if z := e.StartVertex().Point().Z; z > maxZ {
			maxZ = z
		}
	}
	var out []*topo.Edge
	for _, e := range body.Edges() {
		a, b := e.StartVertex().Point(), e.EndVertex().Point()
		if math.Abs(a.X-b.X) > 1e-6 && math.Abs(a.Y-b.Y) < 1e-6 &&
			math.Abs(a.Z-maxZ) < 1e-6 && math.Abs(b.Z-maxZ) < 1e-6 {
			out = append(out, e)
		}
	}
	return out
}

// TestBendsMultiEdgeFlangeDevelopsEveryEdge is the flat-pattern half of #2071: one flange feature
// spanning several edge sets must develop a bend for EACH edge, not just one. Before this the flat
// read exactly one placement per feature, so the extra walls' bend allowances were missing.
func TestBendsMultiEdgeFlangeDevelopsEveryEdge(t *testing.T) {
	d := NewPartComponentDefinition()
	if _, err := d.EnableSheetMetal(); err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	addSquareFace(d, 4)
	edges := topXEdges(t, d.Features().Result()[0])
	if len(edges) < 2 {
		t.Fatalf("seed plate has %d top X-edges, want ≥2", len(edges))
	}
	pf := feature.NewSheetMetalFlangeFeatures(d.Features()).Add(&feature.SheetMetalFlangeDefinition{
		Height: func() float64 { return 1 },
		EdgeSets: []feature.FlangeEdgeSet{
			{EdgeKeys: [][]byte{edges[0].ReferenceKey()}},
			{EdgeKeys: [][]byte{edges[1].ReferenceKey()}},
		},
	})
	d.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("multi-edge flange sick: %s", pf.Health().Reason)
	}

	bends := d.Bends()
	if len(bends) != 2 {
		t.Fatalf("multi-edge flange developed %d bends, want 2 (one per flanged edge)", len(bends))
	}
	for i, b := range bends {
		if b.Feature != pf.Name() {
			t.Errorf("bend %d feature = %q, want %q", i, b.Feature, pf.Name())
		}
		if b.Allowance <= 0 {
			t.Errorf("bend %d allowance = %g, want a real developed length", i, b.Allowance)
		}
	}
	// Two identical 90° bends develop the same allowance, so the total is twice one bend's.
	if got, one := d.TotalBendAllowance(), bends[0].Allowance; math.Abs(got-2*one) > 1e-9 {
		t.Errorf("TotalBendAllowance = %g, want 2× one bend's %g", got, one)
	}
}
