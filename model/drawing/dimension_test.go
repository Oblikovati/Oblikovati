// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"math"
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/subd"
	gmath "oblikovati.org/math"
)

// drawingWithCylinder returns a box-named drawing backed by a solid cylinder (radius cm, height
// 5 cm along +Z) — the fixture for radius/diameter dimension tests (a box has no circular edges).
func drawingWithCylinder(t *testing.T, radius float64) *Content {
	t.Helper()
	cyl, err := brep.SolidCylinder(gmath.P3(0, 0, 0), gmath.V3(0, 0, 1), radius, 5)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	c := NewContent()
	c.SetBodyResolver(fakeBodyResolver{body: cyl})
	c.SetModelReference("box.opd")
	return c
}

// topBase adds a TOP base view (so a cylinder's rim circles project as circles) at scale 1.
func topBase(t *testing.T, views *DrawingViews) {
	t.Helper()
	if _, err := views.AddBase(BaseViewSpec{Name: "TOP", Orientation: types.BaseViewTop, Scale: 1, CenterX: 100, CenterY: 100}); err != nil {
		t.Fatalf("AddBase TOP: %v", err)
	}
}

// TestRadialDimensionMeasuresCircle: a radius dimension on a 2 cm cylinder reports R20 and a
// diameter dimension Ø40 — the true model size, with leader glyph curves.
func TestRadialDimensionMeasuresCircle(t *testing.T) {
	c := drawingWithCylinder(t, 2)
	views := c.Sheets().Active().Views()
	topBase(t, views)
	dims := c.Sheets().Active().Dimensions()

	rd, err := dims.AddRadial("R1", "TOP", types.RadiusDimension, 100, 100)
	if err != nil {
		t.Fatalf("AddRadial radius: %v", err)
	}
	if math.Abs(rd.ValueMM()-20) > 1e-6 || !strings.HasPrefix(rd.Text(), "R") || rd.CurveCount() == 0 {
		t.Errorf("radius dim = (%v mm, %q, %d curves), want 20 / R… / glyph", rd.ValueMM(), rd.Text(), rd.CurveCount())
	}
	dd, err := dims.AddRadial("D1", "TOP", types.DiameterDimension, 100, 100)
	if err != nil {
		t.Fatalf("AddRadial diameter: %v", err)
	}
	if math.Abs(dd.ValueMM()-40) > 1e-6 || !strings.HasPrefix(dd.Text(), "Ø") {
		t.Errorf("diameter dim = (%v mm, %q), want 40 / Ø…", dd.ValueMM(), dd.Text())
	}
}

// TestArcLengthDimensionMeasuresCircumference: an arc-length dimension on a 2 cm cylinder's rim
// reports its circumference (2π·20 ≈ 125.66 mm) with a glyph that follows the arc, and re-measures
// when the model changes.
func TestArcLengthDimensionMeasuresCircumference(t *testing.T) {
	c := drawingWithCylinder(t, 2)
	views := c.Sheets().Active().Views()
	topBase(t, views)
	dims := c.Sheets().Active().Dimensions()

	ad, err := dims.AddArcLength("L1", "TOP", 100, 100)
	if err != nil {
		t.Fatalf("AddArcLength: %v", err)
	}
	want := 2 * math.Pi * 20 // circumference of a 20 mm-radius rim
	if d := ad.Type(); d != types.ArcLengthDimension {
		t.Errorf("type = %v, want ArcLengthDimension", d)
	}
	if math.Abs(ad.ValueMM()-want) > 1e-3 || ad.CurveCount() == 0 {
		t.Errorf("arc-length = (%v mm, %d curves), want %v with a glyph", ad.ValueMM(), ad.CurveCount(), want)
	}

	wider, err := brep.SolidCylinder(gmath.P3(0, 0, 0), gmath.V3(0, 0, 1), 3, 5)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	c.SetBodyResolver(fakeBodyResolver{body: wider})
	c.RecomputeViews()
	if want := 2 * math.Pi * 30; math.Abs(ad.ValueMM()-want) > 1e-3 {
		t.Errorf("after the cylinder widened, arc-length = %v mm, want %v", ad.ValueMM(), want)
	}
}

