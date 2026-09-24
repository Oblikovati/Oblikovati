// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import "oblikovati.org/math"

// Spans bound the boundary-clearance query by what it can possibly touch.
//
// Every interior node of a chart covering asks whether any boundary segment lies within its clearance
// margin, and before spans it asked every segment of every chain that the chain's whole box admitted.
// A rim chain's box admits most of the face, so that was O(nodes × segments): profiled on
// occtparity's TestWaveETorusRimPerFaceAgainstDrawexe it was 59% of the run, 29% inside math.Hypot, and
// it made the test 12× slower than on develop (11.9 s → 148 s) once #3517 routed torus rims through the
// covering. On CI's smaller runners the same cost pushed model/feature past its 60-minute budget.
//
// A span is chainSpanLength consecutive segments with their own (u,v) box. Rim chains are traced in
// order, so a span's box is tight, and a node far from most of the rim skips most spans at the cost of
// four comparisons each.
//
// Skipping a span must never change the answer, because output is byte-identical and this decides
// which grid nodes the mesh keeps. The question is a pure disjunction — is ANY segment nearer than the
// margin? — so the order segments are asked in cannot matter; only a wrongly skipped segment could.
// A span is therefore skipped only when its box is more than spanSkipFactor margins away. The segment
// test's own rounding (the foot point ax+t·dx can land an ulp outside the box) is ~1e-13 of the
// coordinates, and the gap kept is a whole margin, so a skipped segment is one whose computed distance
// could not have been below the margin.

// chainSpanLength is how many consecutive segments one span covers: long enough that the span test
// replaces many Hypot calls, short enough that a span's box stays close to its segments.
const chainSpanLength = 16

// spanSkipFactor is how many margins a span's box must lie beyond before the span is skipped. It is a
// structural bound, not a tolerance: any factor above one leaves a gap the segment test's rounding
// cannot cross, and two leaves a whole margin.
const spanSkipFactor = 2

// chainSpan is segments [lo, hi) of a chain — segment i runs uv[i] → uv[i+1] — and their (u,v) box.
type chainSpan struct {
	lo, hi                 int
	uMin, uMax, vMin, vMax float64
}

// chainSpans cuts a traced chain into spans, in order.
//
//	c.spans = chainSpans(c.uv)
func chainSpans(uv []math.Point2) []chainSpan {
	var out []chainSpan
	for lo := 0; lo+1 < len(uv); lo += chainSpanLength {
		hi := min(lo+chainSpanLength, len(uv)-1)
		uMin, uMax, vMin, vMax := uvBBox(uv[lo : hi+1])
		out = append(out, chainSpan{lo: lo, hi: hi, uMin: uMin, uMax: uMax, vMin: vMin, vMax: vMax})
	}
	return out
}

// spanIsNear reports whether any segment of span s, shifted by sh, lies within margin of the scaled
// point (x, y). It is the per-segment test chainIsNear always ran, restricted to one span.
func (b *chartCover) spanIsNear(c chartChain, s chainSpan, sh [2]float64, x, y, margin float64) bool {
	for i := s.lo; i < s.hi; i++ {
		a, e := c.uv[i], c.uv[i+1]
		if distToSeg2D(x, y, (float64(a.X)+sh[0])*b.su, (float64(a.Y)+sh[1])*b.sv,
			(float64(e.X)+sh[0])*b.su, (float64(e.Y)+sh[1])*b.sv) < margin {
			return true
		}
	}
	return false
}
