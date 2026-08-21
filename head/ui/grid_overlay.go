//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// gridCells is how many grid cells extend from the origin in each direction (so the
// grid spans 2·gridCells·spacing on each axis). Caps the line count regardless of
// spacing.
const gridCells = 20

// gridCoplanarNudge is the fraction of the local coordinate magnitude by which the sketch grid is
// pushed off its host plane toward the eye (#2087). The grid is coplanar with the face it is drawn
// on; a constant depth bias cannot win that z-fight at every zoom, because the fixed world-space
// coplanarity error maps to an ever-larger NDC gap as the plane approaches the near plane on
// zoom-in (the reported "zooming in hides the grid"). A world-space nudge that scales with the
// coordinate magnitude — the float32 rounding error's own scale, ~1.2e-7·|coord| — instead of with
// zoom stays a fixed few-hundred-ULP push at all zooms: always above the coplanarity error, always
// far below any real feature gap (sub-micron for a centimetre-scale part), so nearer model geometry
// still occludes the grid as intended (#909).
const gridCoplanarNudge = 1e-5

// gridOverlay builds the sketch grid: lines parallel to the plane's axes, spaced by
// spacing (model units), centred on the plane origin (the sketch's 0,0). Minor, major
// (every `major` lines) and the two origin axis lines are separate colored items: the
// origin lines take the universal CAD axis colors (X red, Y green) so users read axis
// identity by color exactly as on the orientation triad. Returns nil when spacing is
// non-positive.
func gridOverlay(plane sketch.Plane, spacing float64, major int, eye math.Point3) []renderer.DrawItem {
	if spacing <= 0 {
		return nil
	}
	minor, maj := &segAccum{}, &segAccum{}
	xAxis, yAxis := &segAccum{}, &segAccum{}
	half := float64(gridCells) * spacing
	plane = nudgedTowardEye(plane, eye, half) // #2087: lift the grid a hair off its host face
	for i := -gridCells; i <= gridCells; i++ {
		c := float64(i) * spacing
		uAcc, vAcc := gridLineAccum(i, major, minor, maj, xAxis, yAxis)
		uAcc.seg(plane, math.P2(c, -half), math.P2(c, half)) // line of constant u (the Y axis at i==0)
		vAcc.seg(plane, math.P2(-half, c), math.P2(half, c)) // line of constant v (the X axis at i==0)
	}
	var items []renderer.DrawItem
	items = appendGrid(items, minor, chromeTheme.gridMinorColor)
	items = appendGrid(items, maj, chromeTheme.gridMajorColor)
	items = appendGrid(items, xAxis, axisColorX)
	items = appendGrid(items, yAxis, axisColorY)
	return items
}

// nudgedTowardEye returns plane translated along its normal toward eye by a tiny, zoom-independent
// offset, so the coplanar grid wins the depth test against its host face at any zoom (#2087). The
// offset scales with the coordinate magnitude (the plane origin's distance from the world origin,
// or the grid's own half-extent, whichever is larger) — the scale of the float32 rounding error it
// must overcome — never with the eye distance, so zooming in does not shrink it below that error.
func nudgedTowardEye(plane sketch.Plane, eye math.Point3, halfExtent float64) sketch.Plane {
	n := plane.Normal().AsVector()
	scale := stdmath.Max(halfExtent, float64(plane.Origin().DistanceTo(math.P3(0, 0, 0))))
	eps := gridCoplanarNudge * stdmath.Max(scale, 1)
	if plane.Origin().VectorTo(eye).Dot(n) < 0 { // eye on the −normal side: push that way instead
		eps = -eps
	}
	shifted, err := sketch.NewPlane(plane.Origin().TranslateBy(n.Scale(math.Scalar(eps))), plane.XAxis(), plane.YAxis())
	if err != nil {
		return plane // degenerate axes can't happen for a real host plane; keep the grid on-plane
	}
	return shifted
}

// gridLineAccum returns the accumulators for the two grid lines at index i: the
// constant-u line (uAcc) and the constant-v line (vAcc). Off-origin lines share one
// minor/major group; at the origin (i==0) the constant-u line is the Y axis and the
// constant-v line is the X axis, so each goes to its own axis-colored group.
func gridLineAccum(i, major int, minor, maj, xAxis, yAxis *segAccum) (uAcc, vAcc *segAccum) {
	if i == 0 {
		return yAxis, xAxis
	}
	if major > 0 && i%major == 0 {
		return maj, maj
	}
	return minor, minor
}

// appendGrid adds a colored line item for a non-empty accumulator.
// appendStroke is appendGrid with a stroke width in pixels (#2015): above a hairline the viewport
// expands the segments into screen-space quads, so a line weight holds its width at any zoom.
func appendStroke(items []renderer.DrawItem, acc *segAccum, color [4]float32, width float32) []renderer.DrawItem {
	if len(acc.pos) == 0 {
		return items
	}
	return append(items, renderer.DrawItem{
		Primitive: renderer.Lines, Positions: acc.pos, Indices: acc.idx, Color: color, Width: width,
	})
}

func appendGrid(items []renderer.DrawItem, acc *segAccum, color [4]float32) []renderer.DrawItem {
	if len(acc.pos) == 0 {
		return items
	}
	return append(items, renderer.DrawItem{
		Primitive: renderer.Lines, Positions: acc.pos, Indices: acc.idx, Color: color,
	})
}