// TestArcLengthNeedsCircularEdge: a box has no circular edge, so an arc-length dimension errors.
func TestArcLengthNeedsCircularEdge(t *testing.T) {
	c := drawingWithBox(t)
	frontBase(t, c.Sheets().Active().Views())
	if _, err := c.Sheets().Active().Dimensions().AddArcLength("L1", "FRONT", 100, 100); err == nil {
		t.Error("AddArcLength on a box (no circular edges) = ok, want error")
	}
}

// TestRadialDimensionIsAssociative: a radius dimension follows the model — widening the cylinder
// 2→3 cm re-measures 20→30 mm.
func TestRadialDimensionIsAssociative(t *testing.T) {
	c := drawingWithCylinder(t, 2)
	views := c.Sheets().Active().Views()
	topBase(t, views)
	dims := c.Sheets().Active().Dimensions()
	rd, err := dims.AddRadial("R1", "TOP", types.RadiusDimension, 100, 100)
	if err != nil {
		t.Fatalf("AddRadial: %v", err)
	}
	if math.Abs(rd.ValueMM()-20) > 1e-6 {
		t.Fatalf("initial radius = %v mm, want 20", rd.ValueMM())
	}
	wider, err := brep.SolidCylinder(gmath.P3(0, 0, 0), gmath.V3(0, 0, 1), 3, 5)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	c.SetBodyResolver(fakeBodyResolver{body: wider})
	c.RecomputeViews()
	if math.Abs(rd.ValueMM()-30) > 1e-6 {
		t.Errorf("after the cylinder widened, radius = %v mm, want 30", rd.ValueMM())
	}
}

// TestRadialForEachCircleDedupsRims: auto-dimensioning a cylinder's circular edges dimensions the
// through-feature once (its two coincident rims), not twice.
func TestRadialForEachCircleDedupsRims(t *testing.T) {
	c := drawingWithCylinder(t, 2)
	views := c.Sheets().Active().Views()
	topBase(t, views)
	n, err := c.Sheets().Active().Dimensions().AddRadialForEachCircle("TOP", types.DiameterDimension)
	if err != nil {
		t.Fatalf("AddRadialForEachCircle: %v", err)
	}
	if n != 1 {
		t.Errorf("added %d dimensions, want 1 (the two coincident rims dedup)", n)
	}
}

// TestBaselineAndChainSets: a baseline set measures from one corner to each of the others
// (stacked), a chain set measures between consecutive corners, and both are associative linear
// dimensions.
func TestBaselineAndChainSets(t *testing.T) {
	c := drawingWithBox(t) // box 2×3×4 cm; FRONT corners at x∈{90,110}, y∈{80,120}
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	corners := [][2]float64{{90, 80}, {110, 80}, {110, 120}, {90, 120}}

	base, err := c.Sheets().Active().Dimensions().AddBaselineSet("FRONT", types.AlignedDimension, corners)
	if err != nil {
		t.Fatalf("AddBaselineSet: %v", err)
	}
	if len(base) != 3 {
		t.Fatalf("baseline set = %d dims, want 3 (one per non-datum corner)", len(base))
	}
	if math.Abs(base[0].ValueMM()-20) > 1e-6 { // datum→bottom-right = 2 cm width
		t.Errorf("baseline[0] = %v mm, want 20", base[0].ValueMM())
	}
	if base[0].CurveCount() == 0 {
		t.Error("baseline dimension has no glyph")
	}

	chain, err := c.Sheets().Active().Dimensions().AddChainSet("FRONT", types.AlignedDimension, corners)
	if err != nil {
		t.Fatalf("AddChainSet: %v", err)
	}
	if len(chain) != 3 {
		t.Fatalf("chain set = %d dims, want 3 (between consecutive corners)", len(chain))
	}
	if math.Abs(chain[0].ValueMM()-20) > 1e-6 { // bottom edge = 2 cm
		t.Errorf("chain[0] = %v mm, want 20", chain[0].ValueMM())
	}
}

