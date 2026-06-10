// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// These are the work-plane definitions that build on other work features (planes,
// axes, points) resolved through the [workResolver] — the reference-model half of
// Inventor's WorkPlanes constructors. The surface-tangent half (built on a B-rep face)
// lives in work_plane_tangent.go. Every definition captures its references + scalar
// parameters so it round-trips and re-derives on recompute (work_plane.go owns the
// WorkPlane/WorkPlanes types and the offset/three-point definitions).

// fixedFramePlaneDef is a user plane fixed in space by an origin and two in-plane axes
// (Inventor's AddFixed) — absolute geometry, unlike the offset/three-point planes that
// float on their references. The origin is a closure so it can track a parameter.
type fixedFramePlaneDef struct {
	origin func() math.Point3
	x, y   math.UnitVector3
}

func (d *fixedFramePlaneDef) kindName() string { return "fixed-frame" }
func (d *fixedFramePlaneDef) refs() []WorkRef  { return nil }
func (d *fixedFramePlaneDef) eval(workResolver) (sketch.Plane, error) {
	return sketch.NewPlane(d.origin(), d.x, d.y)
}

// planeAndPointPlaneDef is a plane parallel to a base plane through a point (Inventor's
// AddByPlaneAndPoint): it inherits the base orientation and is repositioned at the point.
type planeAndPointPlaneDef struct {
	base  WorkRef
	point WorkRef
}

func (d *planeAndPointPlaneDef) kindName() string { return "plane-point" }
func (d *planeAndPointPlaneDef) refs() []WorkRef  { return []WorkRef{d.base, d.point} }
func (d *planeAndPointPlaneDef) eval(r workResolver) (sketch.Plane, error) {
	base, err := r.plane(d.base)
	if err != nil {
		return sketch.Plane{}, err
	}
	p, err := r.point(d.point)
	if err != nil {
		return sketch.Plane{}, err
	}
	return sketch.NewPlane(p, base.XAxis(), base.YAxis())
}

// twoPlanesPlaneDef bisects two planes (Inventor's AddByTwoPlanes): the dihedral-angle
// bisector through their intersection line, or the mid-plane when they are parallel.
type twoPlanesPlaneDef struct{ p1, p2 WorkRef }

func (d *twoPlanesPlaneDef) kindName() string { return "two-planes" }
func (d *twoPlanesPlaneDef) refs() []WorkRef  { return []WorkRef{d.p1, d.p2} }
func (d *twoPlanesPlaneDef) eval(r workResolver) (sketch.Plane, error) {
	p1, err := r.plane(d.p1)
	if err != nil {
		return sketch.Plane{}, err
	}
	p2, err := r.plane(d.p2)
	if err != nil {
		return sketch.Plane{}, err
	}
	return bisectingPlane(p1, p2)
}

// linePlaneAnglePlaneDef passes through a line at a given angle from a plane, swung
// about the line (Inventor's AddByLinePlaneAndAngle). The angle is a closure so it can
// track a parameter (the datum re-angles when the parameter changes).
type linePlaneAnglePlaneDef struct {
	line  WorkRef
	base  WorkRef
	angle func() float64
}

func (d *linePlaneAnglePlaneDef) kindName() string { return "line-plane-angle" }
func (d *linePlaneAnglePlaneDef) refs() []WorkRef  { return []WorkRef{d.line, d.base} }
func (d *linePlaneAnglePlaneDef) eval(r workResolver) (sketch.Plane, error) {
	line, err := r.axis(d.line)
	if err != nil {
		return sketch.Plane{}, err
	}
	base, err := r.plane(d.base)
	if err != nil {
		return sketch.Plane{}, err
	}
	return planeThroughLineAtAngle(line, base, d.angle())
}

// twoLinesPlaneDef builds a plane from two lines: Line1 is the X axis and the normal is
// Line1×Line2 (Inventor's AddByTwoLines) — so the plane holds Line1 and, if they are
// coplanar, Line2 as well.
type twoLinesPlaneDef struct{ l1, l2 WorkRef }

func (d *twoLinesPlaneDef) kindName() string { return "two-lines" }
func (d *twoLinesPlaneDef) refs() []WorkRef  { return []WorkRef{d.l1, d.l2} }
func (d *twoLinesPlaneDef) eval(r workResolver) (sketch.Plane, error) {
	l1, err := r.axis(d.l1)
	if err != nil {
		return sketch.Plane{}, err
	}
	l2, err := r.axis(d.l2)
	if err != nil {
		return sketch.Plane{}, err
	}
	x := l1.Direction()
	normal, err := math.UnitVector3FromVector(x.AsVector().Cross(l2.Direction().AsVector()))
	if err != nil {
		return sketch.Plane{}, errors.New("the two lines are parallel, so no plane is defined")
	}
	y, err := math.UnitVector3FromVector(normal.AsVector().Cross(x.AsVector()))
	if err != nil {
		return sketch.Plane{}, err
	}
	return sketch.NewPlane(l1.Origin(), x, y)
}

// normalToCurvePlaneDef passes through a point and is normal to a curve there (Inventor's
// AddByNormalToCurve). Only linear curves (work axes) are supported in phase A; the
// plane's normal is the axis direction.
type normalToCurvePlaneDef struct {
	curve WorkRef
	point WorkRef
}

func (d *normalToCurvePlaneDef) kindName() string { return "normal-to-curve" }
func (d *normalToCurvePlaneDef) refs() []WorkRef  { return []WorkRef{d.curve, d.point} }
func (d *normalToCurvePlaneDef) eval(r workResolver) (sketch.Plane, error) {
	axis, err := r.axis(d.curve)
	if err != nil {
		return sketch.Plane{}, err
	}
	p, err := r.point(d.point)
	if err != nil {
		return sketch.Plane{}, err
	}
	return planeFromOriginNormal(p, axis.Direction())
}

