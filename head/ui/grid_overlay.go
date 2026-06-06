//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati/math"
	"oblikovati/model/sketch"
	"oblikovati/renderer"
)

// gridCells is how many grid cells extend from the origin in each direction (so the
// grid spans 2·gridCells·spacing on each axis). Caps the line count regardless of
// spacing.
const gridCells = 20

// gridOverlay builds the sketch grid: lines parallel to the plane's axes, spaced by
// spacing (model units), centred on the plane origin (the sketch's 0,0). Minor, major
// (every `major` lines) and the two axis lines are separate colored items. Returns nil
// when spacing is non-positive.
func gridOverlay(plane sketch.Plane, spacing float64, major int) []renderer.DrawItem {
	if spacing <= 0 {
		return nil
	}
	minor, maj, axis := &segAccum{}, &segAccum{}, &segAccum{}
	half := float64(gridCells) * spacing
	for i := -gridCells; i <= gridCells; i++ {
		acc := gridAccum(i, major, minor, maj, axis)
		c := float64(i) * spacing
		acc.seg(plane, math.P2(c, -half), math.P2(c, half)) // line of constant u
		acc.seg(plane, math.P2(-half, c), math.P2(half, c)) // line of constant v
	}
	var items []renderer.DrawItem
	items = appendGrid(items, minor, gridMinorColor)
	items = appendGrid(items, maj, gridMajorColor)
	items = appendGrid(items, axis, gridAxisColor)
	return items
}

// gridAccum routes line index i to the axis (0), major (every `major`), or minor group.
func gridAccum(i, major int, minor, maj, axis *segAccum) *segAccum {
	switch {
	case i == 0:
		return axis
	case major > 0 && i%major == 0:
		return maj
	default:
		return minor
	}
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
