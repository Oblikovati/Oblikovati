// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// drawingWithCylinderSession builds a 2 cm-radius cylinder part + an active drawing of it — the
// fixture for the radial dimension tool (a box has no circular edges).
func drawingWithCylinderSession(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("commands: %v", err)
	}
	part, err := compdef.AddPart(s.Workspace(), "box.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := part.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	sk.Circles().AddByCenterRadius(gmath.P2(0, 0), 2)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
	if _, err := s.NewDrawing(); err != nil {
		t.Fatalf("NewDrawing: %v", err)
	}
	c, _ := ActiveDrawing(s)
	c.SetModelReference("box.opd")
	return s
}

// TestRadialDimensionToolDimensionsHoles: the radial tool dimensions a base view's circular edges
// as diameter callouts (the auto "dimension all holes" action).
func TestRadialDimensionToolDimensionsHoles(t *testing.T) {
	s := drawingWithCylinderSession(t)
	base := NewBaseViewTool()
	base.Start(s)
	base.Params().Choices[0].Set(1) // Top, so the cylinder's rim projects as a circle
	base.SetPlacement(120, 100)
	if err := base.Commit(s); err != nil {
		t.Fatalf("place base view: %v", err)
	}

	tool := NewRadialDimensionTool()
	tool.Start(s)
	tool.Params().Choices[1].Set(1) // Type = Diameter
	if err := tool.Commit(s); err != nil {
		t.Fatalf("radial Commit: %v", err)
	}
	c, _ := ActiveDrawing(s)
	dims := c.Sheets().Active().Dimensions()
	if dims.Count() != 1 {
		t.Fatalf("dimension count = %d, want 1 (the deduped hole)", dims.Count())
	}
	d := dims.Item(0)
	if d.Type() != types.DiameterDimension || math.Abs(d.ValueMM()-40) > 1e-6 {
		t.Errorf("dimension = (%v, %v mm), want a diameter ⌀40", d.Type(), d.ValueMM())
	}
}

// TestDimensionSetToolDimensionsCorners: the set tool places a baseline set of three linear
// dimensions on a base view's corners.
func TestDimensionSetToolDimensionsCorners(t *testing.T) {
	s := drawingWithModelSession(t)
	base := NewBaseViewTool()
	base.Start(s)
	base.SetPlacement(120, 100)
	if err := base.Commit(s); err != nil {
		t.Fatalf("place base view: %v", err)
	}
	tool := NewDimensionSetTool()
	tool.Start(s)
	tool.Params().Choices[2].Set(2) // Type = Aligned
	if err := tool.Commit(s); err != nil {
		t.Fatalf("set Commit: %v", err)
	}
	c, _ := ActiveDrawing(s)
	if n := c.Sheets().Active().Dimensions().Count(); n != 3 {
		t.Errorf("dimension set placed %d dimensions, want 3", n)
	}
}

// TestOrdinateDimensionToolDimensionsCorners: the ordinate tool places a leaderless ordinate per
// corner of a base view, measured from the bottom-left datum.
func TestOrdinateDimensionToolDimensionsCorners(t *testing.T) {
	s := drawingWithModelSession(t)
	base := NewBaseViewTool()
	base.Start(s)
	base.SetPlacement(120, 100)
	if err := base.Commit(s); err != nil {
		t.Fatalf("place base view: %v", err)
	}
	tool := NewOrdinateDimensionTool()
	tool.Start(s)
	tool.Params().Choices[1].Set(1) // Axis = Vertical
	if err := tool.Commit(s); err != nil {
		t.Fatalf("ordinate Commit: %v", err)
	}
	c, _ := ActiveDrawing(s)
	dims := c.Sheets().Active().Dimensions()
	if dims.Count() != 4 {
		t.Fatalf("ordinate set placed %d dimensions, want 4 (one per corner)", dims.Count())
	}
	if dims.Item(0).Type() != types.OrdinateDimension {
		t.Errorf("dimension type = %v, want OrdinateDimension", dims.Item(0).Type())
	}
}

// TestAngularDimensionToolDimensionsCorner: the angular tool dimensions a base view's corner angle
// (a box's perpendicular edges → 90°).
func TestAngularDimensionToolDimensionsCorner(t *testing.T) {
	s := drawingWithModelSession(t)
	base := NewBaseViewTool()
	base.Start(s)
	base.SetPlacement(120, 100)
	if err := base.Commit(s); err != nil {
		t.Fatalf("place base view: %v", err)
	}

	tool := NewAngularDimensionTool()
	tool.Start(s)
	if err := tool.Commit(s); err != nil {
		t.Fatalf("angular Commit: %v", err)
	}
	c, _ := ActiveDrawing(s)
	dims := c.Sheets().Active().Dimensions()
	if dims.Count() != 1 {
		t.Fatalf("dimension count = %d, want 1", dims.Count())
	}
	if d := dims.Item(0); d.Type() != types.AngularDimension || math.Abs(d.ValueDeg()-90) > 1e-6 {
		t.Errorf("dimension = (%v, %v°), want an angular 90°", d.Type(), d.ValueDeg())
	}
}

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
