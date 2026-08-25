//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// projectedSketch returns an XY sketch on a fresh part with the origin centre point and the YZ
// origin plane projected into it (a reference point and a reference line, #1262).
func projectedSketch(t *testing.T) *sketch.Sketch {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "p.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	sk.ProjectPoint(compdef.NewWorkPointRefSource(def, feature.OriginCenter))
	sk.ProjectCurve(compdef.NewWorkPlaneRefSource(def, feature.OriginYZPlane, sketch.XYPlane()))
	return sk
}

// TestProjectedCurveOverlayDrawsReferenceLines: a projected curve becomes a drawn polyline.
func TestProjectedCurveOverlayDrawsReferenceLines(t *testing.T) {
	if got := projectedCurveOverlay(projectedSketch(t), nil, nil); len(got) != 1 {
		t.Fatalf("projectedCurveOverlay = %d items, want 1 (the projected reference line)", len(got))
	}
	if got := projectedCurveOverlay(nil, nil, nil); got != nil {
		t.Errorf("projectedCurveOverlay(nil, nil, nil) = %v, want nil", got)
	}
}

// TestProjectedCurveOverlayEmptyWhenNoProjection: a plain sketch projects no curves.
func TestProjectedCurveOverlayEmptyWhenNoProjection(t *testing.T) {
	s := app.NewSession()
	pd, _ := compdef.AddPart(s.Workspace(), "q.opd", true)
	sk := pd.Content().(*compdef.PartComponentDefinition).Sketches().Add(sketch.XYPlane())
	if got := projectedCurveOverlay(sk, nil, nil); got != nil {
		t.Errorf("projectedCurveOverlay of a curve-less sketch = %v, want nil", got)
	}
}

// TestPointsOverlayIncludesProjectedPoint: the projected origin anchor is glyphed alongside
// regular sketch points (so the auto-projected centre point is visible, #1262).
func TestPointsOverlayIncludesProjectedPoint(t *testing.T) {
	sk := projectedSketch(t)
	item, ok := pointsOverlay(sk.Plane(), sk, 0.1)
	if !ok || len(item.Positions) == 0 {
		t.Fatal("pointsOverlay should glyph the projected origin point")
	}
}

// firstProjectedCurve returns the sketch's first projected reference curve.
func firstProjectedCurve(sk *sketch.Sketch) *sketch.ProjectedCurve {
	for _, e := range sk.Entities() {
		if pc, ok := e.(*sketch.ProjectedCurve); ok {
			return pc
		}
	}
	return nil
}

// drawListHasColor reports whether any item is drawn in color c.
func drawListHasColor(items []renderer.DrawItem, c [4]float32) bool {
	for _, it := range items {
		if it.Color == c {
			return true
		}
	}
	return false
}

// TestProjectedCurveOverlayTintsHoverAndSelection is the #2158 follow-up: a projected curve must
// draw in the candidate colour when it is the hover candidate and the selected colour when selected,
// so the offset workflow sees what it is picking — before, projected curves always drew plain, so
// selection worked but gave no feedback.
func TestProjectedCurveOverlayTintsHoverAndSelection(t *testing.T) {
	sk := projectedSketch(t)
	pc := firstProjectedCurve(sk)
	if pc == nil {
		t.Fatal("setup: no projected curve")
	}
	if got := projectedCurveOverlay(sk, nil, pc); !drawListHasColor(got, chromeTheme.sketchCandidateColor) {
		t.Error("hovered projected curve not drawn in the candidate colour")
	}
	sel := func(e sketch.Entity) bool { return e == pc }
	if got := projectedCurveOverlay(sk, sel, nil); !drawListHasColor(got, chromeTheme.sketchSelectedColor) {
		t.Error("selected projected curve not drawn in the selected colour")
	}
}
