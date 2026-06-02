// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/health"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// WorkAxis is a parametric datum line: an origin and a unit direction.
type WorkAxis struct {
	id       ID
	name     string
	evaluate func() (math.Point3, math.UnitVector3, error)
	origin   math.Point3
	dir      math.UnitVector3
	health   health.Health
}

func (w *WorkAxis) ID() ID                      { return w.id }
func (w *WorkAxis) Name() string                { return w.name }
func (w *WorkAxis) Health() health.Health       { return w.health }
func (w *WorkAxis) Origin() math.Point3         { return w.origin }
func (w *WorkAxis) Direction() math.UnitVector3 { return w.dir }

// Recompute re-derives the axis, going sick on a degenerate definition.
func (w *WorkAxis) Recompute() {
	o, d, err := w.evaluate()
	if err != nil {
		w.health = health.Sicken("work axis: " + err.Error())
		return
	}
	w.origin, w.dir, w.health = o, d, health.Healthy
}

// WorkAxes is the collection of datum axes.
type WorkAxes struct {
	items []*WorkAxis
	byID  map[ID]*WorkAxis
}

// NewWorkAxes returns an empty collection.
func NewWorkAxes() *WorkAxes { return &WorkAxes{byID: map[ID]*WorkAxis{}} }

// AddByTwoPoints creates an axis through two (parametric) points.
func (c *WorkAxes) AddByTwoPoints(a, b func() math.Point3) *WorkAxis {
	eval := func() (math.Point3, math.UnitVector3, error) {
		d, err := math.UnitVector3FromVector(a().VectorTo(b()))
		return a(), d, err
	}
	return c.add(eval)
}

// AddByPlaneIntersection creates the axis where two planes meet.
func (c *WorkAxes) AddByPlaneIntersection(p1, p2 sketch.Plane) *WorkAxis {
	eval := func() (math.Point3, math.UnitVector3, error) { return planeIntersectionLine(p1, p2) }
	return c.add(eval)
}

func (c *WorkAxes) add(eval func() (math.Point3, math.UnitVector3, error)) *WorkAxis {
	w := &WorkAxis{id: nextID(), name: "WorkAxis", evaluate: eval}
	w.Recompute()
	c.items = append(c.items, w)
	c.byID[w.id] = w
	return w
}

// Count/Item index the collection.
func (c *WorkAxes) Count() int           { return len(c.items) }
func (c *WorkAxes) Item(i int) *WorkAxis { return c.items[i] }

// WorkPoint is a parametric datum point.
type WorkPoint struct {
	id       ID
	name     string
	evaluate func() (math.Point3, error)
	point    math.Point3
	health   health.Health
}

func (w *WorkPoint) ID() ID                { return w.id }
func (w *WorkPoint) Name() string          { return w.name }
func (w *WorkPoint) Health() health.Health { return w.health }
func (w *WorkPoint) Point() math.Point3    { return w.point }

// Recompute re-derives the point.
func (w *WorkPoint) Recompute() {
	p, err := w.evaluate()
	if err != nil {
		w.health = health.Sicken("work point: " + err.Error())
		return
	}
	w.point, w.health = p, health.Healthy
}

// WorkPoints is the collection of datum points.
type WorkPoints struct {
	items []*WorkPoint
	byID  map[ID]*WorkPoint
}

// NewWorkPoints returns an empty collection.
func NewWorkPoints() *WorkPoints { return &WorkPoints{byID: map[ID]*WorkPoint{}} }

// AddByPoint creates a datum point at a (parametric) position.
func (c *WorkPoints) AddByPoint(at func() math.Point3) *WorkPoint {
	return c.add(func() (math.Point3, error) { return at(), nil })
}

// AddByPlaneAndAxisIntersection creates the point where an axis pierces a plane.
func (c *WorkPoints) AddByPlaneAndAxisIntersection(plane sketch.Plane, axis *WorkAxis) *WorkPoint {
	eval := func() (math.Point3, error) {
		n := plane.Normal().AsVector()
		denom := axis.dir.AsVector().Dot(n)
		if math.IsNearZero(denom, math.DefaultTolerance) {
			return math.Point3{}, errors.New("axis is parallel to the plane")
		}
		t := axis.origin.VectorTo(plane.Origin()).Dot(n) / denom
		return axis.origin.TranslateBy(axis.dir.AsVector().Scale(t)), nil
	}
	return c.add(eval)
}

func (c *WorkPoints) add(eval func() (math.Point3, error)) *WorkPoint {
	w := &WorkPoint{id: nextID(), name: "WorkPoint", evaluate: eval}
	w.Recompute()
	c.items = append(c.items, w)
	c.byID[w.id] = w
	return w
}

// Count/Item index the collection.
func (c *WorkPoints) Count() int            { return len(c.items) }
func (c *WorkPoints) Item(i int) *WorkPoint { return c.items[i] }

// planeIntersectionLine returns a point on, and the direction of, the line where
// two planes meet (error if they are parallel).
func planeIntersectionLine(p1, p2 sketch.Plane) (math.Point3, math.UnitVector3, error) {
	n1, n2 := p1.Normal().AsVector(), p2.Normal().AsVector()
	cross := n1.Cross(n2)
	dir, err := math.UnitVector3FromVector(cross)
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, errors.New("planes are parallel")
	}
	c1 := n1.Dot(p1.Origin().AsVector())
	c2 := n2.Dot(p2.Origin().AsVector())
	numer := n2.Cross(cross).Scale(c1).Add(cross.Cross(n1).Scale(c2))
	return numer.Scale(1 / cross.LengthSquared()).AsPoint(), dir, nil
}
