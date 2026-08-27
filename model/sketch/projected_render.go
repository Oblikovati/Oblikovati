// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Small 2D helpers shared by projected-curve rendering and the offset paths. The projected-curve
// analytic form is now a geom.Curve2 (ADR-0055); these are the remaining sampling/geometry bits.

// projectedRenderSegments is how finely an analytic projected curve is sampled for drawing and
// hit-testing, so a projected arc reads as a smooth curve rather than the source facets.
const projectedRenderSegments = 64

// perpDistanceToLine is the signed perpendicular distance from p to the infinite line through a→b.
func perpDistanceToLine(a, b, p math.Point2) float64 {
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	length := stdmath.Hypot(dx, dy)
	if length < 1e-12 {
		return float64(p.DistanceTo(a))
	}
	return (dx*float64(p.Y-a.Y) - dy*float64(p.X-a.X)) / length
}
