// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/math"
)

// Sketch tools place 2D geometry by clicking in the active sketch's plane: a click
// is a camera ray, intersected with the sketch plane and mapped to sketch
// coordinates (Inventor's sketch environment). They are interactive [Tool]s driven
// by the same synthetic input, so a test draws a rectangle by clicking corners.

// screenToSketch maps a clicked pixel to a point in the active sketch's 2D space, by
// intersecting the camera ray with the sketch plane. ok=false if the ray is parallel
// to the plane.
func screenToSketch(s *Session, px, py float64) (math.Point2, bool) {
	sk := s.activeSketch
	origin, dir := s.camera.RayThrough(px, py)
	plane := sk.Plane()
	n := plane.Normal().AsVector()
	denom := dir.Dot(n)
	if math.IsNearZero(denom, math.DefaultTolerance) {
		return math.Point2{}, false
	}
	t := origin.VectorTo(plane.Origin()).Dot(n) / denom
	hit := origin.TranslateBy(dir.Scale(t))
	return plane.ToSketch(hit), true
}

// RectangleTool draws an axis-aligned rectangle from two clicked corners (two
// lines per pair of corners), the most common closed sketch profile.
type RectangleTool struct {
	corners []math.Point2
}

// NewRectangleTool returns a two-corner rectangle tool.
func NewRectangleTool() *RectangleTool { return &RectangleTool{} }

// Name implements [Tool].
func (t *RectangleTool) Name() string { return "Rectangle" }

// Start requires an active sketch (the sketch environment must be open).
func (t *RectangleTool) Start(*Session) {}

// Pick is unused; rectangle corners come from raw clicks (see ClickAt).
func (t *RectangleTool) Pick(*Session, Selectable) {}

// ClickAt records a corner from a clicked pixel (snapped to points/grid).
func (t *RectangleTool) ClickAt(s *Session, px, py float64) {
	if p, ok := s.sketchClickPoint(px, py); ok {
		t.corners = append(t.corners, p)
	}
}

// CanCommit is true once two opposite corners are placed.
func (t *RectangleTool) CanCommit() bool { return len(t.corners) == 2 }

// Commit adds the four rectangle lines (sharing corner points) to the active sketch.
func (t *RectangleTool) Commit(s *Session) error {
	if s.activeSketch == nil {
		return errors.New("rectangle: no active sketch")
	}
	a, c := t.corners[0], t.corners[1]
	b := math.P2(c.X, a.Y)
	d := math.P2(a.X, c.Y)
	sk := s.activeSketch
	p0 := sk.Points().Add(a)
	p1 := sk.Points().Add(b)
	p2 := sk.Points().Add(c)
	p3 := sk.Points().Add(d)
	sk.Lines().Add(p0, p1)
	sk.Lines().Add(p1, p2)
	sk.Lines().Add(p2, p3)
	sk.Lines().Add(p3, p0)
	return nil
}

// Cancel discards the in-progress rectangle.
func (t *RectangleTool) Cancel(*Session) { t.corners = nil }

// AutoCommits creates the rectangle on the second click, then deactivates the tool.
func (t *RectangleTool) AutoCommits() bool { return true }

// Prompt guides the user through placing the two corners (Inventor's status-bar prompts).
func (t *RectangleTool) Prompt(*Session) string {
	switch len(t.corners) {
	case 0:
		return "Click the first corner of the rectangle"
	case 1:
		return "Click the opposite corner"
	default:
		return "Click OK to create the rectangle"
	}
}

// LineTool draws a single line between two clicked points.
type LineTool struct {
	points []math.Point2
}

// NewLineTool returns a two-point line tool.
func NewLineTool() *LineTool { return &LineTool{} }

func (t *LineTool) Name() string              { return "Line" }
func (t *LineTool) Start(*Session)            {}
func (t *LineTool) Pick(*Session, Selectable) {}

// ClickAt records a line endpoint from a clicked pixel (snapped to points/grid).
func (t *LineTool) ClickAt(s *Session, px, py float64) {
	if p, ok := s.sketchClickPoint(px, py); ok {
		t.points = append(t.points, p)
	}
}

// CanCommit is true once both endpoints are placed.
func (t *LineTool) CanCommit() bool { return len(t.points) == 2 }

// Commit adds the line to the active sketch and runs constraint inference on
// it (M06-F10, #625): a near-axis line picks up its horizontal/vertical (or
// parallel/perpendicular, per the session preference) automatically.
func (t *LineTool) Commit(s *Session) error {
	if s.activeSketch == nil {
		return errors.New("line: no active sketch")
	}
	sk := s.activeSketch
	l := sk.Lines().Add(sk.Points().Add(t.points[0]), sk.Points().Add(t.points[1]))
	sk.ApplyLineInference(l, s.SketchInferenceOptions())
	return nil
}

// Cancel discards the in-progress line.
func (t *LineTool) Cancel(*Session) { t.points = nil }

// AutoCommits creates the line on the second click, then deactivates the tool.
func (t *LineTool) AutoCommits() bool { return true }

// Prompt guides the user through placing the two endpoints (Inventor's status-bar prompts).
func (t *LineTool) Prompt(*Session) string {
	switch len(t.points) {
	case 0:
		return "Click the line start point"
	case 1:
		return "Click the line end point"
	default:
		return "Click OK to create the line"
	}
}

// PlaneClickTool is a sketch tool whose input is raw plane-point clicks (a pixel
// mapped through the sketch plane), not entity picks — every geometry tool in the
// sketch environment (line, rectangle, circle, arc, …) implements it.
type PlaneClickTool interface {
	ClickAt(s *Session, px, py float64)
}

// AutoCommitTool is a fixed-arity geometry tool that creates its geometry as soon as it
// has enough clicks and then deactivates (one shape per activation), instead of waiting
// for a separate OK. Variable-length tools (spline) omit it and commit on OK.
type AutoCommitTool interface {
	AutoCommits() bool
}

// sketchClick routes a sketch-plane click to the active tool when it consumes plane
// points, returning true if it handled the click (so it is not treated as a pick). When
// the tool is an [AutoCommitTool] and now has enough input, the geometry is created
// immediately and the tool deactivates — so a single click sequence produces the shape.
func (s *Session) sketchClick(px, py float64) bool {
	if s.tool == nil {
		return false
	}
	ct, ok := s.tool.tool.(PlaneClickTool)
	if !ok {
		return false
	}
	ct.ClickAt(s, px, py)
	s.autoCommitSketchTool()
	return true
}

// autoCommitSketchTool commits an auto-commit tool once it is ready; OK clears the tool
// so it deactivates after one shape (a failed commit keeps it open to re-click).
func (s *Session) autoCommitSketchTool() {
	ac, ok := s.tool.tool.(AutoCommitTool)
	if !ok || !ac.AutoCommits() || !s.tool.tool.CanCommit() {
		return
	}
	_ = s.OK()
}
