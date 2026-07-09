// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// The typed entity collections are factories bound to a sketch: an Add* call
// creates the entity (minting its shared points), registers it with the sketch, and
// returns it. They mirror COM's Lines/Arcs/… Add overloads.

// entityList is the shared storage core the typed factory collections embed, so each
// collection carries only its Add* factories (#1656). Item is deliberately UNGUARDED
// (panics out of range) — these are internal collections indexed by trusted callers;
// the nil-guarded shape belongs to the contract-facing views (G7, #1655).
type entityList[T comparable] struct{ items []T }

func (l *entityList[T]) Count() int   { return len(l.items) }
func (l *entityList[T]) Item(i int) T { return l.items[i] }
func (l *entityList[T]) append(x T)   { l.items = append(l.items, x) }
func (l *entityList[T]) remove(x T)   { l.items = removeItem(l.items, x) }

// removeItem drops the first occurrence of x from xs (used by the collections' remove).
func removeItem[T comparable](xs []T, x T) []T {
	for i, v := range xs {
		if v == x {
			return append(xs[:i], xs[i+1:]...)
		}
	}
	return xs
}

// Lines creates and tracks line segments.
type Lines struct {
	s *Sketch
	entityList[*Line]
}

// AddByTwoPoints creates a line between two new endpoints at the given positions.
func (c *Lines) AddByTwoPoints(a, b math.Point2) *Line {
	return c.Add(c.s.newPoint(a), c.s.newPoint(b))
}

// Add creates a line between two existing points; sharing a point with another
// entity makes the two coincident structurally.
func (c *Lines) Add(a, b *Point) *Line {
	l := &Line{entityBase: newEntity(), A: a, B: b}
	c.s.add(l)
	c.append(l)
	return l
}

// Centerlines returns the sketch's centerline lines (axes for revolve/mirror/symmetry).
func (s *Sketch) Centerlines() []*Line {
	var out []*Line
	for _, l := range s.lines.items {
		if l.IsCenterline() {
			out = append(out, l)
		}
	}
	return out
}

// Circles creates and tracks circles.
type Circles struct {
	s *Sketch
	entityList[*Circle]
}

// AddByCenterRadius creates a circle from a new center point and a radius.
func (c *Circles) AddByCenterRadius(center math.Point2, radius math.Scalar) *Circle {
	return c.Add(c.s.newPoint(center), radius)
}

// Add creates a circle around an existing center point; sharing the center with
// another entity makes them concentric structurally. Used by restore to rebuild a
// circle that shares points with other geometry.
func (c *Circles) Add(center *Point, radius math.Scalar) *Circle {
	circ := &Circle{entityBase: newEntity(), Center: center, Radius: radius}
	c.s.add(circ)
	c.append(circ)
	return circ
}

// Arcs creates and tracks arcs.
type Arcs struct {
	s *Sketch
	entityList[*Arc]
}

// AddByCenterStartEnd creates an arc from a center and two endpoints.
func (c *Arcs) AddByCenterStartEnd(center, start, end math.Point2, ccw bool) *Arc {
	return c.Add(c.s.newPoint(center), c.s.newPoint(start), c.s.newPoint(end), ccw)
}

// Add creates an arc from existing center/start/end points; sharing endpoints with
// other entities makes them coincident structurally. Used by restore.
func (c *Arcs) Add(center, start, end *Point, ccw bool) *Arc {
	a := &Arc{
		entityBase:       newEntity(),
		Center:           center,
		Start:            start,
		End:              end,
		CounterClockwise: ccw,
	}
	a.circularity = newArcCircularity(a) // keep End on the circle (#1419)
	c.s.add(a)
	c.append(a)
	return a
}

// Ellipses creates and tracks ellipses.
type Ellipses struct {
	s *Sketch
	entityList[*Ellipse]
}

// Add creates an ellipse from a center, major-axis direction, and the two radii.
func (c *Ellipses) Add(center math.Point2, majorAxis math.Vector2, majorR, minorR math.Scalar) *Ellipse {
	return c.AddWithCenter(c.s.newPoint(center), majorAxis, majorR, minorR)
}

