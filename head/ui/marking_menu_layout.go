// SPDX-License-Identifier: GPL-2.0-only

package ui

import "oblikovati.org/api/types"

// markingMenuRadius is the minimum distance from the popup centre to a ring slot.
// The layout grows it when the live font or icon scale makes neighbouring labels
// too tall to sit on separate radial rows.
const markingMenuRadius = 72

const (
	markingMenuPadding = 12
	markingMenuSlotGap = 8
)

// markingRingLayout is the geometry reserved for the radial portion of the popup.
// Slot coordinates are relative to the cursor position at which the ring starts.
type markingRingLayout struct {
	size          float32
	center        float32
	radius        float32
	maxSlotWidth  float32
	maxSlotHeight float32
}

// newMarkingRingLayout grows the ring enough for the tallest live slot and reserves
// the widest live slot on both sides of the radial centre. The fixed minimum keeps
// the default menu compact while the dynamic terms make long labels and scaled fonts
// safe instead of allowing them to overlap or clip at the popup edge.
func newMarkingRingLayout(maxSlotWidth, maxSlotHeight float32) markingRingLayout {
	if maxSlotWidth < 0 {
		maxSlotWidth = 0
	}
	if maxSlotHeight < 0 {
		maxSlotHeight = 0
	}
	radius := markingRingRequiredRadius(maxSlotWidth, maxSlotHeight)
	content := maxSlotWidth
	if maxSlotHeight > content {
		content = maxSlotHeight
	}
	size := 2*radius + content + 2*markingMenuPadding
	return markingRingLayout{
		size:          size,
		center:        size / 2,
		radius:        radius,
		maxSlotWidth:  maxSlotWidth,
		maxSlotHeight: maxSlotHeight,
	}
}

// markingRingRequiredRadius finds the smallest radius for which every adjacent pair
// of slot rectangles has at least one non-overlapping axis. A radial pair may be
// separated horizontally or vertically; requiring both axes would make the menu
// needlessly large for wide command labels.
func markingRingRequiredRadius(maxSlotWidth, maxSlotHeight float32) float32 {
	radius := float32(markingMenuRadius)
	quadrants := []types.ScreenQuadrant{
		types.QuadrantNorth, types.QuadrantNorthEast, types.QuadrantEast,
		types.QuadrantSouthEast, types.QuadrantSouth, types.QuadrantSouthWest,
		types.QuadrantWest, types.QuadrantNorthWest,
	}
	widthSpan := maxSlotWidth + markingMenuSlotGap
	heightSpan := maxSlotHeight + markingMenuSlotGap
	for i, q := range quadrants {
		next := quadrants[(i+1)%len(quadrants)]
		qx, qy := markingQuadrantDirection(q)
		nx, ny := markingQuadrantDirection(next)
		dx, dy := absoluteF32(qx-nx), absoluteF32(qy-ny)
		best := float32(1e9)
		if dx > 0 {
			best = widthSpan / dx
		}
		if dy > 0 && heightSpan/dy < best {
			best = heightSpan / dy
		}
		if best > radius {
			radius = best
		}
	}
	return radius
}

func absoluteF32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// markingQuadrantDirection returns the screen-space unit offset for one slot.
// Screen Y grows downward, so north is negative Y.
func markingQuadrantDirection(q types.ScreenQuadrant) (float32, float32) {
	switch q {
	case types.QuadrantNorth:
		return 0, -1
	case types.QuadrantNorthEast:
		return 0.7, -0.7
	case types.QuadrantEast:
		return 1, 0
	case types.QuadrantSouthEast:
		return 0.7, 0.7
	case types.QuadrantSouth:
		return 0, 1
	case types.QuadrantSouthWest:
		return -0.7, 0.7
	case types.QuadrantWest:
		return -1, 0
	case types.QuadrantNorthWest:
		return -0.7, -0.7
	default:
		return 0, 0
	}
}

// markingSlotPosition returns the top-left position of a slot rectangle relative
// to the ring origin, centring the rectangle on its compass direction.
func markingSlotPosition(layout markingRingLayout, q types.ScreenQuadrant, width, height float32) (float32, float32) {
	dx, dy := markingQuadrantDirection(q)
	return layout.center + dx*layout.radius - width/2,
		layout.center + dy*layout.radius - height/2
}
