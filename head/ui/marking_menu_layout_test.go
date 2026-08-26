// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/api/types"
)

func TestMarkingRingLayoutKeepsScaledSlotsInsideAndSeparate(t *testing.T) {
	layout := newMarkingRingLayout(160, 48)
	quadrants := []types.ScreenQuadrant{
		types.QuadrantNorth, types.QuadrantNorthEast, types.QuadrantEast,
		types.QuadrantSouthEast, types.QuadrantSouth, types.QuadrantSouthWest,
		types.QuadrantWest, types.QuadrantNorthWest,
	}
	for i, q := range quadrants {
		x, y := markingSlotPosition(layout, q, layout.maxSlotWidth, layout.maxSlotHeight)
		if x < 0 || y < 0 || x+layout.maxSlotWidth > layout.size || y+layout.maxSlotHeight > layout.size {
			t.Fatalf("slot %d (%v) escapes ring bounds: (%.1f, %.1f) in %.1f", i, q, x, y, layout.size)
		}
		for j := range i {
			px, py := markingSlotPosition(layout, quadrants[j], layout.maxSlotWidth, layout.maxSlotHeight)
			if rectanglesOverlap(x, y, layout.maxSlotWidth, layout.maxSlotHeight, px, py, layout.maxSlotWidth, layout.maxSlotHeight) {
				t.Fatalf("slots %v and %v overlap", q, quadrants[j])
			}
		}
	}
}

func rectanglesOverlap(ax, ay, aw, ah, bx, by, bw, bh float32) bool {
	return ax < bx+bw && bx < ax+aw && ay < by+bh && by < ay+ah
}