// AddWithCenter creates an ellipse around an existing center point. Used by restore.
func (c *Ellipses) AddWithCenter(center *Point, majorAxis math.Vector2, majorR, minorR math.Scalar) *Ellipse {
	e := &Ellipse{
		entityBase:  newEntity(),
		Center:      center,
		MajorAxis:   majorAxis,
		MajorRadius: majorR,
		MinorRadius: minorR,
	}
	c.s.add(e)
	c.append(e)
	return e
}

// EllipticalArcs creates and tracks elliptical arcs.
type EllipticalArcs struct {
	s *Sketch
	entityList[*EllipticalArc]
}

// Add creates an elliptical arc from a center, major-axis direction, the two radii, and the TRUE
// geometric start/end angles (radians, Inventor's convention — the angle at the centre from the
// major axis to the ray through each endpoint, #1829). They are converted to the internal parametric
// (eccentric-anomaly) angles the arc stores. An importer that already supplies parametric angles
// (DXF/DWG) uses [EllipticalArcs.AddParametric] instead.
func (c *EllipticalArcs) Add(center math.Point2, majorAxis math.Vector2, majorR, minorR, startTrue, endTrue math.Scalar) *EllipticalArc {
	aStart, aEnd := paramArcFromTrue(float64(startTrue), float64(endTrue), float64(majorR), float64(minorR))
	return c.AddWithCenter(c.s.newPoint(center), majorAxis, majorR, minorR, math.Scalar(aStart), math.Scalar(aEnd))
}

// AddParametric creates an elliptical arc from PARAMETRIC (eccentric-anomaly) start/end angles — the
// arc's verbatim internal representation, no true-angle conversion. For callers that already hold the
// parametric angle: DXF/DWG ELLIPSE start/end params are eccentric-anomaly, so importing them through
// the true-angle [EllipticalArcs.Add] would mis-place the arc.
func (c *EllipticalArcs) AddParametric(center math.Point2, majorAxis math.Vector2, majorR, minorR, start, end math.Scalar) *EllipticalArc {
	return c.AddWithCenter(c.s.newPoint(center), majorAxis, majorR, minorR, start, end)
}

// AddWithCenter creates an elliptical arc around an existing center point. Used by restore.
func (c *EllipticalArcs) AddWithCenter(center *Point, majorAxis math.Vector2, majorR, minorR, start, end math.Scalar) *EllipticalArc {
	e := &EllipticalArc{
		entityBase:  newEntity(),
		Center:      center,
		MajorAxis:   majorAxis,
		MajorRadius: majorR,
		MinorRadius: minorR,
		StartAngle:  start,
		EndAngle:    end,
	}
	c.s.add(e)
	c.append(e)
	return e
}

// Splines creates and tracks splines.
type Splines struct {
	s *Sketch
	entityList[*Spline]
}

// AddByPoints creates a fit-point spline through the given positions.
func (c *Splines) AddByPoints(pts []math.Point2, closed bool) *Spline {
	return c.add(pts, closed, true)
}

// AddByControlPoints creates a control-point spline.
func (c *Splines) AddByControlPoints(pts []math.Point2, closed bool) *Spline {
	return c.add(pts, closed, false)
}

func (c *Splines) add(pts []math.Point2, closed, fit bool) *Spline {
	points := make([]*Point, len(pts))
	for i, p := range pts {
		points[i] = c.s.newPoint(p)
	}
	return c.AddWithPoints(points, closed, fit)
}

// AddWithPoints creates a spline through existing points. Used by restore to rebuild
// a spline that may share endpoints with adjacent geometry.
func (c *Splines) AddWithPoints(points []*Point, closed, fit bool) *Spline {
	sp := &Spline{entityBase: newEntity(), Points: points, Closed: closed, fit: fit}
	c.s.add(sp)
	c.append(sp)
	return sp
}

// Points creates and tracks standalone sketch points.
type Points struct {
	s *Sketch
	entityList[*Point]
}

// Add creates a standalone point at pos. Unlike curve endpoints, it appears in the
// sketch's entity list.
func (c *Points) Add(pos math.Point2) *Point {
	p := c.s.newPoint(pos)
	c.s.add(p)
	c.append(p)
	return p
}
