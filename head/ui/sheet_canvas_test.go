//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/math"
)

// TestFitSheetCentersAndScales checks the sheet fits inside the panel with padding,
// preserves aspect (uniform scale), and is centered.
func TestFitSheetCentersAndScales(t *testing.T) {
	// A landscape A3 (420×297 mm) in an 840×600 panel at origin (0,0). After 40 px
	// padding each side: width fit (840-80)/420 ≈ 1.810, height fit (600-80)/297 ≈ 1.751.
	// Height is the tighter constraint, so it sets the uniform scale.
	r := fitSheet(0, 0, 840, 600, 420, 297)
	wantScale := float32((600.0 - 80) / 297.0)
	if d := r.scale - wantScale; d < -1e-3 || d > 1e-3 {
		t.Errorf("scale = %g, want %g", r.scale, wantScale)
	}
	if d := r.w - 420*r.scale; d < -1e-3 || d > 1e-3 {
		t.Errorf("width = %g, want %g", r.w, 420*r.scale)
	}
	// Centered: equal margins left/right.
	if leftMargin, rightMargin := r.x, 840-(r.x+r.w); abs32(leftMargin-rightMargin) > 1e-2 {
		t.Errorf("not horizontally centered: left=%g right=%g", leftMargin, rightMargin)
	}
	if r.y <= 0 || r.y+r.h >= 600 {
		t.Errorf("sheet not within panel vertically: y=%g h=%g", r.y, r.h)
	}
}

// TestTitleBlockRectAnchorsLowerRight checks the title block sits at the inner area's
// lower-right corner and stays within it.
func TestTitleBlockRectAnchorsLowerRight(t *testing.T) {
	inner := rect{x: 100, y: 50, w: 600, h: 400, scale: 2}
	box := titleBlockRect(inner, 6)
	if box.x+box.w-(inner.x+inner.w) > 1e-3 || box.y+box.h-(inner.y+inner.h) > 1e-3 {
		t.Errorf("title block not anchored to lower-right: box=%+v inner=%+v", box, inner)
	}
	if box.x < inner.x || box.y < inner.y {
		t.Errorf("title block escapes the inner area: box=%+v inner=%+v", box, inner)
	}
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// TestScreenSheetRoundTrip checks the sheet↔screen mapping inverts: a sheet point mapped to
// the screen and back lands where it started (the basis for cursor placement and hit-testing).
func TestScreenSheetRoundTrip(t *testing.T) {
	face := rect{x: 100, y: 50, w: 600, h: 400, scale: 2}
	for _, p := range []math.Point2{math.P2(0, 0), math.P2(120, 90), math.P2(300, 200)} {
		sx, sy := curveToScreen(face, p)
		cx, cy := screenToSheet(face, sx, sy)
		if abs32(float32(cx)-float32(p.X)) > 1e-2 || abs32(float32(cy)-float32(p.Y)) > 1e-2 {
			t.Errorf("round-trip %v → screen(%g,%g) → sheet(%g,%g)", p, sx, sy, cx, cy)
		}
	}
}

// TestOffset2 checks the placement offset shifts a curve point by the cursor position.
func TestOffset2(t *testing.T) {
	got := offset2(math.P2(10, 20), 5, -3)
	if got.X != 15 || got.Y != 17 {
		t.Errorf("offset2 = %v, want (15,17)", got)
	}
}
