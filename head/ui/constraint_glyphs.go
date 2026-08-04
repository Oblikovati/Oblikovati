//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// Show Constraints overlay: a marker per geometric constraint in the open sketch, drawn on the
// geometry it relates. Clicking one selects it (the app routes the pick ahead of the entity pick)
// and Delete removes it — until now a placed constraint was invisible, so a wrong auto-inferred
// relation could only be undone, never repaired.
//
// The glyph vocabulary follows the inference glyphs beside the cursor (inference_glyphs.go): the
// same dash, tick, twin ticks, corner and diamond, so a relation looks the same while it is being
// suggested and after it is placed. Kinds without a distinct glyph fall back to a small square,
// which is still selectable and deletable — the point of the overlay is reachability, not a
// complete pictogram set.

// constraintGlyphPixels is the marker's half-size on screen. It matches the pick radius the app
// uses, so the drawn target and the clickable target are the same size.
const constraintGlyphPixels = 14

// constraintGlyphSource is the slice of the session this overlay needs: the markers to draw, and
// nothing else. Taking this instead of the whole *app.Session is the audit-I5 consumer-interface
// pattern — the signature now says the overlay reads and cannot touch the model.
type constraintGlyphSource interface {
	SketchConstraintGlyphs() []app.ConstraintGlyphView
}

var _ constraintGlyphSource = (*app.Session)(nil)

// constraintGlyphOverlay builds the marker items for the active sketch: one line item for the
// unselected markers and one for the selected, so the selection reads at a glance.
func constraintGlyphOverlay(s constraintGlyphSource, plane sketch.Plane, hWorld float64) []renderer.DrawItem {
	plain, chosen := &segAccum{}, &segAccum{}
	for _, g := range s.SketchConstraintGlyphs() {
		acc := plain
		if g.Selected {
			acc = chosen
		}
		constraintGlyphSegments(acc, plane, g.At, hWorld, g.Kind)
	}
	var items []renderer.DrawItem
	items = appendGrid(items, plain, chromeTheme.sketchColor)
	return appendGrid(items, chosen, chromeTheme.sketchSelectedColor)
}

// constraintGlyphStrokes maps a constraint kind to the glyph drawn for it. Kinds sharing a meaning
// share a stroke set — the four ellipse-axis relations draw as their plain counterparts, because
// to the user they ARE parallel, perpendicular and collinear.
var constraintGlyphStrokes = map[sketch.ConstraintKind]func(acc *segAccum, plane sketch.Plane, at math.Point2, s math.Scalar){
	sketch.CoincidentKind:           diamondGlyph,
	sketch.HorizontalKind:           horizontalGlyph,
	sketch.SingleLineHorizontalKind: horizontalGlyph,
	sketch.EllipseHorizontalKind:    horizontalGlyph,
	sketch.VerticalKind:             verticalGlyph,
	sketch.SingleLineVerticalKind:   verticalGlyph,
	sketch.EllipseVerticalKind:      verticalGlyph,
	sketch.ParallelKind:             parallelGlyph,
	sketch.EllipseParallelKind:      parallelGlyph,
	sketch.PerpendicularKind:        perpendicularGlyph,
	sketch.EllipsePerpendicularKind: perpendicularGlyph,
	sketch.CollinearKind:            collinearGlyph,
	sketch.EllipseCollinearKind:     collinearGlyph,
	sketch.TangentKind:              tangentGlyph,
	sketch.CircularTangentKind:      tangentGlyph,
	sketch.ConcentricKind:           concentricGlyph,
	sketch.EqualRadiusKind:          equalGlyph,
	sketch.EqualLengthKind:          equalGlyph,
	sketch.FixKind:                  fixGlyph,
	sketch.GroundKind:               fixGlyph,
}

// constraintGlyphSegments appends one marker's strokes at the anchor (half-size h in model units).
func constraintGlyphSegments(acc *segAccum, plane sketch.Plane, at math.Point2, h float64, kind sketch.ConstraintKind) {
	s := math.Scalar(h)
	if strokes, known := constraintGlyphStrokes[kind]; known {
		strokes(acc, plane, at, s)
		return
	}
	boxGlyph(acc, plane, at, s)
}

