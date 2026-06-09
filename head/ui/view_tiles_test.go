// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/api/types"
)

func TestTileRectsCounts(t *testing.T) {
	cases := map[types.ViewLayout]int{
		types.LayoutSingle: 1,
		types.LayoutTwoH:   2,
		types.LayoutTwoV:   2,
		types.LayoutThree:  3,
		types.LayoutFour:   4,
	}
	for layout, want := range cases {
		got := tileRects(layout, 800, 600, 4, 0.5, 0.5)
		if len(got) != want {
			t.Errorf("%v: %d tiles, want %d", layout, len(got), want)
		}
	}
}

func TestTileRectsFallsBackWhenFewerViews(t *testing.T) {
	// A quad layout with only two views shows two side-by-side tiles (split at 400 with a
	// 6px gutter ⇒ left width 397, right starts at 403).
	got := tileRects(types.LayoutFour, 800, 600, 2, 0.5, 0.5)
	if len(got) != 2 {
		t.Fatalf("quad with 2 views = %d tiles, want 2", len(got))
	}
	if got[0].W != 397 || got[1].X != 403 {
		t.Errorf("expected a gutter-separated left|right split, got %+v", got)
	}
}

func TestTileRectsTwoHvsTwoV(t *testing.T) {
	h := tileRects(types.LayoutTwoH, 800, 600, 2, 0.5, 0.5)
	if h[0].W != 397 || h[0].H != 600 {
		t.Errorf("TwoH tile0 = %+v, want a left half minus gutter", h[0])
	}
	v := tileRects(types.LayoutTwoV, 800, 600, 2, 0.5, 0.5)
	if v[0].W != 800 || v[0].H != 297 {
		t.Errorf("TwoV tile0 = %+v, want a top half minus gutter", v[0])
	}
}

func TestTileRectsSplitMovesDivider(t *testing.T) {
	// A 0.25 vertical split makes the left tile a quarter wide.
	got := tileRects(types.LayoutTwoH, 800, 600, 2, 0.25, 0.5)
	if got[0].W != 197 { // 800*0.25 - 3
		t.Errorf("left tile width at split 0.25 = %g, want 197", got[0].W)
	}
}

func TestTileRectsCoverAndContain(t *testing.T) {
	rects := tileRects(types.LayoutFour, 800, 600, 4, 0.5, 0.5)
	// The four quad tiles tile the region; a point in each maps back to that tile.
	probes := []struct {
		x, y float32
		want int
	}{{200, 150, 0}, {600, 150, 1}, {200, 450, 2}, {600, 450, 3}}
	for _, p := range probes {
		if got := tileAt(rects, p.x, p.y); got != p.want {
			t.Errorf("tileAt(%g,%g) = %d, want %d", p.x, p.y, got, p.want)
		}
	}
	if tileAt(rects, 900, 900) != -1 {
		t.Error("point outside all tiles should be -1")
	}
}
