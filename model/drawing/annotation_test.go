// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
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