// TestOrdinateSetMeasuresOffsetsFromDatum: an ordinate set measures each point's view-X (or
// view-Y) offset from the datum corner, draws a leaderless witness line, and survives reopen with
// its axis. Box 2×3×4 cm; FRONT corners x∈{90,110} (20 mm wide), y∈{80,120} (40 mm tall).
func TestOrdinateSetMeasuresOffsetsFromDatum(t *testing.T) {
	c := drawingWithBox(t)
	frontBase(t, c.Sheets().Active().Views())
	dims := c.Sheets().Active().Dimensions()
	datum := [2]float64{90, 80} // bottom-left corner
	pts := [][2]float64{{90, 80}, {110, 80}, {110, 120}}

	horiz, err := dims.AddOrdinateSet("FRONT", true, datum, pts)
	if err != nil {
		t.Fatalf("AddOrdinateSet horizontal: %v", err)
	}
	if len(horiz) != 3 {
		t.Fatalf("ordinate set = %d dims, want 3 (one per point)", len(horiz))
	}
	// X-offsets from the datum: the datum itself = 0, the bottom-right = 20 mm wide.
	if math.Abs(horiz[0].ValueMM()-0) > 1e-6 || math.Abs(horiz[1].ValueMM()-20) > 1e-6 {
		t.Errorf("horizontal ordinate values = %v / %v mm, want 0 / 20", horiz[0].ValueMM(), horiz[1].ValueMM())
	}
	if horiz[1].Type() != types.OrdinateDimension || horiz[1].CurveCount() == 0 {
		t.Errorf("ordinate = (%v, %d curves), want OrdinateDimension with a witness line", horiz[1].Type(), horiz[1].CurveCount())
	}

	vert, err := dims.AddOrdinateSet("FRONT", false, datum, [][2]float64{{90, 120}})
	if err != nil {
		t.Fatalf("AddOrdinateSet vertical: %v", err)
	}
	if math.Abs(vert[0].ValueMM()-40) > 1e-6 { // Y-offset datum→top-left = 4 cm tall
		t.Errorf("vertical ordinate value = %v mm, want 40", vert[0].ValueMM())
	}

	restored := reopen(t, c)
	rd := restored.Sheets().Active().Dimensions()
	if rd.Count() != 4 {
		t.Fatalf("reopened ordinate count = %d, want 4", rd.Count())
	}
	// The vertical ordinate's axis must round-trip: its value stays the view-Y offset (40 mm).
	if got := rd.Item(3).ValueMM(); math.Abs(got-40) > 1e-6 {
		t.Errorf("reopened vertical ordinate = %v mm, want 40 (axis persisted)", got)
	}
}

// TestOrdinateSetNeedsAPoint: an ordinate set with no points errors.
func TestOrdinateSetNeedsAPoint(t *testing.T) {
	c := drawingWithBox(t)
	frontBase(t, c.Sheets().Active().Views())
	if _, err := c.Sheets().Active().Dimensions().AddOrdinateSet("FRONT", true, [2]float64{90, 80}, nil); err == nil {
		t.Error("AddOrdinateSet with no points should error")
	}
}

// TestDimensionSetNeedsTwoPoints: a set with fewer than two points errors.
func TestDimensionSetNeedsTwoPoints(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	if _, err := c.Sheets().Active().Dimensions().AddBaselineSet("FRONT", types.AlignedDimension, [][2]float64{{90, 80}}); err == nil {
		t.Error("a one-point baseline set should error")
	}
}

// TestSetForViewCornersBaseline: the single-action corner set dimensions a view's four corners.
func TestSetForViewCornersBaseline(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	dims, err := c.Sheets().Active().Dimensions().AddSetForViewCorners("FRONT", types.AlignedDimension, true)
	if err != nil {
		t.Fatalf("AddSetForViewCorners: %v", err)
	}
	if len(dims) != 3 {
		t.Errorf("corner baseline set = %d dims, want 3", len(dims))
	}
}

