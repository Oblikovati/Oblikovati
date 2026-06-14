// SPDX-License-Identifier: GPL-2.0-only

package assembly

import "oblikovati.org/model/occurrence"

// Collision-stop (M12-F03) halts a drive when a moved component interferes with another. For
// V1 this is a broad-phase proxy: an overlap of world axis-aligned bounding boxes
// (Occurrence.RangeBox), a *query* between solve steps — not the contact solver (M12-F05) and
// not a true B-rep clash (a tighter narrow phase is a later refinement). It is intentionally
// conservative: a coarse box overlap is reported as interference.

// occurrencesInterfere reports whether any non-grounded occurrence's world bounding box
// overlaps another occurrence's box — the moved-component-hits-something test.
func occurrencesInterfere(occs *occurrence.Occurrences) bool {
	placed := placedOccurrences(occs)
	for i, moved := range placed {
		if moved.Grounded() {
			continue
		}
		box := moved.RangeBox()
		for j, other := range placed {
			if i == j {
				continue
			}
			if box.Intersects(other.RangeBox()) {
				return true
			}
		}
	}
	return false
}

// placedOccurrences returns the non-suppressed occurrences (those with a real placement).
func placedOccurrences(occs *occurrence.Occurrences) []*occurrence.Occurrence {
	var out []*occurrence.Occurrence
	for _, o := range occs.All() {
		if !o.Suppressed() {
			out = append(out, o)
		}
	}
	return out
}
