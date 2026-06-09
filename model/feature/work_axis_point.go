// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"oblikovati.org/math"
	"oblikovati.org/model/health"
	"oblikovati.org/model/seq"
	"oblikovati.org/model/sketch"
)

// axisDefinition computes a work axis (origin + unit direction) from its references.
type axisDefinition interface {
	kindName() string
	refs() []WorkRef
	eval(r workResolver) (math.Point3, math.UnitVector3, error)
}

// fixedAxisDef is a grounded axis (an origin axis): fixed geometry, no references.
type fixedAxisDef struct {
	origin math.Point3
	dir    math.UnitVector3
}

func (d fixedAxisDef) kindName() string { return "fixed" }
func (d fixedAxisDef) refs() []WorkRef  { return nil }
func (d fixedAxisDef) eval(workResolver) (math.Point3, math.UnitVector3, error) {
	return d.origin, d.dir, nil
}

// twoPointsAxisDef is the axis through two referenced points.
type twoPointsAxisDef struct{ a, b WorkRef }

func (d twoPointsAxisDef) kindName() string { return "two-points" }
func (d twoPointsAxisDef) refs() []WorkRef  { return []WorkRef{d.a, d.b} }
func (d twoPointsAxisDef) eval(r workResolver) (math.Point3, math.UnitVector3, error) {
	a, err := r.point(d.a)
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, err
	}
	b, err := r.point(d.b)
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, err
	}
	dir, err := math.UnitVector3FromVector(a.VectorTo(b))
	return a, dir, err
}

// planeIntersectionAxisDef is the axis where two referenced planes meet.
type planeIntersectionAxisDef struct{ p1, p2 WorkRef }

func (d planeIntersectionAxisDef) kindName() string { return "plane-intersection" }
func (d planeIntersectionAxisDef) refs() []WorkRef  { return []WorkRef{d.p1, d.p2} }
func (d planeIntersectionAxisDef) eval(r workResolver) (math.Point3, math.UnitVector3, error) {
	p1, err := r.plane(d.p1)
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, err
	}
	p2, err := r.plane(d.p2)
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, err
	}
	return planeIntersectionLine(p1, p2)
}

// WorkAxis is a datum line: an origin and a unit direction.
type WorkAxis struct {
	id               ID
	key              WorkRef
	name             string
	def              axisDefinition
	origin           math.Point3
	dir              math.UnitVector3
	health           health.Health
	visible          bool
	coordinateSystem bool
	grounded         bool
	seq              uint64 // global creation stamp (0 for the origin frame); see model/seq
}

func (w *WorkAxis) ID() ID                          { return w.id }
func (w *WorkAxis) Seq() uint64                     { return w.seq }
func (w *WorkAxis) Key() WorkRef                    { return w.key }
func (w *WorkAxis) Name() string                    { return w.name }
func (w *WorkAxis) Health() health.Health           { return w.health }
func (w *WorkAxis) Origin() math.Point3             { return w.origin }
func (w *WorkAxis) Direction() math.UnitVector3     { return w.dir }
func (w *WorkAxis) IsCoordinateSystemElement() bool { return w.coordinateSystem }
func (w *WorkAxis) Grounded() bool                  { return w.grounded }

// Visible reports whether the datum axis is drawn in the viewport.
func (w *WorkAxis) Visible() bool { return w.visible }

// SetVisible toggles the datum axis viewport visibility.
func (w *WorkAxis) SetVisible(v bool) { w.visible = v }

// recompute re-derives the axis, going sick on a degenerate definition.
func (w *WorkAxis) recompute(r workResolver) {
	o, d, err := w.def.eval(r)
	if err != nil {
		w.health = health.Sicken("work axis: " + err.Error())
		return
	}
	w.origin, w.dir, w.health = o, d, health.Healthy
}

// WorkAxes is the part's collection of datum axes (origin first, then user).
type WorkAxes struct {
	g     *WorkGeometry
	items []*WorkAxis
	byID  map[ID]*WorkAxis
	byKey map[WorkRef]*WorkAxis
}

func newWorkAxes(g *WorkGeometry) *WorkAxes {
	return &WorkAxes{g: g, byID: map[ID]*WorkAxis{}, byKey: map[WorkRef]*WorkAxis{}}
}

func (c *WorkAxes) addOrigin(key WorkRef, name string, origin math.Point3, dir math.UnitVector3) {
	w := &WorkAxis{
		id: nextID(), key: key, name: name, def: fixedAxisDef{origin: origin, dir: dir},
		coordinateSystem: true, grounded: true,
	}
	c.track(w)
}

// AddByTwoPoints creates a user axis through two referenced points.
func (c *WorkAxes) AddByTwoPoints(a, b WorkRef) *WorkAxis {
	return c.addUser(twoPointsAxisDef{a: a, b: b})
}

// AddByPlaneIntersection creates the axis where two referenced planes meet.
func (c *WorkAxes) AddByPlaneIntersection(p1, p2 WorkRef) *WorkAxis {
	return c.addUser(planeIntersectionAxisDef{p1: p1, p2: p2})
}

func (c *WorkAxes) addUser(def axisDefinition) *WorkAxis {
	w := &WorkAxis{id: nextID(), key: userRef("axis", len(c.items)), name: "WorkAxis", def: def, visible: true, seq: seq.Next()}
	c.track(w)
	c.g.recordUser("axis", len(c.items)-1)
	return w
}

