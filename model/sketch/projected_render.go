// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Small 2D geometry helper shared by the offset paths (and the reference-curve test). A projected
// curve is now a concrete reference entity (ADR-0055 phase 3), so no projected-curve sampler lives
// here anymore.

// perpDistanceToLine is the signed perpendicular distance from p to the infinite line through a→b.
func perpDistanceToLine(a, b, p math.Point2) float64 {
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	length := stdmath.Hypot(dx, dy)
	if length < 1e-12 {
		return float64(p.DistanceTo(a))
	}
	return (dx*float64(p.Y-a.Y) - dy*float64(p.X-a.X)) / length
}
