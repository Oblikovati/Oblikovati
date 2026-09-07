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