func (c *WorkAxes) track(w *WorkAxis) {
	w.recompute(c.g)
	c.items = append(c.items, w)
	c.byID[w.id] = w
	c.byKey[w.key] = w
}

// Count/Item index the collection.
func (c *WorkAxes) Count() int           { return len(c.items) }
func (c *WorkAxes) Item(i int) *WorkAxis { return c.items[i] }

// pointDefinition computes a work point's position from its references.
type pointDefinition interface {
	kindName() string
	refs() []WorkRef
	eval(r workResolver) (math.Point3, error)
}

// fixedPointDef is a grounded point (the origin center): fixed geometry, no references.
type fixedPointDef struct{ point math.Point3 }

func (d fixedPointDef) kindName() string                       { return "fixed" }
func (d fixedPointDef) refs() []WorkRef                        { return nil }
func (d fixedPointDef) eval(workResolver) (math.Point3, error) { return d.point, nil }

// positionPointDef is a point at a (parametric) absolute position.
type positionPointDef struct{ at func() math.Point3 }

func (d positionPointDef) kindName() string                       { return "position" }
func (d positionPointDef) refs() []WorkRef                        { return nil }
func (d positionPointDef) eval(workResolver) (math.Point3, error) { return d.at(), nil }

// planeAxisPointDef is the point where a referenced axis pierces a referenced plane.
type planeAxisPointDef struct {
	plane WorkRef
	axis  WorkRef
}

func (d planeAxisPointDef) kindName() string { return "plane-axis-intersection" }
func (d planeAxisPointDef) refs() []WorkRef  { return []WorkRef{d.plane, d.axis} }
func (d planeAxisPointDef) eval(r workResolver) (math.Point3, error) {
	plane, err := r.plane(d.plane)
	if err != nil {
		return math.Point3{}, err
	}
	axis, err := r.axis(d.axis)
	if err != nil {
		return math.Point3{}, err
	}
	n := plane.Normal().AsVector()
	denom := axis.dir.AsVector().Dot(n)
	if math.IsNearZero(denom, math.DefaultTolerance) {
		return math.Point3{}, errors.New("axis is parallel to the plane")
	}
	t := axis.origin.VectorTo(plane.Origin()).Dot(n) / denom
	return axis.origin.TranslateBy(axis.dir.AsVector().Scale(t)), nil
}

// WorkPoint is a datum point.
type WorkPoint struct {
	id               ID
	key              WorkRef
	name             string
	def              pointDefinition
	point            math.Point3
	health           health.Health
	coordinateSystem bool
	grounded         bool
	seq              uint64 // global creation stamp (0 for the origin frame); see model/seq
}

func (w *WorkPoint) ID() ID                          { return w.id }
func (w *WorkPoint) Seq() uint64                     { return w.seq }
func (w *WorkPoint) Key() WorkRef                    { return w.key }
func (w *WorkPoint) Name() string                    { return w.name }
func (w *WorkPoint) Health() health.Health           { return w.health }
func (w *WorkPoint) Point() math.Point3              { return w.point }
func (w *WorkPoint) IsCoordinateSystemElement() bool { return w.coordinateSystem }
func (w *WorkPoint) Grounded() bool                  { return w.grounded }

// recompute re-derives the point.
func (w *WorkPoint) recompute(r workResolver) {
	p, err := w.def.eval(r)
	if err != nil {
		w.health = health.Sicken("work point: " + err.Error())
		return
	}
	w.point, w.health = p, health.Healthy
}

// WorkPoints is the part's collection of datum points (origin first, then user).
type WorkPoints struct {
	g     *WorkGeometry
	items []*WorkPoint
	byID  map[ID]*WorkPoint
	byKey map[WorkRef]*WorkPoint
}

func newWorkPoints(g *WorkGeometry) *WorkPoints {
	return &WorkPoints{g: g, byID: map[ID]*WorkPoint{}, byKey: map[WorkRef]*WorkPoint{}}
}

func (c *WorkPoints) addOrigin(key WorkRef, name string, at math.Point3) {
	w := &WorkPoint{
		id: nextID(), key: key, name: name, def: fixedPointDef{point: at},
		coordinateSystem: true, grounded: true,
	}
	c.track(w)
}

// AddByPosition creates a user point at a (parametric) absolute position.
func (c *WorkPoints) AddByPosition(at func() math.Point3) *WorkPoint {
	return c.addUser(positionPointDef{at: at})
}

// AddByPlaneAndAxisIntersection creates the point where a referenced axis pierces a
// referenced plane.
func (c *WorkPoints) AddByPlaneAndAxisIntersection(plane, axis WorkRef) *WorkPoint {
	return c.addUser(planeAxisPointDef{plane: plane, axis: axis})
}

func (c *WorkPoints) addUser(def pointDefinition) *WorkPoint {
	w := &WorkPoint{id: nextID(), key: userRef("point", len(c.items)), name: "WorkPoint", def: def, seq: seq.Next()}
	c.track(w)
	c.g.recordUser("point", len(c.items)-1)
	return w
}

func (c *WorkPoints) track(w *WorkPoint) {
	w.recompute(c.g)
	c.items = append(c.items, w)
	c.byID[w.id] = w
	c.byKey[w.key] = w
}

// Count/Item index the collection.
func (c *WorkPoints) Count() int            { return len(c.items) }
func (c *WorkPoints) Item(i int) *WorkPoint { return c.items[i] }

// planeIntersectionLine returns a point on, and the direction of, the line where two
// planes meet (error if they are parallel).
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
