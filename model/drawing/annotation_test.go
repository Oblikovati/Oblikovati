// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/brep"
	gmath "oblikovati.org/math"
)

// TestAddCoGMarkerOnView checks a centre-of-gravity marker is placed on a view (with glyph
// curves) and lands within the view's bounds (the model centroid projects inside its silhouette).
func TestAddCoGMarkerOnView(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	front, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1, CenterX: 100, CenterY: 100})
	if err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	an, err := c.Sheets().Active().Annotations().AddCoGMarker("CG", "FRONT")
	if err != nil {
		t.Fatalf("AddCoGMarker: %v", err)
	}
	if an.Kind() != types.CoGMarkerAnnotation || an.ViewName() != "FRONT" {
		t.Fatalf("annotation = kind %v view %q, want a CoG marker on FRONT", an.Kind(), an.ViewName())
	}
	if an.CurveCount() == 0 {
		t.Fatal("CoG marker produced no glyph curves")
	}
	// The marker centre (a box's centroid projects to the view centre) should sit within bounds.
	minX, minY, maxX, maxY, _ := front.BoundsMM()
	var sx, sy float64
	for _, cv := range an.Curves() {
		sx, sy = float64(cv.Start().X), float64(cv.Start().Y)
		if sx < minX-5 || sx > maxX+5 || sy < minY-5 || sy > maxY+5 {
			t.Errorf("CoG glyph point (%g,%g) outside the view bounds", sx, sy)
		}
	}
}

// TestAddCenterMarksOnCircularEdges: centre-marking a cylinder's TOP view places one crosshair at
// the rim's centre (the two coincident rims dedup), survives reopen, and re-projects on recompute.
func TestAddCenterMarksOnCircularEdges(t *testing.T) {
	c := drawingWithCylinder(t, 2)
	topBase(t, c.Sheets().Active().Views())
	marks, err := c.Sheets().Active().Annotations().AddCenterMarks("TOP")
	if err != nil {
		t.Fatalf("AddCenterMarks: %v", err)
	}
	if len(marks) != 1 {
		t.Fatalf("centre marks = %d, want 1 (the two coincident rims dedup)", len(marks))
	}
	if marks[0].Kind() != types.CenterMarkAnnotation || marks[0].CurveCount() == 0 {
		t.Errorf("mark = (%v, %d curves), want a centre mark with a crosshair glyph", marks[0].Kind(), marks[0].CurveCount())
	}

	// Reopen against the same cylinder so the persisted edge key re-binds and the glyph re-derives.
	data, err := c.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	cyl, err := brep.SolidCylinder(gmath.P3(0, 0, 0), gmath.V3(0, 0, 1), 2, 5)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	restored := NewContent()
	restored.SetBodyResolver(fakeBodyResolver{body: cyl})
	if err := restored.ApplyRecipe(data); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	restored.RecomputeViews()
	ra := restored.Sheets().Active().Annotations()
	if ra.Count() != 1 || ra.Item(0).Kind() != types.CenterMarkAnnotation {
		t.Fatalf("reopened annotations = %d, want 1 centre mark", ra.Count())
	}
	if ra.Item(0).CurveCount() == 0 {
		t.Error("reopened centre mark did not re-derive its glyph (edge key not persisted?)")
	}
}

// TestAddCenterlinesOnView: centerlines on a view produce a dash-dot horizontal+vertical cross
// (many segments) spanning its bounds, survive reopen, and re-derive on recompute.
func TestAddCenterlinesOnView(t *testing.T) {
	c := drawingWithBox(t)
	front, err := c.Sheets().Active().Views().AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 2, CenterX: 100, CenterY: 100})
	if err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	cl, err := c.Sheets().Active().Annotations().AddCenterlines("CL", "FRONT")
	if err != nil {
		t.Fatalf("AddCenterlines: %v", err)
	}
	if cl.Kind() != types.CenterlineAnnotation || cl.CurveCount() < 4 {
		t.Fatalf("centerlines = (%v, %d curves), want a dash-dot cross (many segments)", cl.Kind(), cl.CurveCount())
	}
	// The dash-dot segments should straddle the view bounds (the cross spans the whole view).
	minX, _, maxX, _, _ := front.BoundsMM()
	lo, hi := 1e9, -1e9
	for _, cv := range cl.Curves() {
		lo = min(lo, float64(cv.Start().X))
		hi = max(hi, float64(cv.End().X))
	}
	if lo > minX || hi < maxX {
		t.Errorf("centerline horizontal span [%g,%g] does not cover the view [%g,%g]", lo, hi, minX, maxX)
	}

	rann := reopen(t, c).Sheets().Active().Annotations()
	rcl, ok := rann.ByName("CL")
	if !ok || rcl.Kind() != types.CenterlineAnnotation || rcl.CurveCount() < 4 {
		t.Errorf("reopened centerlines wrong: ok=%v curves=%d", ok, rcl.CurveCount())
	}
}

