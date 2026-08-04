//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// Inference glyph overlay (M06-F10, #625; deferred-work registry #599 row
// "inference glyph overlay"): while a segment is being rubber-banded, the
// glyph of every active inference draws beside the cursor — a dash for
// horizontal, a vertical tick for vertical, twin ticks for parallel, a corner
// for perpendicular, a diamond for a coincident snap — so the user sees what
// the commit will auto-constrain.

// inferenceGlyphs builds the glyph line item for the in-progress tool segment
// at the cursor (the same preview feed toolPreview uses).
func inferenceGlyphs(s *app.Session, plane sketch.Plane, hWorld float64) (renderer.DrawItem, bool) {
	sk := s.ActiveSketch()
	if sk == nil || s.ActiveTool() == nil || !s.ShowConstraintsOnCreation() {
		return renderer.DrawItem{}, false
	}
	cx, cy := viewportCursor()
	cur, ok := s.CursorSketchPoint(cx, cy)
	if !ok {
		return renderer.DrawItem{}, false
	}
	pts, _ := s.ActiveToolPreview(cur)
	if len(pts) < 2 {
		return renderer.DrawItem{}, false
	}
	suggestions := sk.GlyphSuggestions(pts[len(pts)-2], cur, s.SketchInferenceOptions())
	if len(suggestions) == 0 {
		return renderer.DrawItem{}, false
	}
	acc := &segAccum{}
	for i, sg := range suggestions {
		anchor := glyphAnchor(cur, i, hWorld)
		glyphSegments(acc, plane, anchor, hWorld, sg.Kind)
	}
	return renderer.DrawItem{Primitive: renderer.Lines, Positions: acc.pos, Indices: acc.idx, Color: chromeTheme.previewColor}, true
}

// glyphAnchor places the i-th glyph in a row beside the cursor.
func glyphAnchor(cur math.Point2, i int, h float64) math.Point2 {
	offset := math.Scalar(h * 3)
	step := math.Scalar(h * 2.5)
	return math.P2(cur.X+offset+step*math.Scalar(i), cur.Y+offset)
}

// glyphSegments appends one glyph's strokes at the anchor (half-size h).
func glyphSegments(acc *segAccum, plane sketch.Plane, at math.Point2, h float64, kind sketch.SuggestionKind) {
	s := math.Scalar(h)
	switch kind {
	case sketch.SuggestHorizontal:
		acc.seg(plane, math.P2(at.X-s, at.Y), math.P2(at.X+s, at.Y))
	case sketch.SuggestVertical:
		acc.seg(plane, math.P2(at.X, at.Y-s), math.P2(at.X, at.Y+s))
	case sketch.SuggestParallel:
		acc.seg(plane, math.P2(at.X-s, at.Y-s/2), math.P2(at.X+s, at.Y-s/2))
		acc.seg(plane, math.P2(at.X-s, at.Y+s/2), math.P2(at.X+s, at.Y+s/2))
	case sketch.SuggestPerpendicular:
		acc.seg(plane, math.P2(at.X-s, at.Y-s), math.P2(at.X-s, at.Y+s))
		acc.seg(plane, math.P2(at.X-s, at.Y-s), math.P2(at.X+s, at.Y-s))
	case sketch.SuggestCoincident:
		acc.seg(plane, math.P2(at.X, at.Y-s), math.P2(at.X+s, at.Y))
		acc.seg(plane, math.P2(at.X+s, at.Y), math.P2(at.X, at.Y+s))
		acc.seg(plane, math.P2(at.X, at.Y+s), math.P2(at.X-s, at.Y))
		acc.seg(plane, math.P2(at.X-s, at.Y), math.P2(at.X, at.Y-s))
	}
}
