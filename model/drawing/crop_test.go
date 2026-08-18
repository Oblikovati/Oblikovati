// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
)

// allCurvesInsideRect reports whether every curve endpoint lies within the rectangle (a small
// tolerance absorbs the clip's floating-point round-off).
func allCurvesInsideRect(v *DrawingView, x0, y0, x1, y1 float64) bool {
	const eps = 1e-6
	for _, c := range v.Curves() {
		for _, p := range [2][2]float64{{float64(c.A.X), float64(c.A.Y)}, {float64(c.B.X), float64(c.B.Y)}} {
			if p[0] < x0-eps || p[0] > x1+eps || p[1] < y0-eps || p[1] > y1+eps {
				return false
			}
		}
	}
	return true
}

// countBreakCurves tallies a view's break-mark curves.
func countBreakCurves(v *DrawingView) int {
	n := 0
	for _, c := range v.Curves() {
		if c.Kind() == types.DrawingBreakCurve {
			n++
		}
	}
	return n
}

// TestCropRectangleDropsOutsideRestoresOnRemove crops a base view to a rectangle covering part of
// it: curves outside the fence are dropped, the survivors lie inside it, and removing the crop
// restores the full curve set (#1987 AC 1 & 4).
func TestCropRectangleDropsOutsideRestoresOnRemove(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	front, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1, CenterX: 100, CenterY: 100})
	if err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	full := front.CurveCount()
	if full == 0 {
		t.Fatal("base view has no curves to crop")
	}
	minX, minY, maxX, maxY, _ := front.BoundsMM()
	// A fence covering the lower-left quadrant of the view — it must exclude some geometry.
	x0, y0 := minX-1, minY-1
	x1, y1 := (minX+maxX)/2, (minY+maxY)/2
	if _, err := views.AddCrop(CropSpec{View: "FRONT", X0: x0, Y0: y0, X1: x1, Y1: y1}); err != nil {
		t.Fatalf("AddCrop: %v", err)
	}
	if front.CurveCount() >= full {
		t.Errorf("cropped curve count %d not below full %d — nothing dropped", front.CurveCount(), full)
	}
	if front.CurveCount() == 0 {
		t.Fatal("crop dropped every curve — fence too small")
	}
	if !allCurvesInsideRect(front, x0, y0, x1, y1) {
		t.Error("a surviving curve lies outside the crop fence")
	}
	if err := views.RemoveCrop("FRONT"); err != nil {
		t.Fatalf("RemoveCrop: %v", err)
	}
	if front.CurveCount() != full {
		t.Errorf("after RemoveCrop curve count = %d, want the full %d restored", front.CurveCount(), full)
	}
}

// TestCropPersistsAcrossRecompute a crop is re-applied on every model recompute (#1987 AC 3).
func TestCropPersistsAcrossRecompute(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	front, _ := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1, CenterX: 100, CenterY: 100})
	full := front.CurveCount()
	minX, minY, maxX, maxY, _ := front.BoundsMM()
	if _, err := views.AddCrop(CropSpec{View: "FRONT", X0: minX - 1, Y0: minY - 1, X1: (minX + maxX) / 2, Y1: (minY + maxY) / 2}); err != nil {
		t.Fatalf("AddCrop: %v", err)
	}
	cropped := front.CurveCount()
	views.Recompute()
	if front.CurveCount() != cropped {
		t.Errorf("after Recompute cropped count = %d, want it held at %d (crop must survive recompute)", front.CurveCount(), cropped)
	}
	if front.CurveCount() >= full {
		t.Errorf("recompute lost the crop: count %d back at full %d", front.CurveCount(), full)
	}
}

// TestCropBreakMarkRoundTrip the break-mark boundary is drawn and its type survives save/reopen
// (#1987 AC 2).
func TestCropBreakMarkRoundTrip(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	front, _ := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1, CenterX: 100, CenterY: 100})
	minX, minY, maxX, maxY, _ := front.BoundsMM()
	if _, err := views.AddCrop(CropSpec{
		View: "FRONT", X0: minX - 1, Y0: minY - 1, X1: (minX + maxX) / 2, Y1: (minY + maxY) / 2,
		BreakMark: types.ZigzagCropBreakMark,
	}); err != nil {
		t.Fatalf("AddCrop: %v", err)
	}
	if countBreakCurves(front) == 0 {
		t.Error("zigzag crop drew no break-mark curves")
	}
	restored, ok := reopen(t, c).Sheets().Active().Views().ByName("FRONT")
	if !ok {
		t.Fatal("reopened drawing lost view FRONT")
	}
	if restored.CropCount() != 1 {
		t.Fatalf("restored crop count = %d, want 1", restored.CropCount())
	}
	if countBreakCurves(restored) == 0 {
		t.Error("restored crop lost its zigzag break mark (the type did not round-trip)")
	}
}

// TestCropCircleDropsOutside a circular fence clips a base view too (#1987).
func TestCropCircleDropsOutside(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	front, _ := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1, CenterX: 100, CenterY: 100})
	full := front.CurveCount()
	if _, err := views.AddCrop(CropSpec{View: "FRONT", Circle: true, CircleX: 100, CircleY: 100, Radius: 5}); err != nil {
		t.Fatalf("AddCrop circle: %v", err)
	}
	if front.CurveCount() >= full {
		t.Errorf("circular crop kept %d curves, want fewer than %d", front.CurveCount(), full)
	}
	// A degenerate circle (no radius) is rejected.
	if _, err := views.AddCrop(CropSpec{View: "FRONT", Circle: true, CircleX: 100, CircleY: 100, Radius: 0}); err == nil {
		t.Error("zero-radius circular crop accepted, want an error")
	}
}
