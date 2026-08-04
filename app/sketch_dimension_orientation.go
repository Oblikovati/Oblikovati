// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Which distance a placed dimension means (#2025). The model has measured aligned, horizontal
// and vertical distances since #1869 — |P2−P1|, |Δx| and |Δy|, each with its own solver residual
// — but nothing ever asked for anything but aligned, so a diagonal line could only ever carry a
// dimension along itself. These are different CONSTRAINTS, not different renderings: a
// horizontal dimension on a diagonal line fixes ΔX and leaves ΔY free.
//
// As in Inventor, the choice comes from where the label is dragged, which is why this needed the
// placement click from #2022 to exist first.

// orientationForPlacement picks the distance orientation implied by placing a dimension's label
// at `at`, for a measurement between a and b.
//
// Each orientation draws its dimension line along a different direction — aligned runs along the
// measured segment, horizontal along X, vertical along Y — and the label is dragged AWAY from
// that line. So the orientation whose line direction is most perpendicular to the drag is the one
// the user is asking for: dragging off a diagonal at right angles gives aligned, dragging
// straight up gives the horizontal (ΔX) distance, dragging sideways gives the vertical (ΔY) one.
//
// Ties resolve to aligned, which keeps a horizontal or vertical line — where the candidates
// coincide — on the plain Euclidean measurement.
func orientationForPlacement(a, b, at math.Point2) sketch.DistanceOrientation {
	drag := a.Midpoint(b).VectorTo(at)
	if drag.Length() < math.DefaultTolerance {
		return sketch.AlignedDistance // dropped on the segment itself: no direction to read
	}
	drag = drag.Scale(1 / drag.Length())
	best, bestDot := sketch.AlignedDistance, stdmath.Abs(alignedDir(a, b).Dot(drag))
	for _, c := range []struct {
		orientation sketch.DistanceOrientation
		dir         math.Vector2
	}{
		{sketch.HorizontalDistance, math.V2(1, 0)},
		{sketch.VerticalDistance, math.V2(0, 1)},
	} {
		if d := stdmath.Abs(c.dir.Dot(drag)); d < bestDot {
			best, bestDot = c.orientation, d
		}
	}
	return best
}

// alignedDir is the unit direction of the measured segment, +X for a degenerate one.
func alignedDir(a, b math.Point2) math.Vector2 {
	v := a.VectorTo(b)
	if v.Length() < math.DefaultTolerance {
		return math.V2(1, 0)
	}
	return v.Scale(1 / v.Length())
}
