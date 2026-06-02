// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "github.com/Oblikovati/oblikovati/math"

// The typed entity collections are factories bound to a sketch: an Add* call
// creates the entity (minting its shared points), registers it with the sketch, and
// returns it. They mirror COM's Lines/Arcs/… Add overloads.

// Lines creates and tracks line segments.
type Lines struct {
	s     *Sketch
	items []*Line
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
	c.items = append(c.items, l)
	return l
}

// Count returns the number of lines; Item returns the i-th.
func (c *Lines) Count() int       { return len(c.items) }
func (c *Lines) Item(i int) *Line { return c.items[i] }

// Circles creates and tracks circles.
type Circles struct {
	s     *Sketch
	items []*Circle
}

// AddByCenterRadius creates a circle from a new center point and a radius.
func (c *Circles) AddByCenterRadius(center math.Point2, radius math.Scalar) *Circle {
	circ := &Circle{entityBase: newEntity(), Center: c.s.newPoint(center), Radius: radius}
	c.s.add(circ)
	c.items = append(c.items, circ)
	return circ
}

// Count returns the number of circles; Item returns the i-th.
func (c *Circles) Count() int         { return len(c.items) }
func (c *Circles) Item(i int) *Circle { return c.items[i] }

// Arcs creates and tracks arcs.
type Arcs struct {
	s     *Sketch
	items []*Arc
}

// AddByCenterStartEnd creates an arc from a center and two endpoints.
func (c *Arcs) AddByCenterStartEnd(center, start, end math.Point2, ccw bool) *Arc {
	a := &Arc{
		entityBase:       newEntity(),
		Center:           c.s.newPoint(center),
		Start:            c.s.newPoint(start),
		End:              c.s.newPoint(end),
		CounterClockwise: ccw,
	}
	c.s.add(a)
	c.items = append(c.items, a)
	return a
}

// Count returns the number of arcs; Item returns the i-th.
func (c *Arcs) Count() int      { return len(c.items) }
func (c *Arcs) Item(i int) *Arc { return c.items[i] }

// Ellipses creates and tracks ellipses.
type Ellipses struct {
	s     *Sketch
	items []*Ellipse
}

// Add creates an ellipse from a center, major-axis direction, and the two radii.
func (c *Ellipses) Add(center math.Point2, majorAxis math.Vector2, majorR, minorR math.Scalar) *Ellipse {
	e := &Ellipse{
		entityBase:  newEntity(),
		Center:      c.s.newPoint(center),
		MajorAxis:   majorAxis,
		MajorRadius: majorR,
		MinorRadius: minorR,
	}
	c.s.add(e)
	c.items = append(c.items, e)
	return e
}

// Count returns the number of ellipses; Item returns the i-th.
func (c *Ellipses) Count() int          { return len(c.items) }
func (c *Ellipses) Item(i int) *Ellipse { return c.items[i] }

// Splines creates and tracks splines.
type Splines struct {
	s     *Sketch
	items []*Spline
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
	sp := &Spline{entityBase: newEntity(), Points: points, Closed: closed, fit: fit}
	c.s.add(sp)
	c.items = append(c.items, sp)
	return sp
}

// Count returns the number of splines; Item returns the i-th.
func (c *Splines) Count() int         { return len(c.items) }
func (c *Splines) Item(i int) *Spline { return c.items[i] }

// Points creates and tracks standalone sketch points.
type Points struct {
	s     *Sketch
	items []*Point
}

// Add creates a standalone point at pos. Unlike curve endpoints, it appears in the
// sketch's entity list.
func (c *Points) Add(pos math.Point2) *Point {
	p := c.s.newPoint(pos)
	c.s.add(p)
	c.items = append(c.items, p)
	return p
}

// Count returns the number of standalone points; Item returns the i-th.
func (c *Points) Count() int        { return len(c.items) }
func (c *Points) Item(i int) *Point { return c.items[i] }
