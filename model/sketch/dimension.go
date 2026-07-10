// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/solve/ad"
)

// DimKind classifies a dimensional constraint.
type DimKind uint8

const (
	DistanceDim DimKind = iota
	AngleDim
	RadiusDim
	DiameterDim
	ArcLengthDim
	// Appended for M21-F07 PBI-214 (do not reorder — iota ids are stable).
	OffsetDim          // perpendicular distance from a point to a line
	ThreePointAngleDim // angle a–vertex–b
	EllipseRadiusDim   // an ellipse's major radius
	// Appended for M22-F06 (3D dimensions; do not reorder).
	LengthDimKind3D     // a 3D line's length
	PointPlaneDimKind3D // signed distance from a 3D point to an origin plane
	// Appended for issue #144 (do not reorder).
	SplineLengthDimKind3D // a 3D spline's sampled arc length
	// Appended for issue #152 (do not reorder).
	TangentDistanceDim // distance from a line to a circle/arc's near/far tangent point
	// Appended for issue #1874 (do not reorder).
	OffsetSplineDim // an offset spline's offset distance from its parent
)

// ConstraintLimits bounds a dimension's value for drive/animation. When Enabled,
// driving the dimension clamps the value into [Min, Max].
type ConstraintLimits struct {
	Min     float64
	Max     float64
	Enabled bool
}

// clamp returns v constrained to the limits when enabled.
func (l ConstraintLimits) clamp(v float64) float64 {
	if !l.Enabled {
		return v
	}
	if v < l.Min {
		return l.Min
	}
	if v > l.Max {
		return l.Max
	}
	return v
}

// DimensionConstraint sizes a sketch parametrically. Its target value is an owned
// model [param.Parameter] (so it is an editable expression in the parameter DAG,
// core/04); the residual is measured-minus-target, which the solver drives to zero.
// A driven dimension reports the measured value instead of constraining (it
// contributes no residual). The solver and parameter DAG meet only here, through
// the parameter's value (modeling/00).
type DimensionConstraint struct {
	constraintBase
	kind           DimKind
	driven         bool
	param          *param.Parameter
	limits         ConstraintLimits
	measureAD      adFunc1 // the measured quantity over duals; the float measure derives from it
	vars           []*math.Scalar
	refs           []Entity            // the dimensioned geometry (points/lines/arcs), for editing + serialization
	farSide        bool                // tangentDistance only: dimension to the far tangent point (#152)
	orientation    DistanceOrientation // distance only: aligned (Euclidean) / horizontal (Δx) / vertical (Δy) (#1869)
	textPoint      *math.Point2        // annotation-text placement; nil ⇒ unset (Inventor TextPoint, #1875)
	linearDiameter bool                // offset/tangentDistance only: value reads as a diameter, 2× the distance (#1875)
}

// DistanceOrientation selects what a two-point distance dimension measures — Inventor's
// DimensionOrientationEnum.
type DistanceOrientation uint8

const (
	// AlignedDistance measures the Euclidean distance |P2 − P1| (the default).
	AlignedDistance DistanceOrientation = iota
	// HorizontalDistance measures only the X separation |P2.x − P1.x| (the pair stays free to
	// slide vertically).
	HorizontalDistance
	// VerticalDistance measures only the Y separation |P2.y − P1.y|.
	VerticalDistance
)

// Orientation reports what a distance dimension measures (aligned for every non-distance kind).
func (d *DimensionConstraint) Orientation() DistanceOrientation { return d.orientation }

// DistanceOrientationName renders an orientation as its stable name ("" for the aligned default, so
// the common case serializes nothing) — the vocabulary shared by the wire router and the codec.
func DistanceOrientationName(o DistanceOrientation) string {
	switch o {
	case HorizontalDistance:
		return "horizontal"
	case VerticalDistance:
		return "vertical"
	default:
		return ""
	}
}

