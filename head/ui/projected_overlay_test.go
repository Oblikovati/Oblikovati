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

// TestProjectedCurveOverlayDrawsReferenceLines: a projected curve is a concrete reference line now
// (ADR-0055 phase 3), so the standard sketch overlay draws it — no separate projected-curve overlay.
func TestProjectedCurveOverlayDrawsReferenceLines(t *testing.T) {
	if got := sketchOverlay(projectedSketch(t), nil, nil, false); len(got) == 0 {
		t.Fatal("the projected reference line must be drawn by the sketch overlay")
	}
	if got := sketchOverlay(nil, nil, nil, false); got != nil {
		t.Errorf("sketchOverlay(nil, …) = %v, want nil", got)
	}
}

// TestProjectedCurveOverlayEmptyWhenNoProjection: a plain, geometry-less sketch draws nothing.
func TestProjectedCurveOverlayEmptyWhenNoProjection(t *testing.T) {
	s := app.NewSession()
	pd, _ := compdef.AddPart(s.Workspace(), "q.opd", true)
	sk := pd.Content().(*compdef.PartComponentDefinition).Sketches().Add(sketch.XYPlane())
	if got := sketchOverlay(sk, nil, nil, false); len(got) != 0 {
		t.Errorf("sketch overlay of a geometry-less sketch = %d items, want 0", len(got))
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

// referenceCurveEntity returns the concrete reference entity of the sketch's first curve projection
// (ADR-0055 phase 3).
func referenceCurveEntity(sk *sketch.Sketch) sketch.Entity {
	ps := sk.Projections()
	if len(ps) == 0 {
		return nil
	}
	return ps[0].Entity()
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
	pc := referenceCurveEntity(sk)
	if pc == nil {
		t.Fatal("setup: no projected curve")
	}
	if got := sketchOverlay(sk, nil, pc, false); !drawListHasColor(got, chromeTheme.sketchCandidateColor) {
		t.Error("hovered projected curve not drawn in the candidate colour")
	}
	sel := func(e sketch.Entity) bool { return e == pc }
	if got := sketchOverlay(sk, sel, nil, false); !drawListHasColor(got, chromeTheme.sketchSelectedColor) {
		t.Error("selected projected curve not drawn in the selected colour")
	}
}