// TestDimensionTextLiftedAndDraggable: the value text sits off the dimension line by default
// (readable), and dragging the text nudges its anchor.
func TestDimensionTextLiftedAndDraggable(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	dims := c.Sheets().Active().Dimensions()
	d, err := dims.AddLinear("D1", "FRONT", types.HorizontalDimension, 88, 80, 112, 80, -12)
	if err != nil {
		t.Fatalf("AddLinear: %v", err)
	}
	// The text anchor is lifted off the dimension line (curve[2] is the horizontal dimension line).
	line := d.Curves()[2]
	tx, ty := d.TextAnchorMM()
	if math.Abs(ty-float64(line.Start().Y)) < 1e-6 {
		t.Errorf("text anchor y %v sits on the dimension line y %v — should be lifted off", ty, float64(line.Start().Y))
	}

	dims.MoveText("D1", 7, 4)
	mx, my := d.TextAnchorMM()
	if math.Abs(mx-tx-7) > 1e-9 || math.Abs(my-ty-4) > 1e-9 {
		t.Errorf("after MoveText the anchor moved by (%v,%v), want (7,4)", mx-tx, my-ty)
	}
}

// TestMoveLineShiftsDimensionLine: dragging the dimension line moves it perpendicular to itself.
func TestMoveLineShiftsDimensionLine(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	dims := c.Sheets().Active().Dimensions()
	d, err := dims.AddLinear("D1", "FRONT", types.HorizontalDimension, 88, 80, 112, 80, -12)
	if err != nil {
		t.Fatalf("AddLinear: %v", err)
	}
	before := float64(d.Curves()[2].Start().Y)
	dims.MoveLine("D1", 0, -6) // drag the horizontal dimension line down 6 mm
	after := float64(d.Curves()[2].Start().Y)
	if math.Abs(after-before+6) > 1e-6 {
		t.Errorf("dimension line moved by %v mm, want -6", after-before)
	}
}

// TestAngularDimensionMeasuresCorner: an angular dimension between a horizontal and a vertical box
// edge reports 90°, with ValueMM 0 (the value is in degrees) and an arc glyph.
func TestAngularDimensionMeasuresCorner(t *testing.T) {
	c := drawingWithBox(t) // box 2×3×4 cm
	views := c.Sheets().Active().Views()
	frontBase(t, views) // FRONT, scale 1, centre (100,100): bottom edge at y=80, side edges at x∈{90,110}
	dims := c.Sheets().Active().Dimensions()

	d, err := dims.AddAngular("A1", "FRONT", 100, 80, 110, 100) // bottom (horizontal) + a side (vertical)
	if err != nil {
		t.Fatalf("AddAngular: %v", err)
	}
	if math.Abs(d.ValueDeg()-90) > 1e-6 {
		t.Errorf("angle = %v°, want 90 (perpendicular box edges)", d.ValueDeg())
	}
	if d.ValueMM() != 0 || !strings.HasSuffix(d.Text(), "°") || d.CurveCount() == 0 {
		t.Errorf("angular dim = (%v mm, %q, %d curves), want 0 mm / …° / arc glyph", d.ValueMM(), d.Text(), d.CurveCount())
	}
	if d.Type() != types.AngularDimension {
		t.Errorf("type = %v, want angular", d.Type())
	}
}

// TestAngularForFirstCorner: the single-action corner callout finds two non-parallel edges (a box
// corner) and reports 90°.
func TestAngularForFirstCorner(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	d, err := c.Sheets().Active().Dimensions().AddAngularForFirstCorner("FRONT")
	if err != nil {
		t.Fatalf("AddAngularForFirstCorner: %v", err)
	}
	if math.Abs(d.ValueDeg()-90) > 1e-6 {
		t.Errorf("corner angle = %v°, want 90", d.ValueDeg())
	}
}

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
