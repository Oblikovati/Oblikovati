//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// gridCells is how many grid cells extend from the origin in each direction (so the
// grid spans 2·gridCells·spacing on each axis). Caps the line count regardless of
// spacing.
const gridCells = 20

// gridOverlay builds the sketch grid: lines parallel to the plane's axes, spaced by
// spacing (model units), centred on the plane origin (the sketch's 0,0). Minor, major
// (every `major` lines) and the two origin axis lines are separate colored items: the
// origin lines take the universal CAD axis colors (X red, Y green) so users read axis
// identity by color exactly as on the orientation triad. Returns nil when spacing is
// non-positive.
func gridOverlay(plane sketch.Plane, spacing float64, major int) []renderer.DrawItem {
	if spacing <= 0 {
		return nil
	}
	minor, maj := &segAccum{}, &segAccum{}
	xAxis, yAxis := &segAccum{}, &segAccum{}
	half := float64(gridCells) * spacing
	for i := -gridCells; i <= gridCells; i++ {
		c := float64(i) * spacing
		uAcc, vAcc := gridLineAccum(i, major, minor, maj, xAxis, yAxis)
		uAcc.seg(plane, math.P2(c, -half), math.P2(c, half)) // line of constant u (the Y axis at i==0)
		vAcc.seg(plane, math.P2(-half, c), math.P2(half, c)) // line of constant v (the X axis at i==0)
	}
	var items []renderer.DrawItem
	items = appendGrid(items, minor, gridMinorColor)
	items = appendGrid(items, maj, gridMajorColor)
	items = appendGrid(items, xAxis, axisColorX)
	items = appendGrid(items, yAxis, axisColorY)
	return items
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
func appendGrid(items []renderer.DrawItem, acc *segAccum, color [4]float32) []renderer.DrawItem {
	if len(acc.pos) == 0 {
		return items
	}
	return append(items, renderer.DrawItem{
		Primitive: renderer.Lines, Positions: acc.pos, Indices: acc.idx, Color: color,
	})
}
