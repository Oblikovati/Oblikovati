// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// The sketch geometry tools — Inventor's Sketch tab "Create" panel. Each is an
// interactive [Tool] + [PlaneClickTool] driven by plane-point clicks, with a per-step
// [Prompted] message, committing real model/sketch entities. They all require an active
// sketch (the sketch environment must be open). New tools register a ribbon button with
// no UI code edited (see RegisterStandardCommands).

// errNoSketch is the commit error when a geometry tool runs with no active sketch.
func errNoSketch(tool string) error { return errors.New(tool + ": no active sketch") }

// collectClicks is the shared click buffer for the multi-point geometry tools.
type collectClicks struct{ pts []math.Point2 }

// ClickAt maps a pixel to the sketch plane (with snapping) and records it.
func (c *collectClicks) ClickAt(s *Session, px, py float64) {
	if p, ok := s.sketchClickPoint(px, py); ok {
		c.pts = append(c.pts, p)
	}
}

// Cancel discards the in-progress points.
func (c *collectClicks) reset() { c.pts = nil }

// CircleTool draws a circle from a clicked center and a radius point.
type CircleTool struct{ collectClicks }

// NewCircleTool returns a center-radius circle tool.
func NewCircleTool() *CircleTool { return &CircleTool{} }

func (t *CircleTool) Name() string              { return "Circle" }
func (t *CircleTool) Start(*Session)            {}
func (t *CircleTool) Pick(*Session, Selectable) {}
func (t *CircleTool) Cancel(*Session)           { t.reset() }
func (t *CircleTool) CanCommit() bool           { return len(t.pts) == 2 }
func (t *CircleTool) AutoCommits() bool         { return true }

// Commit adds a circle centered at the first click with radius to the second.
func (t *CircleTool) Commit(s *Session) error {
	if s.activeSketch == nil {
		return errNoSketch("circle")
	}
	s.activeSketch.Circles().AddByCenterRadius(t.pts[0], t.pts[0].DistanceTo(t.pts[1]))
	return nil
}

// Prompt guides center then radius.
func (t *CircleTool) Prompt(*Session) string {
	switch len(t.pts) {
	case 0:
		return "Click the circle center"
	case 1:
		return "Click to set the radius"
	default:
		return "Click OK to create the circle"
	}
}

// ArcTool draws a three-point arc (start, end, then a point the arc passes through).
type ArcTool struct{ collectClicks }

// NewArcTool returns a three-point arc tool.
func NewArcTool() *ArcTool { return &ArcTool{} }

func (t *ArcTool) Name() string              { return "Arc" }
func (t *ArcTool) Start(*Session)            {}
func (t *ArcTool) Pick(*Session, Selectable) {}
func (t *ArcTool) Cancel(*Session)           { t.reset() }
func (t *ArcTool) CanCommit() bool           { return len(t.pts) == 3 }
func (t *ArcTool) AutoCommits() bool         { return true }

// Commit fits the arc's center as the circumcenter of the three points; the sweep
// direction is the orientation of start→through→end so the arc passes through it.
func (t *ArcTool) Commit(s *Session) error {
	if s.activeSketch == nil {
		return errNoSketch("arc")
	}
	start, end, through := t.pts[0], t.pts[1], t.pts[2]
	center, ok := circumcenter(start, through, end)
	if !ok {
		return errors.New("arc: the three points are collinear")
	}
	ccw := leftTurn(start, through, end)
	s.activeSketch.Arcs().AddByCenterStartEnd(center, start, end, ccw)
	return nil
}

// Prompt guides start, end, then a point on the arc.
func (t *ArcTool) Prompt(*Session) string {
	switch len(t.pts) {
	case 0:
		return "Click the arc start point"
	case 1:
		return "Click the arc end point"
	case 2:
		return "Click a point on the arc"
	default:
		return "Click OK to create the arc"
	}
}

// PointTool places a sketch point at the click, then deactivates (one point per
// activation, like the other auto-commit tools).
type PointTool struct{ collectClicks }

