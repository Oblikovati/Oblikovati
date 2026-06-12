// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/api/types"

	"oblikovati.org/math"
)

// Point is a constrainable sketch point and the solver's variable carrier: its X,Y
// are the DOFs the solver reads and writes. Entities share points by pointer, so a
// shared endpoint *is* a coincidence, structurally (modeling/00) — no explicit
// constraint needed. A standalone Point is a SketchPoint; endpoints and centers are
// the same type, owned by their curve.
type Point struct {
	id ID
	X  math.Scalar
	Y  math.Scalar
}

// EntityID implements [Entity].
func (p *Point) EntityID() ID { return p.id }

// Position returns the point as a [math.Point2].
func (p *Point) Position() math.Point2 { return math.P2(p.X, p.Y) }

// SetPosition moves the point (used by the solver and by drags).
func (p *Point) SetPosition(q math.Point2) { p.X, p.Y = q.X, q.Y }

// entityBase carries the id, construction flag, and centerline flag shared by curve entities.
type entityBase struct {
	id           ID
	construction bool
	centerline   bool
}

func newEntity() entityBase { return entityBase{id: nextID()} }

// EntityID implements [Entity].
func (e *entityBase) EntityID() ID { return e.id }

// IsConstruction reports whether the entity is construction (reference) geometry — it shapes
// constraints but is not part of a profile. A centerline is always construction (an axis never
// closes a profile).
func (e *entityBase) IsConstruction() bool { return e.construction || e.centerline }

// SetConstruction toggles plain construction geometry.
func (e *entityBase) SetConstruction(c bool) { e.construction = c }

// IsCenterline reports whether the entity is a centerline — construction geometry that also
// serves as an axis (revolve axis, mirror/symmetry axis, diameter dimensions).
func (e *entityBase) IsCenterline() bool { return e.centerline }

// SetCenterline marks the entity as a centerline (implying construction; see IsConstruction).
func (e *entityBase) SetCenterline(c bool) { e.centerline = c }

// Line is a straight segment between two constrainable endpoints.
type Line struct {
	entityBase
	A *Point
	B *Point
}

// StartPoint and EndPoint return the line's endpoints.
func (l *Line) StartPoint() *Point { return l.A }
func (l *Line) EndPoint() *Point   { return l.B }

// Direction returns the (unnormalized) vector from A to B.
func (l *Line) Direction() math.Vector2 { return l.A.Position().VectorTo(l.B.Position()) }

// Axis3D returns the line as a model-space axis (origin at A, direction A→B) on the given sketch
// plane — used to drive a revolve/mirror axis from a centerline.
func (l *Line) Axis3D(plane Plane) (origin math.Point3, dir math.Vector3) {
	a, b := plane.ToModel(l.A.Position()), plane.ToModel(l.B.Position())
	return a, a.VectorTo(b)
}

// Length returns the current endpoint distance.
func (l *Line) Length() math.Scalar { return l.A.Position().DistanceTo(l.B.Position()) }

// Circle is a full circle: a center point and a radius DOF.
type Circle struct {
	entityBase
	Center *Point
	Radius math.Scalar
}

// Arc is a circular arc defined by a center and two endpoints; the radius is
// derived from the center-to-start distance (a tangency/radius constraint keeps the
// end at the same radius). Sweep direction is CounterClockwise.
type Arc struct {
	entityBase
	Center           *Point
	Start            *Point
	End              *Point
	CounterClockwise bool
}

// Radius returns the current center-to-start distance.
func (a *Arc) Radius() math.Scalar { return a.Center.Position().DistanceTo(a.Start.Position()) }

// CircularCurve is sketch geometry defined by a center and a radius — a Circle or an
// Arc. The constraints that act on circular geometry (Concentric, Tangent, EqualRadius
// and the point-on-curve coincidence) accept either, matching the reference API's
// polymorphic constraints: an arc is a circle you can also constrain by its endpoints. The interface
// is sealed (circularVars is unexported) so only Circle and Arc can satisfy it.
type CircularCurve interface {
	Entity
	// CenterPoint returns the curve's center — the concentric/tangent anchor.
	CenterPoint() *Point
	// CurveRadius returns the current radius (a stored DOF for a circle, the
	// center-to-start distance for an arc).
	CurveRadius() math.Scalar
	// circularVars returns the scalar DOFs that move the center or change the radius.
	// For a circle that is its center and radius; for an arc the radius has no DOF of
	// its own, so it is the center and start point that define it.
	circularVars() []*math.Scalar
}

// CenterPoint, CurveRadius and circularVars make Circle a [CircularCurve].
func (c *Circle) CenterPoint() *Point      { return c.Center }
func (c *Circle) CurveRadius() math.Scalar { return c.Radius }
func (c *Circle) circularVars() []*math.Scalar {
	return []*math.Scalar{&c.Center.X, &c.Center.Y, &c.Radius}
}

// CenterPoint, CurveRadius and circularVars make Arc a [CircularCurve].
func (a *Arc) CenterPoint() *Point      { return a.Center }
func (a *Arc) CurveRadius() math.Scalar { return a.Radius() }
func (a *Arc) circularVars() []*math.Scalar {
	// The arc has no radius DOF: its radius is |center − start|, so both points move it.
	return []*math.Scalar{&a.Center.X, &a.Center.Y, &a.Start.X, &a.Start.Y}
}

// Ellipse is a full ellipse: a center, a major-axis direction, and the two radii.
type Ellipse struct {
	entityBase
	Center      *Point
	MajorAxis   math.Vector2
	MajorRadius math.Scalar
	MinorRadius math.Scalar
}

// EllipticalArc is an ellipse restricted to a parametric-angle range [StartAngle,
// EndAngle] (radians, measured in the major/minor frame). It shares the ellipse's
// center, major-axis direction, and two radii.
type EllipticalArc struct {
	entityBase
	Center      *Point
	MajorAxis   math.Vector2
	MajorRadius math.Scalar
	MinorRadius math.Scalar
	StartAngle  math.Scalar
	EndAngle    math.Scalar
}

// SplineFitMethod aliases the public fit-method enum (M06-F11,
// Oblikovati/Oblikovati#626) so call sites stay on the sketch package.
type SplineFitMethod = types.SplineFitMethod

// Spline is a NURBS-style sketch spline through (fit) or near (control) its points.
// FitMethod selects the interpolation parameterization of a fit spline; the zero
// value behaves as the smooth (centripetal) default.
type Spline struct {
	entityBase
	Points    []*Point
	Closed    bool
	FitMethod SplineFitMethod
	fit       bool
	// handles are the active tangency handles keyed by fit-point index
	// (M06-F11; see spline_handles.go).
	handles map[int]*SplineHandle
}

// IsFitType reports whether the spline interpolates its points (fit) rather than
// approximating them (control).
func (s *Spline) IsFitType() bool { return s.fit }

// PointCount returns the number of defining points.
func (s *Spline) PointCount() int { return len(s.Points) }
