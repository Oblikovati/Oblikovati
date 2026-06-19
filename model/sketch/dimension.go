// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/model/param"
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
	kind    DimKind
	driven  bool
	param   *param.Parameter
	limits  ConstraintLimits
	measure func() float64
	vars    []*math.Scalar
	refs    []Entity // the dimensioned geometry (points/lines/arcs), for editing + serialization
	farSide bool     // tangentDistance only: dimension to the far tangent point (#152)
}

// FarSide reports whether a tangent-distance dimension measures to the far tangent point
// (#152); false (the near side) for every other kind.
func (d *DimensionConstraint) FarSide() bool { return d.farSide }

// Refs returns the geometry the dimension measures (points for a distance, the line
// pair for an angle, the circle for a radius, …). It is what serialization records so
// the dimension can be rebound on open.
func (d *DimensionConstraint) Refs() []Entity { return d.refs }

// Kind returns the dimension kind.
func (d *DimensionConstraint) Kind() DimKind { return d.kind }

// Parameter returns the backing model parameter (its expression is the dimension value).
func (d *DimensionConstraint) Parameter() *param.Parameter { return d.param }

// Measured returns the current geometric value of the dimensioned quantity.
func (d *DimensionConstraint) Measured() float64 { return d.measure() }

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

// Residuals returns measured-minus-target when driving, or nil when driven.
func (d *DimensionConstraint) Residuals() []float64 {
	if d.driven {
		return nil
	}
	return []float64{d.measure() - d.param.ModelValue()}
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
			_ = dc.params.Delete(d.param.ID())
			return true
		}
	}
	return false
}

// AddDistance dimensions the distance between two points to expression (e.g. "25 mm").
func (dc *DimensionConstraints) AddDistance(a, b *Point, expression string) (*DimensionConstraint, error) {
	measure := func() float64 { return a.Position().DistanceTo(b.Position()) }
	vars := []*math.Scalar{&a.X, &a.Y, &b.X, &b.Y}
	return dc.create(DistanceDim, expression, []Entity{a, b}, measure, vars)
}

// AddRadius dimensions the radius of a circle or an arc (Inventor allows both). For
// a circle the radius is a stored DOF; for an arc it is the center-to-start distance,
// so the solver drives the center/start points (circularVars) to satisfy the target.
func (dc *DimensionConstraints) AddRadius(c CircularCurve, expression string) (*DimensionConstraint, error) {
	return dc.create(RadiusDim, expression, []Entity{c}, func() float64 { return float64(c.CurveRadius()) }, c.circularVars())
}

// AddDiameter dimensions the diameter of a circle or an arc.
func (dc *DimensionConstraints) AddDiameter(c CircularCurve, expression string) (*DimensionConstraint, error) {
	return dc.create(DiameterDim, expression, []Entity{c}, func() float64 { return 2 * float64(c.CurveRadius()) }, c.circularVars())
}

// AddAngle dimensions the angle (radians, in [0,π]) between two lines.
func (dc *DimensionConstraints) AddAngle(l1, l2 *Line, expression string) (*DimensionConstraint, error) {
	measure := func() float64 {
		d1x, d1y := lineDir(l1)
		d2x, d2y := lineDir(l2)
		return stdmath.Atan2(stdmath.Abs(d1x*d2y-d1y*d2x), d1x*d2x+d1y*d2y)
	}
	return dc.create(AngleDim, expression, []Entity{l1, l2}, measure, lineVars(l1, l2))
}

// AddTangentDistance dimensions the distance from a line to a circle/arc measured to its
// tangent point: |perpendicular distance from the center to the line| ∓ radius — the near
// side (default) subtracts the radius, the far side adds it (#152). The solver drives the
// line's endpoints and the curve's center/radius (circularVars) to satisfy the target.
func (dc *DimensionConstraints) AddTangentDistance(l *Line, c CircularCurve, farSide bool, expression string) (*DimensionConstraint, error) {
	measure := func() float64 {
		signed, ok := signedCenterToLine(l, c)
		if !ok {
			return 0
		}
		d := stdmath.Abs(signed)
		if farSide {
			return d + c.CurveRadius()
		}
		return d - c.CurveRadius()
	}
	vars := append([]*math.Scalar{&l.A.X, &l.A.Y, &l.B.X, &l.B.Y}, c.circularVars()...)
	d, err := dc.create(TangentDistanceDim, expression, []Entity{l, c}, measure, vars)
	if err != nil {
		return nil, err
	}
	d.farSide = farSide
	return d, nil
}

// AddArcLength dimensions an arc's length (radius × swept angle).
func (dc *DimensionConstraints) AddArcLength(a *Arc, expression string) (*DimensionConstraint, error) {
	measure := func() float64 { return a.Radius() * arcSweep(a) }
	vars := []*math.Scalar{&a.Center.X, &a.Center.Y, &a.Start.X, &a.Start.Y, &a.End.X, &a.End.Y}
	return dc.create(ArcLengthDim, expression, []Entity{a}, measure, vars)
}

// create builds a driving dimension backed by a fresh model parameter. refs records
// the dimensioned geometry for editing and serialization.
func (dc *DimensionConstraints) create(kind DimKind, expression string, refs []Entity, measure func() float64, vars []*math.Scalar) (*DimensionConstraint, error) {
	p, err := dc.params.AddModelParameter(dc.nextName(), expression)
	if err != nil {
		return nil, fmt.Errorf("sketch: dimension parameter: %w", err)
	}
	d := &DimensionConstraint{constraintBase: newConstraint(), kind: kind, param: p, measure: measure, vars: vars, refs: refs}
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

// arcSweep returns the unsigned swept angle of an arc about its center.
func arcSweep(a *Arc) float64 {
	c := a.Center.Position()
	s := c.VectorTo(a.Start.Position())
	e := c.VectorTo(a.End.Position())
	sweep := stdmath.Atan2(s.Cross(e), s.Dot(e))
	return stdmath.Abs(sweep)
}