// ParseDistanceOrientation maps a name ("aligned"/"horizontal"/"vertical", empty ⇒ aligned) to its
// orientation; ok is false for an unknown name.
func ParseDistanceOrientation(name string) (DistanceOrientation, bool) {
	switch name {
	case "", "aligned":
		return AlignedDistance, true
	case "horizontal":
		return HorizontalDistance, true
	case "vertical":
		return VerticalDistance, true
	default:
		return AlignedDistance, false
	}
}

// FarSide reports whether a tangent-distance dimension measures to the far tangent point
// (#152); false (the near side) for every other kind.
func (d *DimensionConstraint) FarSide() bool { return d.farSide }

// LinearDiameter reports whether an offset/tangent-distance dimension reads as a diameter —
// its value is 2× the measured linear distance (Inventor bool LinearDiameter, #1875).
func (d *DimensionConstraint) LinearDiameter() bool { return d.linearDiameter }

// TextPoint returns the dimension's annotation-text placement and whether one is stored
// (Inventor Point2d TextPoint, #1875).
func (d *DimensionConstraint) TextPoint() (math.Point2, bool) {
	if d.textPoint == nil {
		return math.Point2{}, false
	}
	return *d.textPoint, true
}

// SetTextPoint records the dimension's annotation-text placement (#1875).
func (d *DimensionConstraint) SetTextPoint(p math.Point2) { d.textPoint = &p }

// Refs returns the geometry the dimension measures (points for a distance, the line
// pair for an angle, the circle for a radius, …). It is what serialization records so
// the dimension can be rebound on open.
func (d *DimensionConstraint) Refs() []Entity { return d.refs }

// Kind returns the dimension kind.
func (d *DimensionConstraint) Kind() DimKind { return d.kind }

// Parameter returns the backing model parameter (its expression is the dimension value).
func (d *DimensionConstraint) Parameter() *param.Parameter { return d.param }

// Measured returns the current geometric value of the dimensioned quantity.
func (d *DimensionConstraint) Measured() float64 { return adMeasureValue(d.vars, d.measureAD) }

// Driven reports whether the dimension only reports (true) or constrains (false).
func (d *DimensionConstraint) Driven() bool { return d.driven }

// SetDriven switches between driving (constrains geometry) and driven (reports).
func (d *DimensionConstraint) SetDriven(driven bool) { d.driven = driven }

// Limits returns the drive limits; SetLimits enables and sets them.
func (d *DimensionConstraint) Limits() ConstraintLimits { return d.limits }

func (d *DimensionConstraint) SetLimits(min, max float64) {
	d.limits = ConstraintLimits{Min: min, Max: max, Enabled: true}
}

// Drive sets the dimension's target value (clamped to any enabled limits) by
// updating the backing parameter. Used by drive/animation.
func (d *DimensionConstraint) Drive(value float64) error {
	return d.param.SetValue(param.Q(d.limits.clamp(value), d.param.Unit()))
}

// residualAD: measured-minus-target, the target being a constant from the parameter DAG.
func (d *DimensionConstraint) residualAD(v []ad.Number) []ad.Number {
	return []ad.Number{d.measureAD(v).AddConst(-d.param.ModelValue())}
}

// Residuals returns measured-minus-target when driving, or nil when driven.
func (d *DimensionConstraint) Residuals() []float64 {
	if d.driven {
		return nil
	}
	return adResiduals(d.vars, d.residualAD)
}

// Partials returns the dimension's exact Jacobian row, or nil when driven (it contributes
// no equation). It reuses the dual measure, so the derivative cannot drift from the value.
func (d *DimensionConstraint) Partials() [][]float64 {
	if d.driven {
		return nil
	}
	return adPartials(d.vars, d.residualAD)
}

// Variables returns the geometry DOFs this dimension constrains (none when driven).
func (d *DimensionConstraint) Variables() []*math.Scalar {
	if d.driven {
		return nil
	}
	return d.vars
}

// DimensionConstraints owns a sketch's dimensional constraints and creates the
// model parameters that back them.
type DimensionConstraints struct {
	params *param.Parameters
	items  []*DimensionConstraint
	seq    int
}

