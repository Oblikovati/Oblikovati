// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/subd"
)

// TestLinearDimensionMeasuresTrueModelSize: a horizontal dimension across the front view's bottom
// edge reports the box's true X-width (2 cm → 20 mm), independent of the view scale, and produces
// glyph curves (extension lines, dimension line, arrowheads).
func TestLinearDimensionMeasuresTrueModelSize(t *testing.T) {
	c := drawingWithBox(t) // box 2×3×4 cm
	views := c.Sheets().Active().Views()
	frontBase(t, views) // FRONT, scale 1, centre (100,100) → front corners at x∈{90,110}, y∈{80,120}

	dims := c.Sheets().Active().Dimensions()
	d, err := dims.AddLinear("D1", "FRONT", types.HorizontalDimension, 88, 80, 112, 80, -12)
	if err != nil {
		t.Fatalf("AddLinear: %v", err)
	}
	if math.Abs(d.ValueMM()-20) > 1e-6 {
		t.Errorf("value = %v mm, want 20 (the box's 2 cm X-width)", d.ValueMM())
	}
	if d.Text() != "20" {
		t.Errorf("text = %q, want %q", d.Text(), "20")
	}
	if d.CurveCount() == 0 {
		t.Error("dimension produced no glyph curves")
	}
	if d.Type() != types.HorizontalDimension || d.ViewName() != "FRONT" {
		t.Errorf("dimension = (%v on %q), want a horizontal dim on FRONT", d.Type(), d.ViewName())
	}
}

// TestLinearDimensionIsAssociative is the PBI-141 acceptance criterion: a dimension updates when
// the model size changes. The dimension attaches to vertices by reference key, so re-resolving
// against a wider box (same topology) re-measures it.
func TestLinearDimensionIsAssociative(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	dims := c.Sheets().Active().Dimensions()
	d, err := dims.AddLinear("D1", "FRONT", types.HorizontalDimension, 88, 80, 112, 80, -12)
	if err != nil {
		t.Fatalf("AddLinear: %v", err)
	}
	if math.Abs(d.ValueMM()-20) > 1e-6 {
		t.Fatalf("initial value = %v mm, want 20", d.ValueMM())
	}

	// The model grows to a 6 cm X-width (same topology); the dimension must follow.
	c.SetBodyResolver(fakeBodyResolver{body: subd.ToBody(subd.Box(6, 3, 4), "box")})
	c.RecomputeViews()
	if math.Abs(d.ValueMM()-60) > 1e-6 {
		t.Errorf("after the model widened, value = %v mm, want 60", d.ValueMM())
	}
}

// TestLinearDimensionAlignedDistance: an aligned dimension between two diagonally opposite front
// corners reports the true planar diagonal (√(2²+4²) cm = √20 cm → ~44.72 mm).
func TestLinearDimensionAlignedDistance(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	dims := c.Sheets().Active().Dimensions()
	d, err := dims.AddLinear("DIAG", "FRONT", types.AlignedDimension, 90, 80, 110, 120, 0)
	if err != nil {
		t.Fatalf("AddLinear: %v", err)
	}
	want := math.Hypot(20, 40) // 2 cm × 4 cm front face diagonal, in mm
	if math.Abs(d.ValueMM()-want) > 1e-6 {
		t.Errorf("aligned value = %v mm, want %v", d.ValueMM(), want)
	}
}

// TestDimensionRejectsNonBaseAndSurvivesReopen: a dimension can only attach to a base view, and a
// persisted dimension re-binds its vertices and re-measures on reopen.
func TestDimensionRejectsNonBaseAndSurvivesReopen(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	if _, err := views.AddProjected(ProjectedViewSpec{Name: "RIGHT", BaseView: "FRONT", Direction: types.ProjectRight, CenterX: 240, CenterY: 100}); err != nil {
		t.Fatalf("AddProjected: %v", err)
	}
	dims := c.Sheets().Active().Dimensions()
	if _, err := dims.AddLinear("BAD", "RIGHT", types.HorizontalDimension, 0, 0, 10, 0, 0); err == nil {
		t.Error("AddLinear on a projected view should be rejected (base views only in this increment)")
	}
	if _, err := dims.AddLinear("D1", "FRONT", types.HorizontalDimension, 88, 80, 112, 80, -12); err != nil {
		t.Fatalf("AddLinear FRONT: %v", err)
	}

	restored := reopen(t, c)
	rd := restored.Sheets().Active().Dimensions()
	if rd.Count() != 1 {
		t.Fatalf("reopened dimension count = %d, want 1", rd.Count())
	}
	if got := rd.Item(0).ValueMM(); math.Abs(got-20) > 1e-6 {
		t.Errorf("reopened dimension value = %v mm, want 20 (vertices re-bound)", got)
	}
}
