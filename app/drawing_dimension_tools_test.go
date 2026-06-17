// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestLinearDimensionToolPlacesOverallDimension: the tool dimensions a base view's overall size in
// the chosen direction, producing an associative dimension with a positive measured value.
func TestLinearDimensionToolPlacesOverallDimension(t *testing.T) {
	s := drawingWithModelSession(t)
	base := NewBaseViewTool()
	base.Start(s)
	base.SetPlacement(120, 100)
	if err := base.Commit(s); err != nil {
		t.Fatalf("place base view: %v", err)
	}

	tool := NewLinearDimensionTool()
	tool.Start(s)
	tool.Params().Choices[1].Set(1) // Type = Vertical
	if err := tool.Commit(s); err != nil {
		t.Fatalf("dimension Commit: %v", err)
	}

	c, _ := ActiveDrawing(s)
	dims := c.Sheets().Active().Dimensions()
	if dims.Count() != 1 {
		t.Fatalf("dimension count = %d, want 1", dims.Count())
	}
	d := dims.Item(0)
	if d.Type() != types.VerticalDimension || d.ValueMM() <= 0 || d.CurveCount() == 0 {
		t.Errorf("dimension = (%v, %v mm, %d curves), want a vertical dim with a positive value + glyph",
			d.Type(), d.ValueMM(), d.CurveCount())
	}
}

// TestLinearDimensionToolNeedsBaseView: committing with no base view errors rather than panicking.
func TestLinearDimensionToolNeedsBaseView(t *testing.T) {
	s := drawingWithModelSession(t)
	tool := NewLinearDimensionTool()
	tool.Start(s)
	if tool.CanCommit() {
		t.Error("CanCommit should be false with no base view to dimension")
	}
	if err := tool.Commit(s); err == nil {
		t.Error("Commit with no base view = ok, want error")
	}
}

// TestDimensionPlacement maps the Type index + view bounds to the two pick points and the
// dimension-line offset: horizontal across the width below, vertical down the height to the right,
// aligned along the diagonal.
func TestDimensionPlacement(t *testing.T) {
	const minX, minY, maxX, maxY = 100.0, 80.0, 140.0, 120.0
	cases := []struct {
		idx      int
		wantType types.DrawingDimensionType
	}{
		{0, types.HorizontalDimension}, {1, types.VerticalDimension}, {2, types.AlignedDimension},
	}
	for _, tc := range cases {
		typ, x1, y1, x2, y2, _ := dimensionPlacement(tc.idx, minX, minY, maxX, maxY)
		if typ != tc.wantType {
			t.Errorf("dimensionPlacement(%d) type = %v, want %v", tc.idx, typ, tc.wantType)
		}
		if x1 == x2 && y1 == y2 {
			t.Errorf("dimensionPlacement(%d) returned coincident pick points", tc.idx)
		}
	}
}