// AddFixed creates a user plane fixed at origin with the given in-plane axes.
//
//	x, _ := math.NewUnitVector3(1, 0, 0)
//	y, _ := math.NewUnitVector3(0, 1, 0)
//	wp := planes.AddFixed(func() math.Point3 { return math.P3(0, 0, 5) }, x, y)
func (c *WorkPlanes) AddFixed(origin func() math.Point3, x, y math.UnitVector3) *WorkPlane {
	return c.addUser(&fixedFramePlaneDef{origin: origin, x: x, y: y})
}

// AddByPlaneAndPoint creates a plane parallel to base passing through point.
//
//	wp := planes.AddByPlaneAndPoint(feature.OriginXYPlane, apex.Key())
func (c *WorkPlanes) AddByPlaneAndPoint(base, point WorkRef) *WorkPlane {
	return c.addUser(&planeAndPointPlaneDef{base: base, point: point})
}

// AddByTwoPlanes creates the bisecting plane of p1 and p2.
//
//	wp := planes.AddByTwoPlanes(feature.OriginXYPlane, feature.OriginXZPlane)
func (c *WorkPlanes) AddByTwoPlanes(p1, p2 WorkRef) *WorkPlane {
	return c.addUser(&twoPlanesPlaneDef{p1: p1, p2: p2})
}

// AddByLinePlaneAndAngle creates a plane through line at angle (radians) from base.
//
//	wp := planes.AddByLinePlaneAndAngle(feature.OriginXAxis, feature.OriginXYPlane,
//		func() float64 { return 0.7853981633974483 }) // 45° in radians
func (c *WorkPlanes) AddByLinePlaneAndAngle(line, base WorkRef, angle func() float64) *WorkPlane {
	return c.addUser(&linePlaneAnglePlaneDef{line: line, base: base, angle: angle})
}

// AddByTwoLines creates a plane from two lines (l1 is the X axis).
//
//	wp := planes.AddByTwoLines(feature.OriginXAxis, feature.OriginYAxis)
func (c *WorkPlanes) AddByTwoLines(l1, l2 WorkRef) *WorkPlane {
	return c.addUser(&twoLinesPlaneDef{l1: l1, l2: l2})
}

// AddByNormalToCurve creates a plane through point, normal to the curve (a work axis).
//
//	wp := planes.AddByNormalToCurve(feature.OriginZAxis, top.Key())
func (c *WorkPlanes) AddByNormalToCurve(curve, point WorkRef) *WorkPlane {
	return c.addUser(&normalToCurvePlaneDef{curve: curve, point: point})
}

// bisectingPlane returns the plane that bisects p1 and p2. Intersecting planes give the
// dihedral bisector (normal n1+n2) through a point on their intersection line; parallel
// planes give the mid-plane, parallel to both and halfway between them.
func bisectingPlane(p1, p2 sketch.Plane) (sketch.Plane, error) {
	origin, _, err := planeIntersectionLine(p1, p2)
	if err != nil {
		n1 := p1.Normal().AsVector()
		dist := n1.Dot(p1.Origin().VectorTo(p2.Origin()))
		mid := p1.Origin().TranslateBy(n1.Scale(dist / 2))
		return sketch.NewPlane(mid, p1.XAxis(), p1.YAxis())
	}
	normal, err := math.UnitVector3FromVector(p1.Normal().AsVector().Add(p2.Normal().AsVector()))
	if err != nil {
		return sketch.Plane{}, err
	}
	return planeFromOriginNormal(origin, normal)
}

// planeThroughLineAtAngle returns the plane that holds line and is rotated about it by
// angle (radians) away from base's orientation. The zero-angle normal is base's normal
// projected perpendicular to the line (so the result always contains the line); errors
// when base is perpendicular to the line, where the reference orientation is undefined.
func planeThroughLineAtAngle(line *WorkAxis, base sketch.Plane, angle float64) (sketch.Plane, error) {
	dir := line.Direction().AsVector()
	bn := base.Normal().AsVector()
	n0, err := math.UnitVector3FromVector(bn.Sub(dir.Scale(bn.Dot(dir))))
	if err != nil {
		return sketch.Plane{}, errors.New("input plane is perpendicular to the line; the angle is undefined")
	}
	rotated := math.Rotation4(angle, line.Direction(), line.Origin()).TransformVector(n0.AsVector())
	normal, err := math.UnitVector3FromVector(rotated)
	if err != nil {
		return sketch.Plane{}, err
	}
	return planeFromOriginNormal(line.Origin(), normal)
}

// planeFromOriginNormal builds a sketch plane at origin with the given normal, choosing
// an arbitrary in-plane X axis (the surface/datum frame's roll is unconstrained for the
// definitions that fix only a normal — bisector, angle, normal-to-curve, tangent).
func planeFromOriginNormal(origin math.Point3, normal math.UnitVector3) (sketch.Plane, error) {
	x, err := perpendicularTo(normal)
	if err != nil {
		return sketch.Plane{}, err
	}
	y, err := math.UnitVector3FromVector(normal.AsVector().Cross(x.AsVector()))
	if err != nil {
		return sketch.Plane{}, err
	}
	return sketch.NewPlane(origin, x, y)
}

// perpendicularTo returns a unit vector perpendicular to n, crossing it with whichever
// principal axis is least aligned with it (so the cross is well-conditioned).
func perpendicularTo(n math.UnitVector3) (math.UnitVector3, error) {
	seed := math.V3(1, 0, 0)
	if d := n.AsVector().Dot(seed); d > 0.9 || d < -0.9 {
		seed = math.V3(0, 1, 0)
	}
	return math.UnitVector3FromVector(n.AsVector().Cross(seed))
}
