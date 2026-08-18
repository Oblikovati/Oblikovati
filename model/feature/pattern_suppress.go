// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	"sort"
)

// Dropping individual pattern occurrences (Oblikovati#1889).
//
// patternBase has carried a per-index suppression set since the pattern feature was written, but
// nothing outside the model could set it and nothing persisted it — the classic unreachable-API
// shape. This file gives it a validated entry point and a reader for the recipe.
//
// Element 0 cannot be suppressed. It is not a copy: it is the source features' own material, cut or
// joined into the running body by those features earlier in the recipe, before the pattern ran. The
// pattern only ever ADDS occurrences 1…n−1 on top of that state, and Input carries no pre-source
// body to rebuild the seed's absence from. Refusing it is therefore the honest answer — a silent
// no-op would leave the caller believing an occurrence had gone. To lose the seed, suppress the
// source feature itself, which is where that material comes from.

// SuppressElements replaces the set of suppressed occurrences (effective next recompute).
// Suppressing element 0 is refused; see the note above.
func (p *patternBase) SuppressElements(indices []int) error {
	next := map[int]bool{}
	for _, i := range indices {
		if i == 0 {
			return fmt.Errorf("pattern: element 0 is the seed — the source features' own material, " +
				"already applied before the pattern ran — so it cannot be suppressed here; " +
				"suppress the source feature instead")
		}
		if i < 0 {
			return fmt.Errorf("pattern: element index %d is negative; occurrences are numbered from 0", i)
		}
		next[i] = true
	}
	p.suppressed = next
	p.applySuppression()
	return nil
}

// applySuppression re-stamps the element list from the suppression set, so a caller that reads
// Elements() before the next recompute sees what it just set.
func (p *patternBase) applySuppression() {
	for i := range p.elements {
		p.elements[i].Suppressed = p.suppressed[p.elements[i].Index]
	}
}

// SuppressedIndices returns the suppressed occurrence indices in ascending order. It reads the
// suppression SET rather than the element list, so it answers before the first recompute has built
// any elements — which is what the recipe writer needs.
func (p *patternBase) SuppressedIndices() []int {
	out := make([]int, 0, len(p.suppressed))
	for i, on := range p.suppressed {
		if on {
			out = append(out, i)
		}
	}
	sort.Ints(out)
	return out
}
