// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/sketch"
)

// The variant geometry tools — the entries of the Sketch tab's split-button dropdowns
// (Inventor's variant flyouts). Each is a sibling of a Create-panel head tool that draws
// the same kind of geometry a different way: Rectangle (Three Point, Two Point Center),
// Circle (Three Point), Arc (Center Point), Slot (Center Point Arc, Three Point Arc),
// Spline (Control Vertex). They reuse the model/sketch composite builders, which already
// produce the underlying lines/arcs; these are the interaction shells, tested via the
// app's headless click path like their heads.

// ThreePointRectangleTool draws a rectangle from three corners: a base edge (first two
// clicks) then the perpendicular extent (third click), so it is not axis-aligned.
type ThreePointRectangleTool struct {
	dialogTool
	collectClicks
}

// NewThreePointRectangleTool returns a three-point rectangle tool.
func NewThreePointRectangleTool() *ThreePointRectangleTool { return &ThreePointRectangleTool{} }

func (t *ThreePointRectangleTool) Name() string      { return "Three Point Rectangle" }
func (t *ThreePointRectangleTool) Cancel(*Session)   { t.reset() }
func (t *ThreePointRectangleTool) CanCommit() bool   { return len(t.pts) == 3 }
func (t *ThreePointRectangleTool) AutoCommits() bool { return true }

// Commit adds the rigid rotated rectangle from the three clicked corners: three perpendicular
// constraints round the loop keep it square under a later drag (#2014).
func (t *ThreePointRectangleTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errNoSketch("three point rectangle")
	}
	return s.commitRecipe(sketch.ThreePointRectangleRecipe(t.pts[0], t.pts[1], t.pts[2]))
}

// Prompt guides the base edge then the width (Inventor's status-bar prompts).
func (t *ThreePointRectangleTool) Prompt(*Session) string {
	switch len(t.pts) {
	case 0:
		return "Click the first corner of the rectangle"
	case 1:
		return "Click the end of the first edge"
	case 2:
		return "Click to set the rectangle width"
	default:
		return "Click OK to create the rectangle"
	}
}

// CenterRectangleTool draws an axis-aligned rectangle from its center and a corner, so the
// center stays put as the rectangle grows (Inventor's Two Point Center Rectangle).
type CenterRectangleTool struct {
	dialogTool
	collectClicks
}

// NewCenterRectangleTool returns a center rectangle tool.
func NewCenterRectangleTool() *CenterRectangleTool { return &CenterRectangleTool{} }

func (t *CenterRectangleTool) Name() string      { return "Two Point Center Rectangle" }
func (t *CenterRectangleTool) Cancel(*Session)   { t.reset() }
func (t *CenterRectangleTool) CanCommit() bool   { return len(t.pts) == 2 }
func (t *CenterRectangleTool) AutoCommits() bool { return true }

// Commit adds the rigid centre-out rectangle: squared by horizontal/vertical constraints, with
// the centre pinned as the midpoint of a construction diagonal (#2014).
func (t *CenterRectangleTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errNoSketch("center rectangle")
	}
	return s.commitRecipe(sketch.CenterRectangleRecipe(t.pts[0], t.pts[1]))
}

// Prompt guides the center then a corner.
func (t *CenterRectangleTool) Prompt(*Session) string {
	if len(t.pts) == 0 {
		return "Click the center of the rectangle"
	}
	return "Click a corner of the rectangle"
}

// ThreePointCircleTool draws the circle passing through three clicked points (Inventor's
// Three Point Circle), the dual of the center-radius head tool.
type ThreePointCircleTool struct {
	dialogTool
	collectClicks
}

// NewThreePointCircleTool returns a three-point circle tool.
func NewThreePointCircleTool() *ThreePointCircleTool { return &ThreePointCircleTool{} }

func (t *ThreePointCircleTool) Name() string      { return "Three Point Circle" }
func (t *ThreePointCircleTool) Cancel(*Session)   { t.reset() }
func (t *ThreePointCircleTool) CanCommit() bool   { return len(t.pts) == 3 }
func (t *ThreePointCircleTool) AutoCommits() bool { return true }

