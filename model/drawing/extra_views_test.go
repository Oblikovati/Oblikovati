// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestAddSliceViewIsCutOutlineOnly checks a slice view keeps only the section cut outline — no
// retained-half edges, no hatch (a zero-thickness slice).
func TestAddSliceViewIsCutOutlineOnly(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	front, _ := views.ByName("FRONT")
	minX, _, maxX, _, _ := front.BoundsMM()
	cy := func() float64 { _, lo, _, hi, _ := front.BoundsMM(); return (lo + hi) / 2 }()

	slice, err := views.AddSlice(SliceViewSpec{Name: "S1", ParentView: "FRONT", X1: minX - 5, Y1: cy, X2: maxX + 5, Y2: cy, CenterX: 100, CenterY: 220})
	if err != nil {
		t.Fatalf("AddSlice: %v", err)
	}
	if slice.Type() != types.DrawingViewSlice {
		t.Fatalf("view type = %v, want slice", slice.Type())
	}
	for _, cv := range slice.Curves() {
		if cv.Kind() != types.DrawingSectionCurve {
			t.Errorf("slice curve kind = %v, want only section-cut outline", cv.Kind())
		}
	}
	if slice.CurveCount() == 0 {
		t.Error("slice produced no cut outline")
	}
}

// TestAddBreakoutRevealsInterior checks a breakout view turns the parent's hidden edges visible
// inside the region (revealing the interior) while keeping a boundary outline.
func TestAddBreakoutRevealsInterior(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	front, _ := views.ByName("FRONT")
	_, fh := front.VisibleHidden()
	minX, minY, maxX, maxY, _ := front.BoundsMM()
	cx, cy := (minX+maxX)/2, (minY+maxY)/2

	bo, err := views.AddBreakout(BreakoutViewSpec{Name: "BO", ParentView: "FRONT", BoundaryX: cx, BoundaryY: cy, RadiusMM: 2 * (maxX - minX), CenterX: 100, CenterY: 220})
	if err != nil {
		t.Fatalf("AddBreakout: %v", err)
	}
	bv, bh := bo.VisibleHidden()
	// A full-cover breakout reveals the parent's hidden edges (more visible, fewer hidden) and
	// adds the boundary outline curves.
	if fh > 0 && bh >= fh {
		t.Errorf("breakout hidden = %d, parent hidden = %d — interior not revealed", bh, fh)
	}
	if bv == 0 {
		t.Error("breakout has no visible curves")
	}
}

// TestAddDraftViewFramesRegion checks a model-less draft view produces a rectangular frame of
// the requested size and needs no model reference.
func TestAddDraftViewFramesRegion(t *testing.T) {
	c := NewContent() // no model reference / body resolver at all
	views := c.Sheets().Active().Views()
	d, err := views.AddDraft(DraftViewSpec{Name: "D1", WidthMM: 80, HeightMM: 50, CenterX: 100, CenterY: 100})
	if err != nil {
		t.Fatalf("AddDraft: %v", err)
	}
	if d.Type() != types.DrawingViewDraft {
		t.Fatalf("view type = %v, want draft", d.Type())
	}
	if d.CurveCount() != 4 {
		t.Errorf("draft frame = %d curves, want 4 (rectangle)", d.CurveCount())
	}
	minX, minY, maxX, maxY, ok := d.BoundsMM()
	if !ok || maxX-minX != 80 || maxY-minY != 50 {
		t.Errorf("draft bounds = %g×%g, want 80×50", maxX-minX, maxY-minY)
	}
}

// TestExtraViewsRecipeRoundTrip checks slice/breakout/draft survive persistence.
func TestExtraViewsRecipeRoundTrip(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	front, _ := views.ByName("FRONT")
	minX, minY, maxX, maxY, _ := front.BoundsMM()
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	if _, err := views.AddSlice(SliceViewSpec{Name: "S1", ParentView: "FRONT", X1: minX, Y1: cy, X2: maxX, Y2: cy}); err != nil {
		t.Fatalf("AddSlice: %v", err)
	}
	if _, err := views.AddBreakout(BreakoutViewSpec{Name: "BO", ParentView: "FRONT", BoundaryX: cx, BoundaryY: cy, RadiusMM: maxX - minX}); err != nil {
		t.Fatalf("AddBreakout: %v", err)
	}
	if _, err := views.AddDraft(DraftViewSpec{Name: "D1", WidthMM: 60, HeightMM: 40, CenterX: 200, CenterY: 200}); err != nil {
		t.Fatalf("AddDraft: %v", err)
	}
	r := reopen(t, c).Sheets().Active().Views()
	for _, want := range []struct {
		name string
		kind types.DrawingViewType
	}{{"S1", types.DrawingViewSlice}, {"BO", types.DrawingViewBreakout}, {"D1", types.DrawingViewDraft}} {
		v, ok := r.ByName(want.name)
		if !ok || v.Type() != want.kind {
			t.Errorf("restored %q = type %v (ok=%v), want %v", want.name, v.Type(), ok, want.kind)
		}
		if v.CurveCount() == 0 {
			t.Errorf("restored %q re-derived no curves", want.name)
		}
	}
}