// NewPointTool returns a point-placement tool.
func NewPointTool() *PointTool { return &PointTool{} }

func (t *PointTool) Name() string              { return "Point" }
func (t *PointTool) Start(*Session)            {}
func (t *PointTool) Pick(*Session, Selectable) {}
func (t *PointTool) Cancel(*Session)           { t.reset() }
func (t *PointTool) CanCommit() bool           { return len(t.pts) >= 1 }
func (t *PointTool) AutoCommits() bool         { return true }

// Commit adds every placed point to the sketch.
func (t *PointTool) Commit(s *Session) error {
	if s.activeSketch == nil {
		return errNoSketch("point")
	}
	for _, p := range t.pts {
		s.activeSketch.Points().Add(p)
	}
	return nil
}

// Prompt guides placing points.
func (t *PointTool) Prompt(*Session) string {
	if len(t.pts) == 0 {
		return "Click to place a point"
	}
	return "Click to place another point, or OK to finish"
}

// SplineTool draws an interpolated spline through clicked points (OK finishes).
type SplineTool struct {
	collectClicks
	withHandles bool
}

// NewSplineTool returns a fit-point spline tool.
func NewSplineTool() *SplineTool { return &SplineTool{} }

func (t *SplineTool) Name() string              { return "Spline" }
func (t *SplineTool) Start(*Session)            {}
func (t *SplineTool) Pick(*Session, Selectable) {}
func (t *SplineTool) Cancel(*Session)           { t.reset() }
func (t *SplineTool) CanCommit() bool           { return len(t.pts) >= 2 }

// SetActivateHandles makes Commit also activate the tangency handle on every
// placed fit point, ready for interactive shaping (M06-F11, #626).
func (t *SplineTool) SetActivateHandles(on bool) { t.withHandles = on }

// Commit fits an open spline through the placed points.
func (t *SplineTool) Commit(s *Session) error {
	if s.activeSketch == nil {
		return errNoSketch("spline")
	}
	sp := s.activeSketch.Splines().AddByPoints(append([]math.Point2(nil), t.pts...), false)
	if !t.withHandles {
		return nil
	}
	for i := range sp.Points {
		if _, err := s.activeSketch.SplineHandles().Activate(sp, i); err != nil {
			return err
		}
	}
	return nil
}

// Prompt guides placing fit points.
func (t *SplineTool) Prompt(*Session) string {
	if len(t.pts) < 2 {
		return "Click spline fit points (at least two)"
	}
	return "Click more fit points, or OK to finish the spline"
}

// EllipseTool draws an ellipse from a center, a major-axis endpoint, and a point
// setting the minor radius.
type EllipseTool struct{ collectClicks }

// NewEllipseTool returns a center-axis ellipse tool.
func NewEllipseTool() *EllipseTool { return &EllipseTool{} }

func (t *EllipseTool) Name() string              { return "Ellipse" }
func (t *EllipseTool) Start(*Session)            {}
func (t *EllipseTool) Pick(*Session, Selectable) {}
func (t *EllipseTool) Cancel(*Session)           { t.reset() }
func (t *EllipseTool) CanCommit() bool           { return len(t.pts) == 3 }
func (t *EllipseTool) AutoCommits() bool         { return true }

// Commit builds the ellipse: the second click sets the major direction and radius, the
// third its perpendicular distance from the major axis sets the minor radius.
func (t *EllipseTool) Commit(s *Session) error {
	if s.activeSketch == nil {
		return errNoSketch("ellipse")
	}
	center := t.pts[0]
	major := center.VectorTo(t.pts[1])
	majorR := major.Length()
	if majorR < math.DefaultTolerance {
		return errors.New("ellipse: major radius is zero")
	}
	minorR := perpDistance(t.pts[2], center, major)
	s.activeSketch.Ellipses().Add(center, major, majorR, minorR)
	return nil
}

// Prompt guides center, major axis, then minor radius.
func (t *EllipseTool) Prompt(*Session) string {
	switch len(t.pts) {
	case 0:
		return "Click the ellipse center"
	case 1:
		return "Click the major-axis endpoint"
	case 2:
		return "Click to set the minor radius"
	default:
		return "Click OK to create the ellipse"
	}
}