// Commit fits and adds the circle through the three clicked points.
func (t *ThreePointCircleTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errNoSketch("three point circle")
	}
	center, ok := circumcenter(t.pts[0], t.pts[1], t.pts[2])
	if !ok {
		return errors.New("three point circle: the three points are collinear")
	}
	return s.commitRecipe(sketch.CircleRecipe(center, center.DistanceTo(t.pts[0])))
}

// Prompt guides the three points the circle passes through.
func (t *ThreePointCircleTool) Prompt(*Session) string {
	switch len(t.pts) {
	case 0:
		return "Click the first point on the circle"
	case 1:
		return "Click the second point on the circle"
	case 2:
		return "Click the third point on the circle"
	default:
		return "Click OK to create the circle"
	}
}

// CenterPointArcTool draws an arc from its center, a start point (sets the radius) and an
// end point (sets the sweep), the dual of the three-point head arc tool.
type CenterPointArcTool struct {
	dialogTool
	collectClicks
}

// NewCenterPointArcTool returns a center-point arc tool.
func NewCenterPointArcTool() *CenterPointArcTool { return &CenterPointArcTool{} }

func (t *CenterPointArcTool) Name() string      { return "Center Point Arc" }
func (t *CenterPointArcTool) Cancel(*Session)   { t.reset() }
func (t *CenterPointArcTool) CanCommit() bool   { return len(t.pts) == 3 }
func (t *CenterPointArcTool) AutoCommits() bool { return true }

// Commit adds the arc; it sweeps counter-clockwise when the end lies CCW of the start
// about the center (the cross product of the two radii is positive).
func (t *CenterPointArcTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errNoSketch("center point arc")
	}
	center, start, end := t.pts[0], t.pts[1], t.pts[2]
	return s.commitRecipe(sketch.ArcRecipe(center, start, end, leftTurn(center, start, end)))
}

// Prompt guides the center, start and end of the arc.
func (t *CenterPointArcTool) Prompt(*Session) string {
	switch len(t.pts) {
	case 0:
		return "Click the arc center"
	case 1:
		return "Click the arc start point"
	case 2:
		return "Click the arc end point"
	default:
		return "Click OK to create the arc"
	}
}

// ControlVertexSplineTool draws a B-spline from its control polygon (the clicked points are
// control vertices, not points the curve passes through), the dual of the interpolation
// head spline tool. OK finishes the curve.
type ControlVertexSplineTool struct {
	dialogTool
	collectClicks
}

// NewControlVertexSplineTool returns a control-vertex spline tool.
func NewControlVertexSplineTool() *ControlVertexSplineTool { return &ControlVertexSplineTool{} }

func (t *ControlVertexSplineTool) Name() string    { return "Control Vertex Spline" }
func (t *ControlVertexSplineTool) Cancel(*Session) { t.reset() }
func (t *ControlVertexSplineTool) CanCommit() bool { return len(t.pts) >= 2 }

// Commit adds the spline from the clicked control polygon.
func (t *ControlVertexSplineTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errNoSketch("control vertex spline")
	}
	return s.commitRecipe(sketch.SplineRecipe(t.pts, false))
}

// Prompt guides successive control vertices.
func (t *ControlVertexSplineTool) Prompt(*Session) string {
	if len(t.pts) < 2 {
		return "Click control vertices for the spline"
	}
	return "Click more vertices, or OK to finish"
}

// Compile-time checks that the variant tools satisfy the interactive-tool contracts the
// session drives them through (the fixed-arity ones auto-commit; the spline finishes on OK).
var (
	_ Tool           = (*ThreePointRectangleTool)(nil)
	_ PlaneClickTool = (*ThreePointRectangleTool)(nil)
	_ AutoCommitTool = (*ThreePointRectangleTool)(nil)
	_ Tool           = (*ControlVertexSplineTool)(nil)
	_ PlaneClickTool = (*ControlVertexSplineTool)(nil)
)