// All returns the dimensions in creation order.
func (dc *DimensionConstraints) All() []*DimensionConstraint {
	out := make([]*DimensionConstraint, len(dc.items))
	copy(out, dc.items)
	return out
}

// Count returns the number of dimensions; Item returns the i-th.
func (dc *DimensionConstraints) Count() int                      { return len(dc.items) }
func (dc *DimensionConstraints) Item(i int) *DimensionConstraint { return dc.items[i] }

// Delete removes a dimension and its backing parameter (used when the user deletes a
// dimension and by AutoDimension's rank trials). Returns whether it was present. The
// parameter is just-created with no dependents in the trial case, so its removal is safe.
func (dc *DimensionConstraints) Delete(d *DimensionConstraint) bool {
	for i, x := range dc.items {
		if x == d {
			dc.items = append(dc.items[:i], dc.items[i+1:]...)
			// Owner cascade: the dimension owns this model parameter, so the
			// in-use guard on Delete does not apply (the owner itself is going).
			_ = dc.params.DeleteForOwner(d.param.ID())
			return true
		}
	}
	return false
}

// AddDistance dimensions the aligned (Euclidean) distance between two points to expression
// (e.g. "25 mm").
func (dc *DimensionConstraints) AddDistance(a, b *Point, expression string) (*DimensionConstraint, error) {
	return dc.AddDistanceOriented(a, b, expression, AlignedDistance)
}

// AddDistanceOriented dimensions the distance between two points along the chosen orientation:
// aligned = Euclidean |P2 − P1|, horizontal = |Δx| (leaves the Y separation free), vertical = |Δy|
// (Inventor's DimensionOrientationEnum, #1869).
func (dc *DimensionConstraints) AddDistanceOriented(a, b *Point, expression string, o DistanceOrientation) (*DimensionConstraint, error) {
	vars := []*math.Scalar{&a.X, &a.Y, &b.X, &b.Y}
	d, err := dc.create(DistanceDim, expression, []Entity{a, b}, distanceMeasure(o), vars)
	if err != nil {
		return nil, err
	}
	d.orientation = o
	return d, nil
}

// distanceMeasure is the autodiff measure for a two-point distance dimension of orientation o over
// the packed vars [a.X, a.Y, b.X, b.Y]; the solver's Jacobian follows from it automatically.
func distanceMeasure(o DistanceOrientation) adFunc1 {
	switch o {
	case HorizontalDistance:
		return func(v []ad.Number) ad.Number { return v[0].Sub(v[2]).Abs() }
	case VerticalDistance:
		return func(v []ad.Number) ad.Number { return v[1].Sub(v[3]).Abs() }
	default:
		return func(v []ad.Number) ad.Number { return ad.V2(v[0], v[1]).Sub(ad.V2(v[2], v[3])).Length() }
	}
}

// AddRadius dimensions the radius of a circle or an arc (Inventor allows both). For
// a circle the radius is a stored DOF; for an arc it is the center-to-start distance,
// so the solver drives the center/start points (circularVars) to satisfy the target.
func (dc *DimensionConstraints) AddRadius(c CircularCurve, expression string) (*DimensionConstraint, error) {
	measure := func(v []ad.Number) ad.Number { _, r, _ := c.circularFrameAD(v, 0); return r }
	return dc.create(RadiusDim, expression, []Entity{c}, measure, c.circularVars())
}

// AddDiameter dimensions the diameter of a circle or an arc.
func (dc *DimensionConstraints) AddDiameter(c CircularCurve, expression string) (*DimensionConstraint, error) {
	measure := func(v []ad.Number) ad.Number { _, r, _ := c.circularFrameAD(v, 0); return r.Scale(2) }
	return dc.create(DiameterDim, expression, []Entity{c}, measure, c.circularVars())
}

// AddAngle dimensions the angle (radians, in [0,π]) between two lines.
func (dc *DimensionConstraints) AddAngle(l1, l2 *Line, expression string) (*DimensionConstraint, error) {
	measure := func(v []ad.Number) ad.Number {
		d1, d2 := adLineDirs(v)
		return d1.Cross(d2).Abs().Atan2(d1.Dot(d2))
	}
	return dc.create(AngleDim, expression, []Entity{l1, l2}, measure, lineVars(l1, l2))
}

