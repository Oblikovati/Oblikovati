// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
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
	dialogTool
	corners []math.Point2
}

// NewRectangleTool returns a two-corner rectangle tool.
func NewRectangleTool() *RectangleTool { return &RectangleTool{} }

// Name implements [Tool].
func (t *RectangleTool) Name() string { return "Rectangle" }

// Start requires an active sketch (the sketch environment must be open).

// Pick is unused; rectangle corners come from raw clicks (see ClickAt).

// ClickAt records a corner from a clicked pixel (snapped to points/grid).
func (t *RectangleTool) ClickAt(s *Session, px, py float64) {
	if p, ok := s.sketchClickPoint(px, py); ok {
		t.corners = append(t.corners, p)
	}
}

// CanCommit is true once two opposite corners are placed.
func (t *RectangleTool) CanCommit() bool { return len(t.corners) == 2 }

// PendingReferencePoint returns the first corner so the dynamic-input HUD shows the opposite
// corner's Length/Angle (#790); false before the first click.
func (t *RectangleTool) PendingReferencePoint() (math.Point2, bool) {
	if len(t.corners) == 0 {
		return math.Point2{}, false
	}
	return t.corners[len(t.corners)-1], true
}

// Commit adds the rigid rectangle: four lines over shared corners, squared by a horizontal and
// a vertical constraint per edge so that dragging a corner cannot shear it. Any in-place input
// field the user typed into also becomes a driving dimension (#2014).
func (t *RectangleTool) Commit(s *Session) error {
	if s.activeSketch == nil {
		return errors.New("rectangle: no active sketch")
	}
	return s.commitRecipe(sketch.RectangleRecipe(t.corners[0], t.corners[1]))
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

// LineTool draws lines between clicked points. By default (the viewport) it is a single
// two-point line that auto-commits on the second click. Started from the command line it
// runs continuous (EnableContinuous): point after point becomes a connected chain with
// [Close/Undo] options, ended by Enter or Close — AutoCAD's LINE (M26 F02 follow-up).
type LineTool struct {
	dialogTool
	points     []math.Point2
	continuous bool // command-line polyline mode: chain segments until Enter/Close
	closed     bool // a Close keyword was given: connect the last point back to the first
}

// NewLineTool returns a two-point line tool.
func NewLineTool() *LineTool { return &LineTool{} }

func (t *LineTool) Name() string { return "Line" }

// EnableContinuous switches the tool to AutoCAD-style continuous-polyline mode (the command
// line calls this when it starts a LINE). The viewport path leaves it off.
func (t *LineTool) EnableContinuous() { t.continuous = true }

// ClickAt records a line endpoint from a clicked pixel (snapped to points/grid).
func (t *LineTool) ClickAt(s *Session, px, py float64) {
	if p, ok := s.sketchClickPoint(px, py); ok {
		t.points = append(t.points, p)
	}
}

// CanCommit is true once enough endpoints are placed: two for a single line, or at least two
// for a continuous chain.
func (t *LineTool) CanCommit() bool {
	if t.continuous {
		return len(t.points) >= 2
	}
	return len(t.points) == 2
}

// Commit adds the line(s) to the active sketch and runs constraint inference on each (M06-F10,
// #625): a near-axis line picks up its horizontal/vertical (or parallel/perpendicular, per the
// session preference) automatically. In continuous mode it connects every consecutive point
// and, when Close was given, the last point back to the first.
func (t *LineTool) Commit(s *Session) error {
	if s.activeSketch == nil {
		return errNoSketch("line")
	}
	sk := s.activeSketch
	prev := sk.Points().Add(t.points[0])
	first := prev
	for i := 1; i < len(t.points); i++ {
		cur := sk.Points().Add(t.points[i])
		sk.ApplyLineInference(sk.Lines().Add(prev, cur), s.SketchInferenceOptions())
		prev = cur
	}
	if t.closed && len(t.points) >= 3 {
		sk.ApplyLineInference(sk.Lines().Add(prev, first), s.SketchInferenceOptions())
	}
	return nil
}

// Cancel discards the in-progress line.
func (t *LineTool) Cancel(*Session) { t.points = nil }

// PendingReferencePoint returns the last placed endpoint so the dynamic-input HUD shows the
// next segment's Length/Angle (#790); false before the first click.
func (t *LineTool) PendingReferencePoint() (math.Point2, bool) {
	if len(t.points) == 0 {
		return math.Point2{}, false
	}
	return t.points[len(t.points)-1], true
}

// AutoCommits creates the line on the second click in single-line mode; a continuous chain
// finishes on Enter/Close instead, so it does not auto-commit.
func (t *LineTool) AutoCommits() bool { return !t.continuous }

// Prompt guides the user through placing the endpoints (Inventor's status-bar prompts).
func (t *LineTool) Prompt(*Session) string {
	if t.continuous {
		if len(t.points) == 0 {
			return "Specify first point"
		}
		return "Specify next point"
	}
	switch len(t.points) {
	case 0:
		return "Click the line start point"
	case 1:
		return "Click the line end point"
	default:
		return "Click OK to create the line"
	}
}

// CommandOptions offers AutoCAD's chain options once a continuous line is under way: Undo
// after the first point, Close once a closing segment is possible.
func (t *LineTool) CommandOptions(*Session) []string {
	if !t.continuous || len(t.points) == 0 {
		return nil
	}
	if len(t.points) >= 2 {
		return []string{"Close", "Undo"}
	}
	return []string{"Undo"}
}

// SubmitToken adds a typed endpoint, or applies a Close/Undo option in continuous mode. The
// engine has already resolved any relative/polar input to an absolute sketch-plane coordinate.
func (t *LineTool) SubmitToken(_ *Session, tok CommandToken) error {
	switch tok.Kind {
	case CoordToken:
		t.points = append(t.points, math.P2(tok.Coord.X, tok.Coord.Y))
		return nil
	case KeywordToken:
		return t.applyOption(tok.Keyword)
	default:
		return errors.New("line: expected a point coordinate")
	}
}

// applyOption handles the Close/Undo chain options.
func (t *LineTool) applyOption(keyword string) error {
	switch keyword {
	case "Close":
		t.closed = true
	case "Undo":
		if len(t.points) > 0 {
			t.points = t.points[:len(t.points)-1]
		}
	default:
		return errors.New("line: unknown option " + keyword)
	}
	return nil
}

// FinishAfterToken tells the command-line engine to commit once a closing segment is set, so
// typing Close ends the polyline (M26 F02 follow-up). Otherwise the chain continues until Enter.
func (t *LineTool) FinishAfterToken() bool { return t.closed }

// SubmitToken adds a typed corner to the rectangle (M26).
func (t *RectangleTool) SubmitToken(_ *Session, tok CommandToken) error {
	if tok.Kind != CoordToken {
		return errors.New("rectangle: expected a corner coordinate")
	}
	t.corners = append(t.corners, math.P2(tok.Coord.X, tok.Coord.Y))
	return nil
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
