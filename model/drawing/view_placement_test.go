// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	gmath "oblikovati.org/math"
)

// TestRotatePoint2 pins the rotation math: 90° CCW maps +X to +Y and +Y to −X, about the origin.
func TestRotatePoint2(t *testing.T) {
	r := rotatePoint2(gmath.P2(1, 0), stdmath.Pi/2)
	if stdmath.Abs(float64(r.X)) > 1e-9 || stdmath.Abs(float64(r.Y)-1) > 1e-9 {
		t.Errorf("rotate (1,0) by 90° = (%v,%v), want (0,1)", r.X, r.Y)
	}
}

// TestViewRotationSwapsBounds: rotating a non-square view by 90° swaps its sheet bounding box width
// and height (the curves turned about the view centre), and the rotation survives a recompute.
func TestViewRotationSwapsBounds(t *testing.T) {
	c := drawingWithBox(t) // 2×3×4 box
	views := c.Sheets().Active().Views()
	v := frontBaseView(t, views)
	w0, h0 := boundsWH(t, v)

	if err := views.Rotate("FRONT", 90); err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	w1, h1 := boundsWH(t, v)
	if stdmath.Abs(w1-h0) > 1e-6 || stdmath.Abs(h1-w0) > 1e-6 {
		t.Errorf("after 90° rotation bounds = %.3f×%.3f, want swapped %.3f×%.3f", w1, h1, h0, w0)
	}
	if stdmath.Abs(v.RotationDeg()-90) > 1e-9 {
		t.Errorf("rotation = %.3f°, want 90 (survives recompute)", v.RotationDeg())
	}
	views.Recompute()
	if w, h := boundsWH(t, v); stdmath.Abs(w-w1) > 1e-6 || stdmath.Abs(h-h1) > 1e-6 {
		t.Errorf("after recompute bounds = %.3f×%.3f, want unchanged %.3f×%.3f", w, h, w1, h1)
	}
}

// TestHorizontalAlignmentTracksAnchor: horizontally aligning B to A holds B's Y to A's Y — including
// when A moves — and in-position breaks the lock (#1988).
func TestHorizontalAlignmentTracksAnchor(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	a, err := views.AddBase(BaseViewSpec{Name: "A", Orientation: types.BaseViewFront, Scale: 1, CenterX: 100, CenterY: 100})
	if err != nil {
		t.Fatalf("AddBase A: %v", err)
	}
	b, err := views.AddBase(BaseViewSpec{Name: "B", Orientation: types.BaseViewRight, Scale: 1, CenterX: 200, CenterY: 150})
	if err != nil {
		t.Fatalf("AddBase B: %v", err)
	}

	if err := views.Align("B", "A", types.HorizontalViewAlignment, nil); err != nil {
		t.Fatalf("Align: %v", err)
	}
	if b.centerY != a.centerY {
		t.Errorf("after align, B.centerY = %v, want A.centerY %v", b.centerY, a.centerY)
	}

	// Move the anchor: B follows on Y, keeps its own X.
	if err := views.EditBase("A", types.BaseViewFront, a.Style(), 1, 100, 250); err != nil {
		t.Fatalf("EditBase A: %v", err)
	}
	if b.centerY != 250 {
		t.Errorf("after A moved to Y=250, B.centerY = %v, want 250 (tracked)", b.centerY)
	}
	if b.centerX != 200 {
		t.Errorf("B.centerX = %v, want 200 (horizontal alignment leaves X free)", b.centerX)
	}

	// Break the lock: B no longer follows.
	if err := views.Align("B", "", types.InPositionViewAlignment, nil); err != nil {
		t.Fatalf("Align inPosition: %v", err)
	}
	if err := views.EditBase("A", types.BaseViewFront, a.Style(), 1, 100, 400); err != nil {
		t.Fatalf("EditBase A: %v", err)
	}
	if b.centerY == 400 {
		t.Errorf("after breaking the lock, B.centerY = %v, should not track A (400)", b.centerY)
	}
}

// TestAlignRejectsSelfAndMissingAnchor: aligning a view to itself or to a missing anchor errors.
func TestAlignRejectsSelfAndMissingAnchor(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	if _, err := views.AddBase(BaseViewSpec{Name: "A", Orientation: types.BaseViewFront, Scale: 1, CenterX: 100, CenterY: 100}); err != nil {
		t.Fatalf("AddBase A: %v", err)
	}
	if err := views.Align("A", "A", types.HorizontalViewAlignment, nil); err == nil {
		t.Error("aligning a view to itself should error")
	}
	if err := views.Align("A", "Ghost", types.VerticalViewAlignment, nil); err == nil {
		t.Error("aligning to a missing anchor should error")
	}
}

// frontBaseView adds a FRONT base view named "FRONT" and returns it.
func frontBaseView(t *testing.T, views *DrawingViews) *DrawingView {
	t.Helper()
	v, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1, CenterX: 100, CenterY: 100})
	if err != nil {
		t.Fatalf("AddBase FRONT: %v", err)
	}
	return v
}

// boundsWH returns a view's sheet bounding-box width and height.
func boundsWH(t *testing.T, v *DrawingView) (w, h float64) {
	t.Helper()
	minX, minY, maxX, maxY, ok := v.BoundsMM()
	if !ok {
		t.Fatal("view has no bounds")
	}
	return maxX - minX, maxY - minY
}