// The glyph strokes. Each draws inside the ±s box centred on at.
func horizontalGlyph(acc *segAccum, plane sketch.Plane, at math.Point2, s math.Scalar) {
	acc.seg(plane, math.P2(at.X-s, at.Y), math.P2(at.X+s, at.Y))
}

func verticalGlyph(acc *segAccum, plane sketch.Plane, at math.Point2, s math.Scalar) {
	acc.seg(plane, math.P2(at.X, at.Y-s), math.P2(at.X, at.Y+s))
}

func parallelGlyph(acc *segAccum, plane sketch.Plane, at math.Point2, s math.Scalar) {
	acc.seg(plane, math.P2(at.X-s, at.Y-s/2), math.P2(at.X+s, at.Y-s/2))
	acc.seg(plane, math.P2(at.X-s, at.Y+s/2), math.P2(at.X+s, at.Y+s/2))
}

func perpendicularGlyph(acc *segAccum, plane sketch.Plane, at math.Point2, s math.Scalar) {
	acc.seg(plane, math.P2(at.X-s, at.Y-s), math.P2(at.X-s, at.Y+s))
	acc.seg(plane, math.P2(at.X-s, at.Y-s), math.P2(at.X+s, at.Y-s))
}

func collinearGlyph(acc *segAccum, plane sketch.Plane, at math.Point2, s math.Scalar) {
	acc.seg(plane, math.P2(at.X-s, at.Y), math.P2(at.X-s/4, at.Y))
	acc.seg(plane, math.P2(at.X+s/4, at.Y), math.P2(at.X+s, at.Y))
}

// tangentGlyph is a line meeting a curve's apex — the touch that tangency means.
func tangentGlyph(acc *segAccum, plane sketch.Plane, at math.Point2, s math.Scalar) {
	acc.seg(plane, math.P2(at.X-s, at.Y+s/2), math.P2(at.X+s, at.Y+s/2))
	acc.seg(plane, math.P2(at.X-s, at.Y-s), math.P2(at.X, at.Y+s/2))
	acc.seg(plane, math.P2(at.X, at.Y+s/2), math.P2(at.X+s, at.Y-s))
}

// concentricGlyph is two nested diamonds — one centre, two radii.
func concentricGlyph(acc *segAccum, plane sketch.Plane, at math.Point2, s math.Scalar) {
	diamondGlyph(acc, plane, at, s)
	diamondGlyph(acc, plane, at, s/2)
}

// equalGlyph is the equals sign, for equal length and equal radius alike.
func equalGlyph(acc *segAccum, plane sketch.Plane, at math.Point2, s math.Scalar) {
	acc.seg(plane, math.P2(at.X-s, at.Y-s/3), math.P2(at.X+s, at.Y-s/3))
	acc.seg(plane, math.P2(at.X-s, at.Y+s/3), math.P2(at.X+s, at.Y+s/3))
}

// fixGlyph is a crossed box — pinned in place.
func fixGlyph(acc *segAccum, plane sketch.Plane, at math.Point2, s math.Scalar) {
	boxGlyph(acc, plane, at, s)
	acc.seg(plane, math.P2(at.X-s, at.Y-s), math.P2(at.X+s, at.Y+s))
	acc.seg(plane, math.P2(at.X-s, at.Y+s), math.P2(at.X+s, at.Y-s))
}

func diamondGlyph(acc *segAccum, plane sketch.Plane, at math.Point2, s math.Scalar) {
	acc.seg(plane, math.P2(at.X, at.Y-s), math.P2(at.X+s, at.Y))
	acc.seg(plane, math.P2(at.X+s, at.Y), math.P2(at.X, at.Y+s))
	acc.seg(plane, math.P2(at.X, at.Y+s), math.P2(at.X-s, at.Y))
	acc.seg(plane, math.P2(at.X-s, at.Y), math.P2(at.X, at.Y-s))
}

func boxGlyph(acc *segAccum, plane sketch.Plane, at math.Point2, s math.Scalar) {
	acc.seg(plane, math.P2(at.X-s, at.Y-s), math.P2(at.X+s, at.Y-s))
	acc.seg(plane, math.P2(at.X+s, at.Y-s), math.P2(at.X+s, at.Y+s))
	acc.seg(plane, math.P2(at.X+s, at.Y+s), math.P2(at.X-s, at.Y+s))
	acc.seg(plane, math.P2(at.X-s, at.Y+s), math.P2(at.X-s, at.Y-s))
}
