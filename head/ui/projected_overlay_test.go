//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
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
	if got := projectedCurveOverlay(projectedSketch(t)); len(got) != 1 {
		t.Fatalf("projectedCurveOverlay = %d items, want 1 (the projected reference line)", len(got))
	}
	if got := projectedCurveOverlay(nil); got != nil {
		t.Errorf("projectedCurveOverlay(nil) = %v, want nil", got)
	}
}

// TestProjectedCurveOverlayEmptyWhenNoProjection: a plain sketch projects no curves.
func TestProjectedCurveOverlayEmptyWhenNoProjection(t *testing.T) {
	s := app.NewSession()
	pd, _ := compdef.AddPart(s.Workspace(), "q.opd", true)
	sk := pd.Content().(*compdef.PartComponentDefinition).Sketches().Add(sketch.XYPlane())
	if got := projectedCurveOverlay(sk); got != nil {
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
