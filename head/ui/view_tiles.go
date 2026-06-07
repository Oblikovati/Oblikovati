// SPDX-License-Identifier: GPL-2.0-only

// This file has no cgo build tag: the tile-rectangle layout is pure Go so it can be
// tested headlessly (like navigate.go). The cgo viewport panel positions each tile from
// these rects.

package ui

import "oblikovati/api/types"

// TileRect is one view tile's pixel rectangle, relative to the viewport content region's
// top-left.
type TileRect struct{ X, Y, W, H float32 }

// tileGutter is the pixel gap left between tiles, both for visual separation and as the
// hit zone for the draggable splitters.
const tileGutter = 6

// tileRects lays out the view tiles for a layout inside a w×h content region, returning
// one rect per visible tile. The number of tiles is min(layout's tiles, viewCount),
// clamped to 1..4 — so a document with fewer views than the layout wants falls back to a
// simpler split (e.g. a quad layout with two views shows two side by side). splitX/splitY
// (0..1) place the dividers; a gutter is left at each divider for separation and the
// splitter handle. Arrangements: two = left|right (or top/bottom for the vertical layout),
// three = a tall left pane plus two stacked on the right, four = a 2×2 grid.
func tileRects(layout types.ViewLayout, w, h float32, viewCount int, splitX, splitY float32) []TileRect {
	t := layout.Tiles()
	if viewCount < t {
		t = viewCount
	}
	if t < 1 {
		t = 1
	}
	g := float32(tileGutter)
	vx, vy := w*splitX, h*splitY // divider centres
	lw, rx := vx-g/2, vx+g/2     // left width, right column x
	rw := w - rx                 // right column width
	th, by := vy-g/2, vy+g/2     // top height, bottom row y
	bh := h - by                 // bottom row height
	switch t {
	case 2:
		if layout == types.LayoutTwoV {
			return []TileRect{{0, 0, w, th}, {0, by, w, bh}}
		}
		return []TileRect{{0, 0, lw, h}, {rx, 0, rw, h}}
	case 3:
		return []TileRect{{0, 0, lw, h}, {rx, 0, rw, th}, {rx, by, rw, bh}}
	case 4:
		return []TileRect{{0, 0, lw, th}, {rx, 0, rw, th}, {0, by, lw, bh}, {rx, by, rw, bh}}
	default:
		return []TileRect{{0, 0, w, h}}
	}
}

// tileAt returns the index of the tile whose rect contains (x,y), or -1. Used to route a
// pointer event to the tile (view) under the cursor.
func tileAt(rects []TileRect, x, y float32) int {
	for i, r := range rects {
		if x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			return i
		}
	}
	return -1
}
