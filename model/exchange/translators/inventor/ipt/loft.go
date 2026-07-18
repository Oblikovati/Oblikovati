// SPDX-License-Identifier: GPL-2.0-only

package ipt

// Loft feature decoding from PmDCSegment. A loft blends between profile sketches on parallel
// planes. v1 handles a loft whose sections lie on XY-parallel planes stacked along +Z: the
// first is on XY (z=0), and each later section's plane offset is a leading model parameter
// (the work-plane offsets, authored before the work-plane angle parameters). Rails/centreline
// guides and non-parallel section planes are future work.

// HasLoft reports whether the part has a loft feature ("Loft" node).
func HasLoft(seg []byte) bool {
	return containsUTF16(seg, "Loft")
}

// LoftSectionHeights returns the +Z height of each of the loft's sections in order — the
// first section sits on XY (0), and the remaining sections take the leading model parameters
// (the work-plane offsets). ok=false when the part has no loft or too few offsets for the
// section count.
func LoftSectionHeights(seg []byte, sections int) ([]float64, bool) {
	if !HasLoft(seg) || sections < 2 {
		return nil, false
	}
	mp := modelParamValues(seg)
	if len(mp) < sections-1 {
		return nil, false
	}
	heights := make([]float64, sections)
	for i := 1; i < sections; i++ {
		heights[i] = mp[i-1]
		if heights[i] <= 0 {
			return nil, false
		}
	}
	return heights, true
}
