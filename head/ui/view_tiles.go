// SPDX-License-Identifier: GPL-2.0-only

// This file has no cgo build tag: the tile-rectangle layout is pure Go so it can be
// tested headlessly (like navigate.go). The cgo viewport panel positions each tile from
// these rects.

package ui

import "oblikovati/api/types"

// TileRect is one view tile's pixel rectangle, relative to the viewport content region's
// top-left.
type TileRect struct{ X, Y, W, H float32 }

// tileRects lays out the view tiles for a layout inside a w×h content region, returning
// one rect per visible tile. The number of tiles is min(layout's tiles, viewCount),
// clamped to 1..4 — so a document with fewer views than the layout wants falls back to a
// simpler split (e.g. a quad layout with two views shows two side by side). Arrangements:
// two = left|right (or top/bottom for the vertical layout), three = a tall left pane plus
// two stacked on the right, four = a 2×2 grid.
func tileRects(layout types.ViewLayout, w, h float32, viewCount int) []TileRect {
	t := layout.Tiles()
	if viewCount < t {
		t = viewCount
	}
	if t < 1 {
		t = 1
	}
	hw, hh := w/2, h/2
	switch t {
	case 2:
		if layout == types.LayoutTwoV {
			return []TileRect{{0, 0, w, hh}, {0, hh, w, h - hh}}
		}
		return []TileRect{{0, 0, hw, h}, {hw, 0, w - hw, h}}
	case 3:
		return []TileRect{{0, 0, hw, h}, {hw, 0, w - hw, hh}, {hw, hh, w - hw, h - hh}}
	case 4:
		return []TileRect{
			{0, 0, hw, hh}, {hw, 0, w - hw, hh},
			{0, hh, hw, h - hh}, {hw, hh, w - hw, h - hh},
		}
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
