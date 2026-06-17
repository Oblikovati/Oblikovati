// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestAddBreakViewCompressesAndGlyphs cuts a horizontal break through the middle of a FRONT
// view: the result keeps the model edges (minus the removed band), is narrower than the parent,
// and carries break-line glyph curves at the join.
func TestAddBreakViewCompressesAndGlyphs(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	front, _ := views.ByName("FRONT")
	minX, _, maxX, _, _ := front.BoundsMM()
	// Remove the middle third of the width.
	gapStart := minX + (maxX-minX)/3
	gapEnd := minX + 2*(maxX-minX)/3

	brk, err := views.AddBreak(BreakViewSpec{
		Name: "BREAK-A", ParentView: "FRONT", Orientation: types.BreakHorizontal,
		GapStart: gapStart, GapEnd: gapEnd, CenterX: 100, CenterY: 220,
	})
	if err != nil {
		t.Fatalf("AddBreak: %v", err)
	}
	if brk.Type() != types.DrawingViewBreak || brk.BreakOrientation() != types.BreakHorizontal {
		t.Fatalf("break view = type %v orient %v, want break/horizontal", brk.Type(), brk.BreakOrientation())
	}
	var glyphs int
	for _, cv := range brk.Curves() {
		if cv.Kind() == types.DrawingBreakCurve {
			glyphs++
		}
	}
	if glyphs == 0 {
		t.Error("break view has no break-line glyph curves")
	}
	// The compressed view is narrower than the parent by ~the removed band width.
	bMinX, _, bMaxX, _, _ := brk.BoundsMM()
	if bMaxX-bMinX >= maxX-minX {
		t.Errorf("break view width %g not narrower than parent %g — band not removed", bMaxX-bMinX, maxX-minX)
	}
}

// TestBreakRejectsNonBaseParent checks a break can only compress a base view.
func TestBreakRejectsNonBaseParent(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	if _, err := views.AddBreak(BreakViewSpec{ParentView: "NOPE", GapStart: 1, GapEnd: 2}); err == nil {
		t.Error("break off a missing parent = ok, want error")
	}
}

// TestBreakRecipeRoundTrip checks a break view's type, parent, orientation and gap survive
// persistence and that its curves (with glyphs) re-derive on open.
func TestBreakRecipeRoundTrip(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	front, _ := views.ByName("FRONT")
	minX, _, maxX, _, _ := front.BoundsMM()
	gs, ge := minX+(maxX-minX)/3, minX+2*(maxX-minX)/3
	if _, err := views.AddBreak(BreakViewSpec{Name: "BK", ParentView: "FRONT", Orientation: types.BreakVertical, GapStart: gs, GapEnd: ge, CenterX: 100, CenterY: 220}); err != nil {
		t.Fatalf("AddBreak: %v", err)
	}
	v, ok := reopen(t, c).Sheets().Active().Views().ByName("BK")
	if !ok || v.Type() != types.DrawingViewBreak || v.BreakOrientation() != types.BreakVertical {
		t.Fatalf("restored break wrong: ok=%v type=%v orient=%v", ok, v.Type(), v.BreakOrientation())
	}
	if start, end := v.BreakGapMM(); start != gs || end != ge {
		t.Errorf("restored break gap = (%g,%g), want (%g,%g)", start, end, gs, ge)
	}
	if v.CurveCount() == 0 {
		t.Error("restored break re-derived no curves")
	}
}
