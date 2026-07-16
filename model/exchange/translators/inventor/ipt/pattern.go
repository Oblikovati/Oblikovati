// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "math"

// Pattern feature decoding from PmDCSegment. A pattern replicates a feature; its two
// dimensions (count + spacing/angle) are stored as model parameters authored after the
// base features' dimensions (d0…), so they follow the extrude distances in order:
// [d0=extrude, …, dN=count, dN+1=spacing|angle].

// RectPattern is a decoded 1D rectangular pattern: Count occurrences spaced Spacing cm
// apart. v1 assumes the +X direction (the corpus default); arbitrary direction, the second
// grid axis, and mirror patterns are future work.
type RectPattern struct {
	Count   int
	Spacing float64
}

// CircPattern is a decoded circular pattern: Count occurrences spread over Angle radians
// (total sweep) about the Z axis through the origin (the corpus default). Arbitrary axis
// and partial-angle spacing modes are future work.
type CircPattern struct {
	Count int
	Angle float64
}

// DecodeRectPattern reports the part's rectangular pattern, if present (see patternDims).
func DecodeRectPattern(d *Document) (RectPattern, bool) {
	seg, ok := d.Segment("PmDCSegment")
	if !ok || !containsUTF16(seg, "Rectangular") || !containsUTF16(seg, "Pattern") {
		return RectPattern{}, false
	}
	count, spacing, ok := patternDims(d, seg)
	if !ok {
		return RectPattern{}, false
	}
	return RectPattern{Count: count, Spacing: spacing}, true
}

// DecodeCircPattern reports the part's circular pattern, if present (see patternDims); the
// second dimension is the total sweep in radians.
func DecodeCircPattern(d *Document) (CircPattern, bool) {
	seg, ok := d.Segment("PmDCSegment")
	if !ok || !containsUTF16(seg, "Circular") || !containsUTF16(seg, "Pattern") {
		return CircPattern{}, false
	}
	count, angle, ok := patternDims(d, seg)
	if !ok {
		return CircPattern{}, false
	}
	return CircPattern{Count: count, Angle: angle}, true
}

// patternDims returns a pattern's occurrence count and its second dimension (spacing cm for
// rectangular, sweep radians for circular): the two model parameters authored right after
// the base features' distances — count at index nExtrudes, the second at nExtrudes+1.
func patternDims(d *Document, seg []byte) (count int, second float64, ok bool) {
	mp := modelParamValues(seg)
	n := len(DecodeExtrudes(d))
	if n == 0 || len(mp) < n+2 {
		return 0, 0, false
	}
	count = int(math.Round(mp[n]))
	second = mp[n+1]
	if count < 2 || second <= 0 {
		return 0, 0, false
	}
	return count, second, true
}
