// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	"oblikovati.org/math"
)

// twoBoxAssembly places two unit-ish boxes; boxB is offset by dx along X so the caller controls
// whether they overlap. Returns the assembly and the two occurrence ids.
func twoBoxAssembly(t *testing.T, dx float64) (*AssemblyComponentDefinition, uint64, uint64) {
	t.Helper()
	a := partWithBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2))
	b := partWithBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2))
	asm := NewAssemblyComponentDefinition()
	oa := asm.Place("a:1", a, math.Identity4())
	ob := asm.Place("b:1", b, math.Translation4(math.V3(dx, 0, 0)))
	return asm, oa.ID(), ob.ID()
}

// TestAnalyzeInterferenceDetectsOverlap covers AnalyzeInterference and its helpers: overlapping
// occurrences are reported with positive volume; separated ones are not.
func TestAnalyzeInterferenceDetectsOverlap(t *testing.T) {
	overlap, _, _ := twoBoxAssembly(t, 1) // B spans x∈[1,3], overlaps A's x∈[0,2]
	res := overlap.AnalyzeInterference(nil)
	if res.Count() != 1 {
		t.Fatalf("interference count = %d, want 1", res.Count())
	}
	if res.Total <= 0 {
		t.Errorf("interference total volume = %g, want > 0", res.Total)
	}

	apart, _, _ := twoBoxAssembly(t, 10) // B spans x∈[10,12], no overlap
	if got := apart.AnalyzeInterference(nil).Count(); got != 0 {
		t.Errorf("separated parts report %d interferences, want 0", got)
	}
}

// TestAnalyzeInterferenceSubsetFilter covers pairInSubset/idInSet: an empty-but-non-nil subset
// (a pair not naming both ids) excludes the pair.
func TestAnalyzeInterferenceSubsetFilter(t *testing.T) {
	asm, a, b := twoBoxAssembly(t, 1)
	if got := asm.AnalyzeInterference([]uint64{a, b}).Count(); got != 1 {
		t.Errorf("subset naming both ids: count = %d, want 1", got)
	}
	if got := asm.AnalyzeInterference([]uint64{a, a}).Count(); got != 0 {
		t.Errorf("subset not naming b: count = %d, want 0", got)
	}
}

// TestWouldContactBlockWithoutContactSet covers the early no-block path: an occurrence in no
// contact set never blocks a move.
func TestWouldContactBlockWithoutContactSet(t *testing.T) {
	asm, a, _ := twoBoxAssembly(t, 1)
	if asm.WouldContactBlock(a, math.Translation4(math.V3(0.5, 0, 0))) {
		t.Error("WouldContactBlock should be false when the occurrence shares no contact set")
	}
}
