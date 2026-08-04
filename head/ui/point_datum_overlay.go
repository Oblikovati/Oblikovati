//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/renderer"
)

// Drawing the model's datum points — the origin Center Point and user work points. Until #2016
// nothing drew them at all: a work point carried a Visible flag and the picker offered it as a
// snap target, but no overlay ever turned it into pixels, so showing the Center Point had no
// effect and clicking near the origin snapped to something invisible.

// datumPointPixels is the datum-point cross's arm length on screen. A little larger than a
// placed sketch point's marker so a datum reads as model reference geometry, not sketch geometry.
const datumPointPixels = 6.0

// pointsDatumOverlay draws each visible datum point as a small three-axis cross, sized in world
// units so the caller can keep it screen-constant. Hidden points are skipped — the rule
// axesOverlay follows, and the one [app.Session.PickableWorkPoints] mirrors so that what is not
// drawn is not clickable.
func pointsDatumOverlay(points *feature.WorkPoints, selected *feature.WorkPoint, hidden scopeFilter, hWorld float64) []renderer.DrawItem {
	if points == nil {
		return nil
	}
	items := make([]renderer.DrawItem, 0, points.Count())
	for i := 0; i < points.Count(); i++ {
		pt := points.Item(i)
		if pt.Visible() && !hidden(pt.Seq()) {
			items = append(items, datumPointCross(pt, datumPointColor(pt, selected), hWorld))
		}
	}
	return items
}

// datumPointColor highlights the selected datum point, matching how an axis and a plane colour
// theirs.
func datumPointColor(pt, selected *feature.WorkPoint) [4]float32 {
	if pt == selected {
		return chromeTheme.selectedPlaneColor
	}
	return chromeTheme.faintPlaneColor
}

// datumPointCross builds the three axis-aligned segments through a datum point. A cross rather
// than a dot because a single point is one pixel at any zoom, and because it reads as a datum
// (Inventor's work-point glyph) instead of as sketch geometry.
func datumPointCross(pt *feature.WorkPoint, color [4]float32, h float64) renderer.DrawItem {
	c := pt.Point()
	pos := []math.Point3{
		math.P3(c.X-h, c.Y, c.Z), math.P3(c.X+h, c.Y, c.Z),
		math.P3(c.X, c.Y-h, c.Z), math.P3(c.X, c.Y+h, c.Z),
		math.P3(c.X, c.Y, c.Z-h), math.P3(c.X, c.Y, c.Z+h),
	}
	return renderer.DrawItem{
		Primitive: renderer.Lines,
		Positions: pos,
		Indices:   []int{0, 1, 2, 3, 4, 5},
		Color:     color,
		OnTop:     true, // a reference marker: it stays legible against the body it sits on
	}
}

// selectedWorkPoint returns the first selected datum point, or nil — the counterpart of
// selectedWorkAxis, used to highlight it in the overlay.
func selectedWorkPoint(sel *app.Selection) *feature.WorkPoint {
	for _, item := range sel.Items() {
		if h, ok := item.(app.WorkPointHandle); ok {
			return h.Point
		}
	}
	return nil
}
