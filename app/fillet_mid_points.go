// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"sort"

	"oblikovati.org/model/feature"
)

// Intermediate radius stops for the variable-fillet panel (#695). A variable fillet blends
// each edge from a start radius to an end radius; a stop lets the radius pass through an
// arbitrary value partway along the edge, so the blend can swell or pinch between its ends.
// The panel edits the stops in insertion order; the kernel walks them sorted by T, so the
// ordering only matters when building the feature.

// FilletMidPoint is one intermediate radius stop the panel edits: T is the fraction along the
// edge (start vertex 0 → end vertex 1) and Radius the rolling-ball radius there (database units).
type FilletMidPoint struct {
	T      float64
	Radius float64
}

// MidPoints returns a copy of the intermediate radius stops for the panel to render.
func (t *FilletTool) MidPoints() []FilletMidPoint {
	return append([]FilletMidPoint(nil), t.midPoints...)
}

// AddMidPoint appends a stop at the midpoint of the widest gap in the current partition, its
// radius interpolated between the start and end radii — a valid default the user then nudges.
//
// Example: with no stops it lands at T=0.5; a second Add lands at T=0.25 or 0.75.
func (t *FilletTool) AddMidPoint() {
	tt := nextMidT(t.midPoints)
	r := t.startRadius + (t.endRadius-t.startRadius)*tt
	t.midPoints = append(t.midPoints, FilletMidPoint{T: tt, Radius: r})
}

// RemoveMidPoint drops the stop at index i (out-of-range is ignored).
func (t *FilletTool) RemoveMidPoint(i int) {
	if i < 0 || i >= len(t.midPoints) {
		return
	}
	t.midPoints = append(t.midPoints[:i], t.midPoints[i+1:]...)
}

// SetMidPointT edits the position of the stop at index i (out-of-range is ignored).
func (t *FilletTool) SetMidPointT(i int, v float64) {
	if i >= 0 && i < len(t.midPoints) {
		t.midPoints[i].T = v
	}
}

// SetMidPointR edits the radius of the stop at index i (out-of-range is ignored).
func (t *FilletTool) SetMidPointR(i int, v float64) {
	if i >= 0 && i < len(t.midPoints) {
		t.midPoints[i].Radius = v
	}
}

// nextMidT picks the default T for a new stop: the midpoint of the widest gap in the sorted
// [0, existing T…, 1] partition, so successive Adds spread out instead of stacking on 0.5.
func nextMidT(pts []FilletMidPoint) float64 {
	bounds := []float64{0, 1}
	for _, p := range pts {
		bounds = append(bounds, p.T)
	}
	sort.Float64s(bounds)
	lo, hi, widest := 0.0, 1.0, -1.0
	for i := 1; i < len(bounds); i++ {
		if w := bounds[i] - bounds[i-1]; w > widest {
			widest, lo, hi = w, bounds[i-1], bounds[i]
		}
	}
	return (lo + hi) / 2
}

// radiusPoints converts the panel's stops into feature radius-point closures sorted by T — the
// order the kernel's varying blend walks the edge (#695). Empty when there are no stops.
func (t *FilletTool) radiusPoints() []feature.FilletRadiusPoint {
	pts := t.MidPoints()
	sort.Slice(pts, func(a, b int) bool { return pts[a].T < pts[b].T })
	out := make([]feature.FilletRadiusPoint, len(pts))
	for i, p := range pts {
		r := p.Radius // capture per stop so the closure doesn't read the loop variable
		out[i] = feature.FilletRadiusPoint{T: p.T, Radius: func() float64 { return r }}
	}
	return out
}

// midPointsValid reports whether the stops satisfy the kernel's validateRadiusPoints contract:
// strictly-increasing interior fractions (0<T<1) with a positive radius (#695). No stops is valid.
func (t *FilletTool) midPointsValid() bool {
	pts := t.MidPoints()
	sort.Slice(pts, func(a, b int) bool { return pts[a].T < pts[b].T })
	prev := 0.0
	for _, p := range pts {
		if p.T <= prev || p.T >= 1 || p.Radius <= 0 {
			return false
		}
		prev = p.T
	}
	return true
}

// midPointsFromSet seeds the panel's stops from a committed variable set's radius points,
// evaluating each closure once for the editable buffer (edit mode).
func midPointsFromSet(pts []feature.FilletRadiusPoint) []FilletMidPoint {
	out := make([]FilletMidPoint, len(pts))
	for i, p := range pts {
		out[i] = FilletMidPoint{T: p.T, Radius: callOrZeroFn(p.Radius)}
	}
	return out
}
