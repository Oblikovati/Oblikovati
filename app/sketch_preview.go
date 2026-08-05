// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Live preview: while a geometry tool has some clicks but not enough to commit, it shows the
// shape it would create at the current cursor. It reads the same [sketch.Recipe] the commit
// applies, so what is drawn while dragging and what appears on release cannot disagree — the
// preview used to hold its own approximation of each shape, which was the third of three
// definitions of a polygon in this codebase (#2014).
//
// Only five of the seventeen Create tools previewed at all before this; the rest drew nothing
// while placing.

// RecipeTool is a geometry tool that can describe the shape it would create at the cursor.
// locked carries each input field's typed override ("" ⇒ that field tracks the cursor), so a
// locked width freezes that dimension while the drag changes only the rest.
type RecipeTool interface {
	PendingRecipe(s *Session, cursor math.Point2, locked []string) (sketch.Recipe, bool)
}

// ActiveToolRecipe returns the active tool's provisional shape at the cursor, honouring any
// locked input fields. ok is false when no tool is active or it has too few points yet.
func (s *Session) ActiveToolRecipe(cursor math.Point2) (sketch.Recipe, bool) {
	if s.tool == nil {
		return sketch.Recipe{}, false
	}
	rt, ok := s.tool.tool.(RecipeTool)
	if !ok {
		return sketch.Recipe{}, false
	}
	return rt.PendingRecipe(s, cursor, s.placementFieldValues())
}

// ActiveToolPreviewCurves returns the provisional shape as styled curves — real geometry and
// construction geometry kept apart so the head can draw them solid and dashed respectively.
func (s *Session) ActiveToolPreviewCurves(cursor math.Point2) []sketch.PreviewCurve {
	r, ok := s.ActiveToolRecipe(cursor)
	if !ok {
		return nil
	}
	return sketch.RecipeCurves(r)
}

// CursorSketchPoint maps a viewport pixel to a snapped point in the active sketch's plane — the
// point a click would place, used to drive the preview. It remembers the result so the in-place
// input fields can be rebuilt for the same cursor without re-deriving it.
func (s *Session) CursorSketchPoint(px, py float64) (math.Point2, bool) {
	p, ok := s.sketchClickPoint(px, py)
	if ok {
		s.lastCursorSketchPoint = p
	}
	return p, ok
}

// ActiveToolPreview returns the provisional shape's real geometry as one polyline. It is
// derived from the recipe, so it cannot drift from ActiveToolPreviewCurves; the inference-glyph
// overlay reads it to find the segment being rubber-banded.
func (s *Session) ActiveToolPreview(cursor math.Point2) (pts []math.Point2, closed bool) {
	r, ok := s.ActiveToolRecipe(cursor)
	if !ok {
		return nil, false
	}
	return sketch.RecipeOutline(r)
}

// --- per-tool pending shapes ----------------------------------------------
//
// Each tool answers one question: given the points already placed and the cursor, what shape am
// I making? Everything else — preview, glyphs, input fields, and the commit itself — follows
// from that one answer.

// PendingRecipe previews the WHOLE chain placed so far, ending at the cursor.
//
// It used to describe only the segment being rubber-banded, so in a continuous chain every
// segment already placed was invisible until the command finished: the tool holds its points and
// creates all the lines at once in Commit, so nothing existed in the sketch to draw, and the
// preview — the only thing that could have shown them — drew just the last one (#2030). The
// preview's contract is the shape the commit would create, and for a chain that is all of it.
func (t *LineTool) PendingRecipe(_ *Session, cursor math.Point2, _ []string) (sketch.Recipe, bool) {
	if len(t.points) == 0 {
		return sketch.Recipe{}, false
	}
	return sketch.LineChainRecipe(append(append([]math.Point2(nil), t.points...), cursor)), true
}

// PendingRecipe previews the rectangle from its placed corner through the cursor, with any
// locked Width or Height overriding what the cursor would give.
func (t *RectangleTool) PendingRecipe(s *Session, cursor math.Point2, locked []string) (sketch.Recipe, bool) {
	if len(t.corners) != 1 {
		return sketch.Recipe{}, false
	}
	return sketch.RectangleRecipe(t.corners[0], s.lockedCorner(t.corners[0], cursor, locked)), true
}

// PendingRecipe previews the rotated rectangle once its base edge is placed.
func (t *ThreePointRectangleTool) PendingRecipe(_ *Session, cursor math.Point2, _ []string) (sketch.Recipe, bool) {
	if len(t.pts) != 2 {
		return sketch.Recipe{}, false
	}
	return sketch.ThreePointRectangleRecipe(t.pts[0], t.pts[1], cursor), true
}

// PendingRecipe previews the centre rectangle growing about its placed centre.
func (t *CenterRectangleTool) PendingRecipe(s *Session, cursor math.Point2, locked []string) (sketch.Recipe, bool) {
	if len(t.pts) != 1 {
		return sketch.Recipe{}, false
	}
	return sketch.CenterRectangleRecipe(t.pts[0], s.lockedCorner(t.pts[0], cursor, locked)), true
}

// PendingRecipe previews the circle from its placed centre out to the cursor.
func (t *CircleTool) PendingRecipe(s *Session, cursor math.Point2, locked []string) (sketch.Recipe, bool) {
	if len(t.pts) != 1 {
		return sketch.Recipe{}, false
	}
	r := t.pts[0].DistanceTo(cursor)
	if d, ok := s.lockedLength(locked, 0); ok {
		r = d / 2 // the field is a diameter
	}
	return sketch.CircleRecipe(t.pts[0], math.Scalar(r)), true
}

