// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/math"
)

// TestFlattenStopsOnCyclicDAG builds a definition cycle — assembly A contains B, B contains A —
// and checks PlacedBodies terminates with a bounded result instead of recursing forever (M34-F6).
// Without the cycle guard this test would hang / overflow the stack.
func TestFlattenStopsOnCyclicDAG(t *testing.T) {
	part := partWithBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	a := NewAssemblyComponentDefinition()
	b := NewAssemblyComponentDefinition()
	a.Place("p", part, math.Identity4())
	a.Place("toB", b, math.Identity4())
	b.Place("toA", a, math.Identity4())

	placed := a.PlacedBodies() // must return, not hang
	if len(placed) == 0 {
		t.Fatal("expected the part to be emitted at least once")
	}
	if len(placed) > 8 {
		t.Errorf("cyclic flatten emitted %d bodies; guard should keep it small/bounded", len(placed))
	}
}

// TestFlattenDepthCapBoundsDeepChain nests assemblies far past maxAssemblyDepth with a part only at
// the very bottom, and a part at the top. The shallow part is emitted; the part beyond the depth cap
// is not — proving the depth backstop engages and the walk stays finite (M34-F6).
func TestFlattenDepthCapBoundsDeepChain(t *testing.T) {
	part := partWithBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	top := NewAssemblyComponentDefinition()
	top.Place("shallow", part, math.Identity4())

	cur := top
	for i := 0; i < maxAssemblyDepth+50; i++ {
		next := NewAssemblyComponentDefinition()
		cur.Place("child", next, math.Identity4())
		cur = next
	}
	cur.Place("deep", part, math.Identity4()) // sits below the depth cap

	placed := top.PlacedBodies()
	if len(placed) != 1 {
		t.Fatalf("got %d bodies, want 1 (only the shallow part; the deep one is past the cap)", len(placed))
	}
}
