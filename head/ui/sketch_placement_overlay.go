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
// dashed construction geometry, and a dimension — extension lines, an arrowed dimension line and
// its value — standing off each measured edge. gap is the stand-off distance in model units.
func placementOverlayItems(s placementSession, plane sketch.Plane, cursor math.Point2, gap float64) []renderer.DrawItem {
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
	return appendPlacementItem(items, placementDimensionSegments(s, plane, gap), chromeTheme.dimensionSketchColor)
}

// placementDimensionSegments accumulates each in-place dimension: two extension lines running from
// the measured edge out past the dimension line, the dimension line itself, and an arrowhead at
// each of its ends.
//
// The dimension stands OFF the geometry along the field's outward direction. It used to be drawn
// straight along the witness segment — which for a rectangle IS an edge — so the whole annotation
// lay on the shape it measured (#2032).
func placementDimensionSegments(s placementSession, plane sketch.Plane, gap float64) *segAccum {
	acc := &segAccum{}
	for _, f := range s.PlacementFields() {
		off := f.Outward.Scale(math.Scalar(gap))
		a, b := f.Witness[0].TranslateBy(off), f.Witness[1].TranslateBy(off)
		acc.patterned(plane, []math.Point2{f.Witness[0], a.TranslateBy(f.Outward.Scale(math.Scalar(gap * extensionOvershoot)))}, false, placementWitnessPattern)
		acc.patterned(plane, []math.Point2{f.Witness[1], b.TranslateBy(f.Outward.Scale(math.Scalar(gap * extensionOvershoot)))}, false, placementWitnessPattern)
		acc.seg(plane, a, b)
		appendArrowhead(acc, plane, a, a.VectorTo(b), gap)
		appendArrowhead(acc, plane, b, b.VectorTo(a), gap)
	}
	return acc
}

// extensionOvershoot is how far an extension line runs PAST the dimension line, as a fraction of
// the stand-off gap — the small tick beyond the arrow that drafting convention uses.
const extensionOvershoot = 0.25

// arrowheadSpread is the half-angle of an arrowhead's barbs, as a fraction of the head's length
// measured across the dimension line.
const arrowheadSpread = 0.35

// appendArrowhead adds the two barbs of an arrowhead at tip, opening along dir (which points back
// down the dimension line). Its size follows the stand-off gap, so the arrow keeps its proportion
// at every zoom level.
func appendArrowhead(acc *segAccum, plane sketch.Plane, tip math.Point2, dir math.Vector2, gap float64) {
	if dir.Length() == 0 {
		return
	}
	u := dir.Scale(1 / dir.Length())
	perp := math.V2(-u.Y, u.X)
	length := math.Scalar(gap * 0.6)
	base := tip.TranslateBy(u.Scale(length))
	half := perp.Scale(length * arrowheadSpread)
	acc.seg(plane, tip, base.TranslateBy(half))
	acc.seg(plane, tip, base.TranslateBy(half.Scale(-1)))
}

// accumulatePlacementCurve adds one preview curve to whichever accumulator matches its style.
func accumulatePlacementCurve(plane sketch.Plane, c sketch.PreviewCurve, solid, construction *segAccum) {
	if c.Construction {
		construction.patterned(plane, c.Points, c.Closed, linetype.Builtin(types.SketchLineDashed))
		return
	}
	solid.polyline(plane, c.Points, c.Closed)
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
