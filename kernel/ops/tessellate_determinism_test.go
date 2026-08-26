// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
)

// Tessellation must be a PURE function of the body: the same body re-tessellated must give a byte-for-
// byte identical mesh. It briefly was not — the constrained-Delaunay cavity and the fold-repair pass
// both iterated Go maps, whose order is randomized per range, so an imported solid's free-edge count
// flickered run-to-run (filleted_box 36↔29) and no watertightness metric was reproducible. These tests
// pin determinism at the CDT, at the fold-repair edge order, and end-to-end on an imported solid.

// TestConstrainedDelaunayDeterministic runs the CDT many times on one input (a square with an interior
// Steiner grid — enough points to force repeated cavity rebuilds) and requires identical output every
// time. This is the direct guard on the cavity-order fix (cdt_build.go).
func TestConstrainedDelaunayDeterministic(t *testing.T) {
	pts := [][2]float64{{0, 0}, {6, 0}, {6, 6}, {0, 6}}
	for i := 1; i < 6; i++ {
		for j := 1; j < 6; j++ {
			pts = append(pts, [2]float64{float64(i), float64(j)})
		}
	}
	first := constrainedDelaunay(pts, [][]int{{0, 1, 2, 3}})
	if len(first) == 0 {
		t.Fatal("constrainedDelaunay returned no triangles")
	}
	for run := range 200 {
		got := constrainedDelaunay(pts, [][]int{{0, 1, 2, 3}})
		if !triIndexEqual(first, got) {
			t.Fatalf("run %d: constrainedDelaunay non-deterministic (%d vs %d tris / differing order)", run, len(first), len(got))
		}
	}
}

// TestSortedEdgeKeysOrders pins that fold-repair visits edges in a fixed (a,b) order, not Go's
// randomized map order — the property that makes repairFoldPass reproducible.
func TestSortedEdgeKeysOrders(t *testing.T) {
	adj := map[edgeKey][]int{{2, 5}: nil, {0, 9}: nil, {2, 3}: nil, {0, 1}: nil}
	got := sortedEdgeKeys(adj)
	want := []edgeKey{{0, 1}, {0, 9}, {2, 3}, {2, 5}}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sortedEdgeKeys[%d] = %v; want %v (order %v)", i, got[i], want[i], got)
		}
	}
}

// TestTessellateBodyDeterministicOnImportedSolid re-tessellates the SAME imported body and requires an
// identical mesh — filleted_box is the fixture whose free-edge count used to flicker; sphere exercises
// the sphere-cap CDT path. End-to-end guard on TessellateBody.
func TestTessellateBodyDeterministicOnImportedSolid(t *testing.T) {
	dir := filepath.Join("..", "exchange", "step", "testdata", "occ")
	for _, name := range []string{"filleted_box", "sphere"} {
		data, err := os.ReadFile(filepath.Join(dir, name+".step"))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
		if err != nil || len(bodies) == 0 {
			t.Fatalf("import %s: %v (n=%d)", name, err, len(bodies))
		}
		q := DefaultQuality()
		first, _ := TessellateBody(bodies[0], q)
		for run := range 20 {
			got, _ := TessellateBody(bodies[0], q)
			if !meshIdentical(first, got) {
				t.Fatalf("%s run %d: TessellateBody non-deterministic on a fixed body", name, run)
			}
		}
	}
}

// triIndexEqual reports whether two triangle-index lists are identical in order.
func triIndexEqual(a, b [][3]int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// meshIdentical reports whether two meshes have byte-for-byte equal positions and indices.
func meshIdentical(a, b *Mesh) bool {
	if len(a.Positions) != len(b.Positions) || len(a.Indices) != len(b.Indices) {
		return false
	}
	for i := range a.Positions {
		if a.Positions[i] != b.Positions[i] {
			return false
		}
	}
	for i := range a.Indices {
		if a.Indices[i] != b.Indices[i] {
			return false
		}
	}
	return true
}