// TestCenterlinesNeedView: centerlines need an existing view with geometry.
func TestCenterlinesNeedView(t *testing.T) {
	c := drawingWithBox(t)
	if _, err := c.Sheets().Active().Annotations().AddCenterlines("CL", "NOPE"); err == nil {
		t.Error("AddCenterlines on a missing view = ok, want error")
	}
}

// TestCenterMarksNeedCircularEdge: a box has no circular edges, so centre-marking errors.
func TestCenterMarksNeedCircularEdge(t *testing.T) {
	c := drawingWithBox(t)
	frontBase(t, c.Sheets().Active().Views())
	if _, err := c.Sheets().Active().Annotations().AddCenterMarks("FRONT"); err == nil {
		t.Error("AddCenterMarks on a box (no circular edges) = ok, want error")
	}
}

// TestCoGMarkerRejectsMissingView checks a CoG marker needs an existing view.
func TestCoGMarkerRejectsMissingView(t *testing.T) {
	c := drawingWithBox(t)
	if _, err := c.Sheets().Active().Annotations().AddCoGMarker("CG", "NOPE"); err == nil {
		t.Error("CoG marker on a missing view = ok, want error")
	}
}

// TestAddRevisionCloud checks a revision cloud produces a scalloped boundary and carries its tag.
func TestAddRevisionCloud(t *testing.T) {
	c := drawingWithBox(t)
	an, err := c.Sheets().Active().Annotations().AddRevisionCloud("REV-A", 50, 50, 60, 40, "A")
	if err != nil {
		t.Fatalf("AddRevisionCloud: %v", err)
	}
	if an.Kind() != types.RevisionCloudAnnotation || an.Tag() != "A" {
		t.Fatalf("annotation = kind %v tag %q, want a revision cloud tagged A", an.Kind(), an.Tag())
	}
	if an.CurveCount() < 8 {
		t.Errorf("revision cloud = %d curves, want a scalloped boundary (many segments)", an.CurveCount())
	}
	if _, err := c.Sheets().Active().Annotations().AddRevisionCloud("BAD", 0, 0, 0, 0, ""); err == nil {
		t.Error("zero-size revision cloud = ok, want error")
	}
}

// TestCoGMarkerTracksModelRecompute checks the marker re-projects when RecomputeViews runs.
func TestCoGMarkerTracksModelRecompute(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	if _, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1, CenterX: 100, CenterY: 100}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	an, _ := c.Sheets().Active().Annotations().AddCoGMarker("CG", "FRONT")
	before := an.CurveCount()
	c.RecomputeViews()
	if an.CurveCount() != before || before == 0 {
		t.Errorf("CoG marker curves after recompute = %d, want a stable non-empty glyph (%d)", an.CurveCount(), before)
	}
	if err := c.Sheets().Active().Annotations().Remove("CG"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if c.Sheets().Active().Annotations().Count() != 0 {
		t.Error("annotation not removed")
	}
}

// TestAnnotationsRecipeRoundTrip checks CoG markers and revision clouds survive persistence.
func TestAnnotationsRecipeRoundTrip(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	if _, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1, CenterX: 100, CenterY: 100}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	an := c.Sheets().Active().Annotations()
	if _, err := an.AddCoGMarker("CG", "FRONT"); err != nil {
		t.Fatalf("AddCoGMarker: %v", err)
	}
	if _, err := an.AddRevisionCloud("REV", 40, 40, 50, 30, "B"); err != nil {
		t.Fatalf("AddRevisionCloud: %v", err)
	}
	rann := reopen(t, c).Sheets().Active().Annotations()
	if rann.Count() != 2 {
		t.Fatalf("restored annotations = %d, want 2", rann.Count())
	}
	cg, ok := rann.ByName("CG")
	if !ok || cg.Kind() != types.CoGMarkerAnnotation || cg.CurveCount() == 0 {
		t.Errorf("restored CG marker wrong: ok=%v curves=%d", ok, cg.CurveCount())
	}
	rev, ok := rann.ByName("REV")
	if !ok || rev.Tag() != "B" || rev.CurveCount() == 0 {
		t.Errorf("restored revision cloud wrong: ok=%v tag=%q curves=%d", ok, rev.Tag(), rev.CurveCount())
	}
}
