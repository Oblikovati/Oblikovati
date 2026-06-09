//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

const (
	// snapGlyphPixels is the snap marker's half-size on screen.
	snapGlyphPixels = 7.0
	// pointMarkerPixels is the persistent marker half-size for placed sketch points.
	pointMarkerPixels = 4.0
	// Equilateral-triangle midpoint marker: an apex pointing up, its base below the
	// center. The base half-width is cos(30°)·h and the base sits sin(30°)·h below.
	triBaseHalfWidth = 0.8660254 // cos(30°) = √3/2
	triBaseDrop      = 0.5       // sin(30°)
)

// snapGlyphColor (the cursor snap marker) and pointMarkerColor (placed sketch points,
// matching the sketch wireframe) are theme-driven and declared in theme_apply.go.

// snapGlyph builds the snap marker at the snapped point: a square for an endpoint, a
// triangle for a line midpoint, a cross for an on-curve (edge) snap. hWorld is the
// half-size in model units (screen-constant via the camera). Returns false for kinds
// with no glyph (grid/none).
func snapGlyph(plane sketch.Plane, r app.SnapResult, hWorld float64) (renderer.DrawItem, bool) {
	acc := &segAccum{}
	c := r.Point
	switch r.Kind {
	case app.SnapPoint:
		acc.polyline(plane, square(c, hWorld), true)
	case app.SnapMidpoint:
		acc.polyline(plane, triangle(c, hWorld), true)
	case app.SnapOnCurve:
		acc.seg(plane, math.P2(c.X-hWorld, c.Y-hWorld), math.P2(c.X+hWorld, c.Y+hWorld))
		acc.seg(plane, math.P2(c.X-hWorld, c.Y+hWorld), math.P2(c.X+hWorld, c.Y-hWorld))
	default:
		return renderer.DrawItem{}, false
	}
	return lineItem(acc, snapGlyphColor), true
}

// pointsOverlay draws Inventor's target glyph (a circle with a centred crosshair) at every
// placed sketch point so it is visible (a bare point is a single pixel). hWorld is the
// half-size in model units (screen-constant via the camera).
func pointsOverlay(plane sketch.Plane, sk *sketch.Sketch, hWorld float64) (renderer.DrawItem, bool) {
	if sk == nil {
		return renderer.DrawItem{}, false
	}
	acc := &segAccum{}
	pts := sk.Points()
	for i := 0; i < pts.Count(); i++ {
		acc.targetMarker(plane, pts.Item(i).Position(), hWorld)
	}
	if len(acc.pos) == 0 {
		return renderer.DrawItem{}, false
	}
	return lineItem(acc, pointMarkerColor), true
}

// targetMarker adds the sketch-point glyph at c: a small circle with a centred crosshair
// (the "target" symbol Inventor uses), screen-constant via hWorld.
func (a *segAccum) targetMarker(plane sketch.Plane, c math.Point2, h float64) {
	a.polyline(plane, circlePoints(c, h*0.7), true)
	a.seg(plane, math.P2(c.X-h, c.Y), math.P2(c.X+h, c.Y))
	a.seg(plane, math.P2(c.X, c.Y-h), math.P2(c.X, c.Y+h))
}

// circlePoints samples a closed circle of radius r about c.
func circlePoints(c math.Point2, r float64) []math.Point2 {
	const n = 16
	pts := make([]math.Point2, n)
	for i := 0; i < n; i++ {
		t := 2 * stdmath.Pi * float64(i) / n
		pts[i] = math.P2(c.X+r*stdmath.Cos(t), c.Y+r*stdmath.Sin(t))
	}
	return pts
}

func lineItem(acc *segAccum, color [4]float32) renderer.DrawItem {
	return renderer.DrawItem{Primitive: renderer.Lines, Positions: acc.pos, Indices: acc.idx, Color: color}
}

func square(c math.Point2, h float64) []math.Point2 {
	return []math.Point2{
		math.P2(c.X-h, c.Y-h), math.P2(c.X+h, c.Y-h), math.P2(c.X+h, c.Y+h), math.P2(c.X-h, c.Y+h),
	}
}

func triangle(c math.Point2, h float64) []math.Point2 {
	return []math.Point2{
		math.P2(c.X, c.Y+h),
		math.P2(c.X-triBaseHalfWidth*h, c.Y-triBaseDrop*h),
		math.P2(c.X+triBaseHalfWidth*h, c.Y-triBaseDrop*h),
	}
}
