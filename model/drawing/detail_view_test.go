// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
)

// frontForDetail adds a FRONT base view and returns it with its sheet bounds — the parent the
// detail tests magnify.
func frontForDetail(t *testing.T, vs *DrawingViews) (front *DrawingView, minX, minY, maxX, maxY float64) {
	t.Helper()
	frontBase(t, vs)
	front, _ = vs.ByName("FRONT")
	minX, minY, maxX, maxY, _ = front.BoundsMM()
	return front, minX, minY, maxX, maxY
}

// TestAddDetailViewClipsACorner magnifies a corner of a FRONT view: the boundary over a corner
// keeps that corner's edges (so the detail has curves) but fewer than the whole parent (the rest
// is clipped away).
func TestAddDetailViewClipsACorner(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	front, minX, _, maxX, maxY := frontForDetail(t, views)

	det, err := views.AddDetail(DetailViewSpec{
		Name: "DETAIL-A", ParentView: "FRONT", BoundaryX: maxX, BoundaryY: maxY,
		RadiusMM: (maxX - minX) / 2, Scale: 4, CenterX: 100, CenterY: 220,
	})
	if err != nil {
		t.Fatalf("AddDetail: %v", err)
	}
	if det.Type() != types.DrawingViewDetail || det.BaseViewName() != "FRONT" {
		t.Fatalf("detail view = type %v parent %q, want detail off FRONT", det.Type(), det.BaseViewName())
	}
	if det.CurveCount() == 0 {
		t.Fatal("corner detail clipped away all curves — clip mis-mapped")
	}
	if det.CurveCount() >= front.CurveCount() {
		t.Errorf("corner detail kept %d curves, parent has %d — clip did not restrict", det.CurveCount(), front.CurveCount())
	}
	if cx, cy, r := det.DetailBoundaryMM(); cx != maxX || cy != maxY {
		t.Errorf("detail boundary centre = (%g,%g,r=%g), want (%g,%g)", cx, cy, r, maxX, maxY)
	}
}

// TestDetailFullVsEmptyBoundary checks a boundary covering the whole view keeps every parent
// curve, while a tiny boundary in the view's hollow centre keeps none — bracketing the clip.
func TestDetailFullVsEmptyBoundary(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	front, minX, minY, maxX, maxY := frontForDetail(t, views)
	cx, cy := (minX+maxX)/2, (minY+maxY)/2

	full, err := views.AddDetail(DetailViewSpec{ParentView: "FRONT", BoundaryX: cx, BoundaryY: cy, RadiusMM: 2 * (maxX - minX), Scale: 2})
	if err != nil {
		t.Fatalf("AddDetail full: %v", err)
	}
	if full.CurveCount() != front.CurveCount() {
		t.Errorf("full-cover detail kept %d curves, want all %d", full.CurveCount(), front.CurveCount())
	}
	empty, err := views.AddDetail(DetailViewSpec{ParentView: "FRONT", BoundaryX: cx, BoundaryY: cy, RadiusMM: (maxX - minX) / 100, Scale: 2})
	if err != nil {
		t.Fatalf("AddDetail empty: %v", err)
	}
	if empty.CurveCount() != 0 {
		t.Errorf("tiny central detail kept %d curves, want 0 (hollow outline centre)", empty.CurveCount())
	}
}

// TestDetailRejectsNonBaseParent checks a detail can only magnify a base view.
func TestDetailRejectsNonBaseParent(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	if _, err := views.AddDetail(DetailViewSpec{ParentView: "NOPE", RadiusMM: 5, Scale: 2}); err == nil {
		t.Error("detail off a missing parent = ok, want error")
	}
}

// TestDetailRecipeRoundTrip checks a detail view's type, parent, boundary and scale survive
// persistence and that its curves re-project (and re-clip) on open.
func TestDetailRecipeRoundTrip(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	_, minX, _, maxX, maxY := frontForDetail(t, views)
	bx, by, r := maxX, maxY, (maxX-minX)/2 // a corner, so the clip keeps real geometry
	if _, err := views.AddDetail(DetailViewSpec{Name: "D1", ParentView: "FRONT", BoundaryX: bx, BoundaryY: by, RadiusMM: r, Scale: 3, CenterX: 100, CenterY: 220}); err != nil {
		t.Fatalf("AddDetail: %v", err)
	}
	v, ok := reopen(t, c).Sheets().Active().Views().ByName("D1")
	if !ok || v.Type() != types.DrawingViewDetail || v.Scale() != 3 {
		t.Fatalf("restored detail wrong: ok=%v type=%v scale=%g", ok, v.Type(), v.Scale())
	}
	if cx, _, rr := v.DetailBoundaryMM(); cx != bx || rr != r {
		t.Errorf("restored boundary = (%g,_,%g), want (%g,_,%g)", cx, rr, bx, r)
	}
	if v.CurveCount() == 0 {
		t.Error("restored detail re-projected no curves")
	}
}
