//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/linetype"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// The placement overlay (#2014): what the user sees while a sketch shape is being placed.
// Real geometry draws solid, construction geometry dashed with the same pattern committed
// construction geometry uses (so the preview looks like the result), and a fine dotted witness
// line runs to each in-place dimension box.
//
// Everything here reads app.Session's recipe-derived views; the head decides only how it looks.

// placementWitnessPattern is the dash pattern for the extension lines running to the dimension
// boxes — finer than the construction dash so the two read as different things.
var placementWitnessPattern = []float64{0.08, 0.12}

// placementSession is the session surface the placement overlay reads (audit I5, the
// arrowSession pattern): the shape being placed and the input boxes describing it.
type placementSession interface {
	ActiveToolPreviewCurves(cursor math.Point2) []sketch.PreviewCurve
	PlacementFields() []app.PlacementFieldView
}

var _ placementSession = (*app.Session)(nil)

// placementOverlayItems returns the draw items for the shape being placed: solid geometry,
// dashed construction geometry, and dotted witness lines to the input boxes.
func placementOverlayItems(s placementSession, plane sketch.Plane, cursor math.Point2) []renderer.DrawItem {
	curves := s.ActiveToolPreviewCurves(cursor)
	if len(curves) == 0 {
		return nil
	}
	solid, construction := &segAccum{}, &segAccum{}
	for _, c := range curves {
		accumulatePlacementCurve(plane, c, solid, construction)
	}
	items := appendPlacementItem(nil, solid, chromeTheme.previewColor)
	items = appendPlacementItem(items, construction, chromeTheme.previewColor)
	return appendPlacementItem(items, placementWitnessSegments(s, plane), chromeTheme.previewColor)
}

// accumulatePlacementCurve adds one preview curve to whichever accumulator matches its style.
func accumulatePlacementCurve(plane sketch.Plane, c sketch.PreviewCurve, solid, construction *segAccum) {
	if c.Construction {
		construction.patterned(plane, c.Points, c.Closed, linetype.Builtin(types.SketchLineDashed))
		return
	}
	solid.polyline(plane, c.Points, c.Closed)
}

// placementWitnessSegments accumulates the dotted extension line under each input box.
func placementWitnessSegments(s placementSession, plane sketch.Plane) *segAccum {
	acc := &segAccum{}
	for _, f := range s.PlacementFields() {
		acc.patterned(plane, []math.Point2{f.Witness[0], f.Witness[1]}, false, placementWitnessPattern)
	}
	return acc
}

// appendPlacementItem appends the accumulator's lines as one draw item, skipping empties.
func appendPlacementItem(items []renderer.DrawItem, acc *segAccum, color [4]float32) []renderer.DrawItem {
	if len(acc.idx) == 0 {
		return items
	}
	return append(items, renderer.DrawItem{
		Primitive: renderer.Lines, Positions: acc.pos, Indices: acc.idx, Color: color,
	})
}

// splitRecipeGeometry separates a preview recipe's real geometry from its construction geometry.
// It is the pure counterpart of accumulatePlacementCurve, kept testable without a GPU.
func splitRecipeGeometry(curves []sketch.PreviewCurve) (solid, construction []sketch.PreviewCurve) {
	for _, c := range curves {
		if c.Construction {
			construction = append(construction, c)
			continue
		}
		solid = append(solid, c)
	}
	return solid, construction
}