// PendingRecipe previews the three-point circle once two points are placed.
func (t *ThreePointCircleTool) PendingRecipe(_ *Session, cursor math.Point2, _ []string) (sketch.Recipe, bool) {
	if len(t.pts) != 2 {
		return sketch.Recipe{}, false
	}
	center, ok := circumcenter(t.pts[0], t.pts[1], cursor)
	if !ok {
		return sketch.Recipe{}, false // collinear: no circle passes through the three
	}
	return sketch.CircleRecipe(center, math.Scalar(center.DistanceTo(t.pts[0]))), true
}

// PendingRecipe previews the three-point arc once its endpoints are placed; the cursor is the
// point the arc passes through.
func (t *ArcTool) PendingRecipe(_ *Session, cursor math.Point2, _ []string) (sketch.Recipe, bool) {
	if len(t.pts) != 2 {
		return sketch.Recipe{}, false
	}
	start, end := t.pts[0], t.pts[1]
	center, ok := circumcenter(start, cursor, end)
	if !ok {
		return sketch.Recipe{}, false
	}
	return sketch.ArcRecipe(center, start, end, leftTurn(start, cursor, end)), true
}

// PendingRecipe previews the centre-point arc once its centre and start are placed.
func (t *CenterPointArcTool) PendingRecipe(_ *Session, cursor math.Point2, _ []string) (sketch.Recipe, bool) {
	if len(t.pts) != 2 {
		return sketch.Recipe{}, false
	}
	center, start := t.pts[0], t.pts[1]
	return sketch.ArcRecipe(center, start, cursor, leftTurn(center, start, cursor)), true
}

// PendingRecipe previews the ellipse once its centre and major axis are placed; the cursor's
// perpendicular distance from that axis sets the minor radius.
func (t *EllipseTool) PendingRecipe(_ *Session, cursor math.Point2, _ []string) (sketch.Recipe, bool) {
	if len(t.pts) != 2 {
		return sketch.Recipe{}, false
	}
	center, major := t.pts[0], t.pts[1]
	axis := center.VectorTo(major)
	if axis.Length() == 0 {
		return sketch.Recipe{}, false
	}
	unit := axis.Scale(1 / axis.Length())
	minor := math.V2(-unit.Y, unit.X).Dot(center.VectorTo(cursor))
	return sketch.EllipseRecipe(center, unit, math.Scalar(axis.Length()), math.Scalar(stdAbs(minor))), true
}

// PendingRecipe previews the polygon inscribed in the circle through the cursor.
func (t *PolygonTool) PendingRecipe(_ *Session, cursor math.Point2, _ []string) (sketch.Recipe, bool) {
	if len(t.pts) != 1 {
		return sketch.Recipe{}, false
	}
	if t.pts[0].DistanceTo(cursor) == 0 {
		return sketch.Recipe{}, false
	}
	return sketch.PolygonRecipe(t.pts[0], cursor, t.Sides, true), true
}

// PendingRecipe previews the straight slot from its placed centre to the cursor.
func (t *SketchSlotTool) PendingRecipe(_ *Session, cursor math.Point2, _ []string) (sketch.Recipe, bool) {
	if len(t.pts) != 1 || t.pts[0].DistanceTo(cursor) == 0 {
		return sketch.Recipe{}, false
	}
	return sketch.StraightSlotRecipe(t.pts[0], cursor, t.width), true
}

// PendingRecipe previews the centre-point arc slot once its centre and start are placed.
func (t *CenterPointArcSlotTool) PendingRecipe(_ *Session, cursor math.Point2, _ []string) (sketch.Recipe, bool) {
	if len(t.pts) != 2 {
		return sketch.Recipe{}, false
	}
	center, start := t.pts[0], t.pts[1]
	if center.DistanceTo(start) == 0 {
		return sketch.Recipe{}, false
	}
	return sketch.ArcSlotRecipe(center, start, cursor, t.width, leftTurn(center, start, cursor)), true
}

// PendingRecipe previews the three-point arc slot once its endpoints are placed; the cursor is
// the point the arc passes through (the same order as ArcTool's preview — #2028).
func (t *ThreePointArcSlotTool) PendingRecipe(_ *Session, cursor math.Point2, _ []string) (sketch.Recipe, bool) {
	if len(t.pts) != 2 {
		return sketch.Recipe{}, false
	}
	start, end := t.pts[0], t.pts[1]
	center, ok := circumcenter(start, cursor, end)
	if !ok {
		return sketch.Recipe{}, false
	}
	return sketch.ArcSlotRecipe(center, start, end, t.width, leftTurn(start, cursor, end)), true
}

// PendingRecipe previews the spline through its placed fit points and the cursor.
func (t *SplineTool) PendingRecipe(_ *Session, cursor math.Point2, _ []string) (sketch.Recipe, bool) {
	if len(t.pts) == 0 {
		return sketch.Recipe{}, false
	}
	return sketch.SplineRecipe(append(append([]math.Point2(nil), t.pts...), cursor), true), true
}

// PendingRecipe previews the control-vertex spline through its placed vertices and the cursor.
func (t *ControlVertexSplineTool) PendingRecipe(_ *Session, cursor math.Point2, _ []string) (sketch.Recipe, bool) {
	if len(t.pts) == 0 {
		return sketch.Recipe{}, false
	}
	return sketch.SplineRecipe(append(append([]math.Point2(nil), t.pts...), cursor), false), true
}

// PendingRecipe previews the point about to be placed at the cursor.
func (t *PointTool) PendingRecipe(_ *Session, cursor math.Point2, _ []string) (sketch.Recipe, bool) {
	return sketch.PointRecipe(cursor), true
}

// stdAbs is the unsigned magnitude of a scalar measurement.
func stdAbs(v math.Scalar) float64 {
	if v < 0 {
		return float64(-v)
	}
	return float64(v)
}
