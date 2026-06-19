// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestDerivedViewPreviews covers the non-committing preview generators for the derived-view
// tools (section/detail/slice/breakout/break) plus Item: each previews against a FRONT base view
// without mutating the sheet, and the geometry-producing ones return curves.
func TestDerivedViewPreviews(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	front, ok := views.ByName("FRONT")
	if !ok {
		t.Fatal("FRONT base view missing")
	}
	minX, minY, maxX, maxY, hasBounds := front.BoundsMM()
	if !hasBounds {
		t.Fatal("FRONT view has no bounds")
	}
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	r := (maxX - minX) / 4

	if curves, ok := views.PreviewSection("FRONT", minX-5, cy, maxX+5, cy); !ok || len(curves) == 0 {
		t.Errorf("PreviewSection: ok=%v curves=%d, want a cut preview", ok, len(curves))
	}
	if curves, ok := views.PreviewSlice("FRONT", minX-5, cy, maxX+5, cy); !ok || len(curves) == 0 {
		t.Errorf("PreviewSlice: ok=%v curves=%d, want a slice preview", ok, len(curves))
	}
	if _, ok := views.PreviewDetail("FRONT", cx, cy, r, 2); !ok {
		t.Error("PreviewDetail should accept a boundary inside the view")
	}
	// Breakout and break previews must at least execute without mutating the sheet.
	views.PreviewBreakout("FRONT", cx, cy, r)
	views.PreviewBreak("FRONT", types.BreakHorizontal, minY+1, maxY-1)

	if views.Count() != 1 || views.Item(0).Name() != "FRONT" {
		t.Errorf("previews must not add views: count=%d, Item(0)=%q", views.Count(), views.Item(0).Name())
	}
}
