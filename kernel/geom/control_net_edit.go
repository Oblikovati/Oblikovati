// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// Direct NURBS control-net editing (M36-F03) — the Class-A styling interaction of dragging
// control points and watching the limit surface (and its reflections) follow. Unlike sub-D cage
// editing, this moves the *actual* B-spline control points, so the surface keeps its exact degree
// and knot vectors; only the control positions change. The geometry primitives live here (pure,
// testable); the feature and viewport handles build on them.

// ControlPointDelta moves the control point at grid index (U, V) by Delta.
type ControlPointDelta struct {
	U, V  int
	Delta math.Vector3
}

// Falloff selects how a region edit's displacement decays with distance from the dragged
// control points (the "drivers").
type Falloff int

const (
	// FalloffConstant moves every control point within the radius by the full delta (a rigid
	// bubble), the rest not at all.
	FalloffConstant Falloff = iota
	// FalloffLinear decays the delta linearly from 1 at a driver to 0 at the radius.
	FalloffLinear
	// FalloffSmooth decays with a smoothstep (zero slope at both ends) — the default styling feel.
	FalloffSmooth
)

// DisplaceControlPoints returns a copy of s with each listed control point moved by its delta,
// preserving the degree, knot vectors and weights (only control positions change). It errors on
// an out-of-range control index.
//
// Example: moved, _ := s.DisplaceControlPoints([]ControlPointDelta{{U:1,V:1,Delta:math.V3(0,0,1)}}).
func (s BSplineSurface) DisplaceControlPoints(deltas []ControlPointDelta) (BSplineSurface, error) {
	ctrl := copyNet(s.Ctrl)
	for _, d := range deltas {
		if d.U < 0 || d.U >= len(ctrl) || d.V < 0 || d.V >= len(ctrl[d.U]) {
			return BSplineSurface{}, fmt.Errorf("geom: control index (%d,%d) out of range for a %dx%d net", d.U, d.V, len(ctrl), len(s.Ctrl[0]))
		}
		ctrl[d.U][d.V] = ctrl[d.U][d.V].TranslateBy(d.Delta)
	}
	return NewBSplineSurface(s.UDegree, s.VDegree, ctrl, s.Weights, s.UKnots, s.VKnots)
}

// FalloffDeltas computes the per-control-point displacements for dragging the given driver
// control points by move, with the rest of the net following by a weight that falls from 1 at a
// driver to 0 at model-space distance radius (radius <= 0 ⇒ only the drivers move). Driving a
// single index is a single-CV edit; driving a whole row/column is a row/column edit; either with
// radius > 0 adds a soft region falloff. Returns only the control points that actually move.
func (s BSplineSurface) FalloffDeltas(drivers [][2]int, move math.Vector3, radius float64, falloff Falloff) []ControlPointDelta {
	var out []ControlPointDelta
	for i := range s.Ctrl {
		for j := range s.Ctrl[i] {
			w := falloffWeight(s.nearestDriverDistance(i, j, drivers), radius, falloff)
			if w == 0 {
				continue
			}
			out = append(out, ControlPointDelta{U: i, V: j, Delta: move.Scale(math.Scalar(w))})
		}
	}
	return out
}

// nearestDriverDistance returns the model-space distance from control point (i,j) to the closest
// driver control point (0 when (i,j) is itself a driver).
func (s BSplineSurface) nearestDriverDistance(i, j int, drivers [][2]int) float64 {
	best := stdmath.Inf(1)
	p := s.Ctrl[i][j]
	for _, d := range drivers {
		if dist := float64(p.DistanceTo(s.Ctrl[d[0]][d[1]])); dist < best {
			best = dist
		}
	}
	return best
}

// falloffWeight maps a distance-to-driver to a [0,1] displacement weight for the given falloff.
func falloffWeight(dist, radius float64, falloff Falloff) float64 {
	if dist == 0 {
		return 1
	}
	if radius <= 0 || dist >= radius {
		return 0
	}
	t := dist / radius // 0 at a driver, 1 at the radius
	switch falloff {
	case FalloffConstant:
		return 1
	case FalloffLinear:
		return 1 - t
	default: // FalloffSmooth
		return 1 - (3*t*t - 2*t*t*t) // 1 − smoothstep(t)
	}
}