// AddTangentDistance dimensions the distance from a line to a circle/arc measured to its
// tangent point: |perpendicular distance from the center to the line| ∓ radius — the near
// side (default) subtracts the radius, the far side adds it (#152). The solver drives the
// line's endpoints and the curve's center/radius (circularVars) to satisfy the target.
func (dc *DimensionConstraints) AddTangentDistance(l *Line, c CircularCurve, farSide, linearDiameter bool, expression string) (*DimensionConstraint, error) {
	measure := diameterScaled(func(v []ad.Number) ad.Number {
		a, b := ad.V2(v[0], v[1]), ad.V2(v[2], v[3])
		center, radius, _ := c.circularFrameAD(v, 4)
		dir := b.Sub(a)
		length := dir.Length()
		if length.Val() == 0 {
			return ad.Const(0)
		}
		d := dir.Cross(center.Sub(a)).Div(length).Abs() // |perpendicular distance|
		if farSide {
			return d.Add(radius)
		}
		return d.Sub(radius)
	}, linearDiameter)
	vars := append([]*math.Scalar{&l.A.X, &l.A.Y, &l.B.X, &l.B.Y}, c.circularVars()...)
	d, err := dc.create(TangentDistanceDim, expression, []Entity{l, c}, measure, vars)
	if err != nil {
		return nil, err
	}
	d.farSide, d.linearDiameter = farSide, linearDiameter
	return d, nil
}

// diameterScaled wraps a linear measure to read as a diameter — 2× the distance — when
// linearDiameter is set, so an offset/tangent-distance dimension presents and drives a diameter
// value (#1875). Doubling the measure keeps report and drive consistent: a target T solves to a
// linear distance of T/2.
func diameterScaled(measure adFunc1, linearDiameter bool) adFunc1 {
	if !linearDiameter {
		return measure
	}
	return func(v []ad.Number) ad.Number { return measure(v).Scale(2) }
}

// AddArcLength dimensions an arc's length (radius × swept angle).
func (dc *DimensionConstraints) AddArcLength(a *Arc, expression string) (*DimensionConstraint, error) {
	measure := func(v []ad.Number) ad.Number {
		center, start, end := ad.V2(v[0], v[1]), ad.V2(v[2], v[3]), ad.V2(v[4], v[5])
		radius := start.Sub(center).Length()
		s, e := start.Sub(center), end.Sub(center)
		sweep := s.Cross(e).Atan2(s.Dot(e)).Abs() // unsigned swept angle
		return radius.Mul(sweep)
	}
	vars := []*math.Scalar{&a.Center.X, &a.Center.Y, &a.Start.X, &a.Start.Y, &a.End.X, &a.End.Y}
	return dc.create(ArcLengthDim, expression, []Entity{a}, measure, vars)
}

// create builds a driving dimension backed by a fresh model parameter. measure is the
// dimensioned quantity over duals (the single source of both its value and its exact
// Jacobian row, #1417); refs records the dimensioned geometry for editing/serialization.
func (dc *DimensionConstraints) create(kind DimKind, expression string, refs []Entity, measure adFunc1, vars []*math.Scalar) (*DimensionConstraint, error) {
	p, err := dc.params.AddModelParameter(dc.nextName(), expression)
	if err != nil {
		return nil, fmt.Errorf("sketch: dimension parameter: %w", err)
	}
	d := &DimensionConstraint{constraintBase: newConstraint(), kind: kind, param: p, measureAD: measure, vars: vars, refs: refs}
	dc.items = append(dc.items, d)
	return d, nil
}

// nextName mints an unused parameter name (d0, d1, …).
func (dc *DimensionConstraints) nextName() string {
	for {
		name := fmt.Sprintf("d%d", dc.seq)
		dc.seq++
		if _, taken := dc.params.ByName(name); !taken {
			return name
		}
	}
}
