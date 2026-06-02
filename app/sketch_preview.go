// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"

	"github.com/Oblikovati/oblikovati/math"
)

// Live preview: while a geometry tool has some clicks but not enough to commit, it
// shows a provisional "rubber-band" shape from the placed points through the current
// cursor, so the user sees what they are drawing. The shapes are pure sketch-space
// polylines; the head maps them to model space and draws them.

// PreviewTool is a geometry tool that can show a provisional polyline at the cursor.
type PreviewTool interface {
	// PreviewPolyline returns the provisional sketch-space points (and whether the
	// polyline is closed), or nil when there is nothing to preview yet.
	PreviewPolyline(cursor math.Point2) (pts []math.Point2, closed bool)
}

// ActiveToolPreview returns the active tool's provisional polyline at the cursor (nil
// when no tool is active or it has nothing to preview).
func (s *Session) ActiveToolPreview(cursor math.Point2) (pts []math.Point2, closed bool) {
	if s.tool == nil {
		return nil, false
	}
	pv, ok := s.tool.tool.(PreviewTool)
	if !ok {
		return nil, false
	}
	return pv.PreviewPolyline(cursor)
}

// CursorSketchPoint maps a viewport pixel to a snapped point in the active sketch's
// plane — the point a click would place, used to drive the preview.
func (s *Session) CursorSketchPoint(px, py float64) (math.Point2, bool) {
	return s.sketchClickPoint(px, py)
}

// PreviewPolyline draws the line from its first endpoint to the cursor.
func (t *LineTool) PreviewPolyline(cursor math.Point2) ([]math.Point2, bool) {
	if len(t.points) == 1 {
		return []math.Point2{t.points[0], cursor}, false
	}
	return nil, false
}

// PreviewPolyline draws the rectangle from its first corner to the cursor.
func (t *RectangleTool) PreviewPolyline(cursor math.Point2) ([]math.Point2, bool) {
	if len(t.corners) == 1 {
		a := t.corners[0]
		return []math.Point2{a, math.P2(cursor.X, a.Y), cursor, math.P2(a.X, cursor.Y)}, true
	}
	return nil, false
}

// PreviewPolyline draws the circle from its centre with the cursor on the rim.
func (t *CircleTool) PreviewPolyline(cursor math.Point2) ([]math.Point2, bool) {
	if len(t.pts) == 1 {
		return sampleCirclePoints(t.pts[0], t.pts[0].DistanceTo(cursor)), true
	}
	return nil, false
}

// PreviewPolyline draws the regular polygon from its centre through the cursor vertex.
func (t *PolygonTool) PreviewPolyline(cursor math.Point2) ([]math.Point2, bool) {
	if len(t.pts) == 1 {
		return polygonVertices(t.pts[0], cursor, t.Sides), true
	}
	return nil, false
}

// PreviewPolyline draws the spline through its placed points and the cursor.
func (t *SplineTool) PreviewPolyline(cursor math.Point2) ([]math.Point2, bool) {
	if len(t.pts) == 0 {
		return nil, false
	}
	return append(append([]math.Point2(nil), t.pts...), cursor), false
}

// previewCircleSegments is the polyline resolution for the circle preview; it mirrors
// the head's sketchSegments so the rubber-band matches the committed wireframe.
const previewCircleSegments = 64

// sampleCirclePoints returns a closed polyline approximating a circle.
func sampleCirclePoints(center math.Point2, r float64) []math.Point2 {
	pts := make([]math.Point2, previewCircleSegments)
	for i := range pts {
		a := 2 * stdmath.Pi * float64(i) / float64(previewCircleSegments)
		pts[i] = math.P2(center.X+r*stdmath.Cos(a), center.Y+r*stdmath.Sin(a))
	}
	return pts
}