// PolygonTool draws a regular inscribed polygon from a center and a vertex; Sides sets
// the edge count (default 6).
type PolygonTool struct {
	collectClicks
	Sides int
}

// NewPolygonTool returns a regular-polygon tool with the given side count (min 3).
func NewPolygonTool(sides int) *PolygonTool {
	if sides < 3 {
		sides = 6
	}
	return &PolygonTool{Sides: sides}
}

func (t *PolygonTool) Name() string              { return "Polygon" }
func (t *PolygonTool) Start(*Session)            {}
func (t *PolygonTool) Pick(*Session, Selectable) {}
func (t *PolygonTool) Cancel(*Session)           { t.reset() }
func (t *PolygonTool) CanCommit() bool           { return len(t.pts) == 2 }
func (t *PolygonTool) AutoCommits() bool         { return true }

// Commit builds the Sides-gon inscribed in the circle through the vertex click,
// connecting consecutive vertices with lines sharing their corner points.
func (t *PolygonTool) Commit(s *Session) error {
	if s.activeSketch == nil {
		return errNoSketch("polygon")
	}
	verts := polygonVertices(t.pts[0], t.pts[1], t.Sides)
	sk := s.activeSketch
	pts := make([]*sketch.Point, len(verts))
	for i, v := range verts {
		pts[i] = sk.Points().Add(v)
	}
	for i := range pts {
		sk.Lines().Add(pts[i], pts[(i+1)%len(pts)])
	}
	return nil
}

// Prompt guides center then a vertex.
func (t *PolygonTool) Prompt(*Session) string {
	switch len(t.pts) {
	case 0:
		return "Click the polygon center"
	case 1:
		return "Click a vertex to set the size and rotation"
	default:
		return "Click OK to create the polygon"
	}
}

// polygonVertices returns the n vertices of a regular polygon centered at center with
// its first vertex at vertex.
func polygonVertices(center, vertex math.Point2, n int) []math.Point2 {
	r := center.DistanceTo(vertex)
	base := stdmath.Atan2(vertex.Y-center.Y, vertex.X-center.X)
	out := make([]math.Point2, n)
	for i := 0; i < n; i++ {
		a := base + 2*stdmath.Pi*float64(i)/float64(n)
		out[i] = math.P2(center.X+r*stdmath.Cos(a), center.Y+r*stdmath.Sin(a))
	}
	return out
}

// circumcenter returns the center of the circle through three points, false if they
// are collinear.
func circumcenter(a, b, c math.Point2) (math.Point2, bool) {
	d := 2 * (a.X*(b.Y-c.Y) + b.X*(c.Y-a.Y) + c.X*(a.Y-b.Y))
	if stdmath.Abs(d) < math.DefaultTolerance {
		return math.Point2{}, false
	}
	a2, b2, c2 := sq(a), sq(b), sq(c)
	ux := (a2*(b.Y-c.Y) + b2*(c.Y-a.Y) + c2*(a.Y-b.Y)) / d
	uy := (a2*(c.X-b.X) + b2*(a.X-c.X) + c2*(b.X-a.X)) / d
	return math.P2(ux, uy), true
}

func sq(p math.Point2) float64 { return p.X*p.X + p.Y*p.Y }

// leftTurn reports whether a→b→c turns counter-clockwise (positive signed area).
func leftTurn(a, b, c math.Point2) bool {
	return (b.X-a.X)*(c.Y-a.Y)-(b.Y-a.Y)*(c.X-a.X) > 0
}

// perpDistance returns the perpendicular distance of p from the line through origin
// along dir.
func perpDistance(p, origin math.Point2, dir math.Vector2) float64 {
	v := origin.VectorTo(p)
	dl := dir.Length()
	if dl < math.DefaultTolerance {
		return v.Length()
	}
	return stdmath.Abs(v.Cross(dir)) / dl
}
