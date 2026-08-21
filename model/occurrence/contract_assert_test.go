// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import (
	"testing"

	"oblikovati.org/api/contract"
	"oblikovati.org/math"
)

// TestOccurrencePatternsViewMatchesTheSet drives the contract.OccurrencePatterns view (AsContract),
// so the read surface is exercised rather than only compile-asserted: Count and each Item mirror the
// set, and each element reads through the contract.OccurrencePattern interface.
func TestOccurrencePatternsViewMatchesTheSet(t *testing.T) {
	occs := NewOccurrences()
	seed := occs.AddByComponentDefinition("seed", unitComponent(), math.Identity4())
	gen := occs.AddByComponentDefinition("gen", unitComponent(), math.Translation4(math.V3(1, 0, 0)))
	set := NewOccurrencePatternSet(occs)
	pat := NewOccurrencePattern(unitComponent(), math.Identity4(),
		RectangularArrangement{Dir1: unitX(t), Spacing1: 1, Count1: 2, Dir2: unitY(t), Spacing2: 1, Count2: 1})
	added := set.Add(pat, "P1", seed, []*Occurrence{gen})

	var view contract.OccurrencePatterns = set.AsContract()
	if view.Count() != set.Count() || view.Count() != 1 {
		t.Fatalf("view.Count() = %d, want %d (= 1)", view.Count(), set.Count())
	}
	el := view.Item(0)
	if el.ID() != added.ID() {
		t.Errorf("view.Item(0).ID() = %d, want %d", el.ID(), added.ID())
	}
	if el.Count() != added.Count() {
		t.Errorf("view element Count() = %d, want %d", el.Count(), added.Count())
	}
	if el.Suppression() != added.Suppression() {
		t.Errorf("view element Suppression() = %v, want %v", el.Suppression(), added.Suppression())
	}
	// The concrete element also satisfies contract.OccurrencePattern directly.
	var _ contract.OccurrencePattern = added
}
