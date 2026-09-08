// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"oblikovati.org/math"
)

// CurveBox returns the exact axis-aligned bounding box a curve reaches over its parameter span
// [t0, t1] (given in either order). It bounds each world axis in closed form with [AxialExtent], so a
// conic arc's box covers the arc's true reach and not merely its two ends. ok=false for a curve kind
// with no closed-form extent; the caller must then bound that curve by its own sampling.
//
// A face bounded by ONE closed conic has both ends at the same point, so a box taken from loop
// endpoints alone degenerates to that point — and a resolution derived from it is the model-size
// floor rather than the face's own scale (ADR-0042, ADR-0061 stage 4).
//
// Example:
//
//	box, ok := geom.CurveBox(rim, 0, 1)
func CurveBox(c Curve3, t0, t1 float64) (math.Box, bool) {
	var lo, hi [3]float64
	for i, axis := range [3]math.Vector3{math.V3(1, 0, 0), math.V3(0, 1, 0), math.V3(0, 0, 1)} {
		l, h, ok := AxialExtent(c, t0, t1, math.P3(0, 0, 0), axis)
		if !ok {
			return math.EmptyBox(), false
		}
		lo[i], hi[i] = l, h
	}
	return math.NewBox(math.P3(lo[0], lo[1], lo[2]), math.P3(hi[0], hi[1], hi[2])), true
}

// CurveSpanBox bounds a curve over [t0, t1] whatever its kind: exactly through [CurveBox] where a
// closed-form axial extent exists, and by a fixed-count walk of the span otherwise. It is the
// "the caller must then bound that curve by its own sampling" half of CurveBox's contract, written
// ONCE instead of at each call site — the call site that skipped it (brep's faceLoopBox) kept an
// endpoint-only box, so a face bounded by a single CLOSED non-conic section — the torus
// figure-eight's two lobes, whose section is a SpiricArc — measured its own scale as a POINT and
// took the model-size floor (1e-9) as its resolution. Its stitch weld grid then came out at 1e-15,
// finer than the rounding of the coordinates it compares, and the two readings of the lobes' shared
// pinch point stayed unmerged wherever they were not bit-identical (CI run 34280554924
// macos-latest: arm64 fuses x*y+z, so they differed by 1.3e-15 and the seam tore open).
//
// The box sizes a TOLERANCE, never a topological decision, so a sampled bound is sound here: it lies
// inside the curve's true reach and is of the curve's own order, which is all a Resolution reads.
//
// Example:
//
//	res := geom.ResolutionForBox(geom.CurveSpanBox(lobe, t0, t1))
func CurveSpanBox(c Curve3, t0, t1 float64) math.Box {
	if box, ok := CurveBox(c, t0, t1); ok {
		return box
	}
	box := math.EmptyBox()
	for i := 0; i <= curveSpanSamples; i++ {
		box = box.ExtendPoint(c.PointAt(t0 + (t1-t0)*float64(i)/curveSpanSamples))
	}
	return box
}

// curveSpanSamples is how many segments a span with no closed-form extent is walked in. It fixes a
// SCALE, not a shape, so a coarse walk is enough: 32 segments hold a closed section's reach to well
// inside the order of magnitude a Resolution reads from the box.
const curveSpanSamples = 32
