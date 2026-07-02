// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/solve/ad"
)

// DimensionConstraint3D sizes a 3D sketch, backed by a model parameter like its 2D
// counterpart. It implements [Constraint], so the dimension-agnostic Newton core
// (F05) solves 3D sketches unchanged.
type DimensionConstraint3D struct {
	constraintBase
	kind   DimKind
	driven bool
	param  *param.Parameter
	// measure is the float quantity (for reporting and the residual value); measureAD is
	// its dual twin for exact partials. measureAD is nil ONLY for the spline-length
	// dimension, whose value comes from the kernel NURBS sampler — a black box with no
	// closed-form derivative (see Partials) — and which therefore finite-differences.
	measure     func() float64
	measureAD   adFunc1
	vars        []*math.Scalar
	refs        []Entity     // dimensioned operands, for serialization
	planeNormal math.Vector3 // point-plane dimension only
}

// Refs returns the dimensioned operands (points/lines/circles), in factory order.
func (d *DimensionConstraint3D) Refs() []Entity { return d.refs }

// Parameter returns the backing parameter; Measured returns the live value.
func (d *DimensionConstraint3D) Parameter() *param.Parameter { return d.param }
func (d *DimensionConstraint3D) Measured() float64           { return d.measure() }

// KindName returns the wire/types name of this 3D dimension's kind
// ([oblikovati.org/api/types.Dimension3DConstraintKind]). The set grows with the
// 3D dimension factories (M22-F06); unmapped kinds report "unknown".
func (d *DimensionConstraint3D) KindName() string {
	switch d.kind {
	case DistanceDim:
		return "distance"
	case LengthDimKind3D:
		return "lineLength"
	case RadiusDim:
		return "radius"
	case PointPlaneDimKind3D:
		return "pointPlaneDistance"
	case AngleDim:
		return "twoLineAngle"
	case SplineLengthDimKind3D:
		return "splineLength"
	default:
		return "unknown"
	}
}

// Drive sets the dimension's target value (in its parameter's unit), moving the geometry
// on the next solve.
func (d *DimensionConstraint3D) Drive(value float64) error {
	return d.param.SetValue(param.Q(value, d.param.Unit()))
}

// Driven reports/sets whether the dimension only reports rather than constrains.
func (d *DimensionConstraint3D) Driven() bool          { return d.driven }
func (d *DimensionConstraint3D) SetDriven(driven bool) { d.driven = driven }

// Residuals returns measured-minus-target when driving, nil when driven.
func (d *DimensionConstraint3D) Residuals() []float64 {
	if d.driven {
		return nil
	}
	return []float64{d.measure() - d.param.ModelValue()}
}

// Partials returns the dimension's exact Jacobian row from its dual measure, or nil when
// driven. The lone exception is the spline-length dimension (measureAD nil): its value is
// produced by the kernel NURBS sampler, a black box with no closed-form derivative, so it
// alone finite-differences over its own variables (restoring each) — every other 3D
// dimension is exact and perturbs nothing (#1417).
func (d *DimensionConstraint3D) Partials() [][]float64 {
	if d.driven {
		return nil
	}
	if d.measureAD == nil {
		return sampledMeasurePartials(d.vars, d.measure)
	}
	return adPartials(d.vars, func(v []ad.Number) []ad.Number {
		return []ad.Number{d.measureAD(v).AddConst(-d.param.ModelValue())}
	})
}

// Variables returns the 3D DOFs this dimension constrains.
func (d *DimensionConstraint3D) Variables() []*math.Scalar {
	if d.driven {
		return nil
	}
	return d.vars
}

// sampledMeasurePartials central-differences a single black-box measure (the NURBS-sampled
// spline length) over its own variables, restoring each — the one residual whose
// derivative is not closed-form (#1417). The target is constant, so ∂residual/∂v = ∂measure/∂v.
func sampledMeasurePartials(vars []*math.Scalar, measure func() float64) [][]float64 {
	const h = 1e-7 // tol:numeric — black-box (NURBS spline length) finite-difference step
	row := make([]float64, len(vars))
	for i, v := range vars {
		orig := *v
		*v = orig + h
		plus := measure()
		*v = orig - h
		minus := measure()
		*v = orig
		row[i] = (plus - minus) / (2 * h)
	}
	return [][]float64{row}
}

// DimensionConstraints3D owns a 3D sketch's dimensions and their parameters.
type DimensionConstraints3D struct {
	params *param.Parameters
	items  []*DimensionConstraint3D
	seq    int
}

// NewDimensionConstraints3D creates a 3D dimension collection backed by params.
func NewDimensionConstraints3D(params *param.Parameters) *DimensionConstraints3D {
	return &DimensionConstraints3D{params: params}
}

// All returns the 3D dimensions; Count/Item index them.
func (dc *DimensionConstraints3D) All() []*DimensionConstraint3D {
	out := make([]*DimensionConstraint3D, len(dc.items))
	copy(out, dc.items)
	return out
}
func (dc *DimensionConstraints3D) Count() int                        { return len(dc.items) }
func (dc *DimensionConstraints3D) Item(i int) *DimensionConstraint3D { return dc.items[i] }

// create builds a 3D dimension of the given kind, backed by a fresh model parameter, over
// a dual measure (the source of both its value and its exact Jacobian row, #1417) and the
// scalar DOFs it constrains. It is the shared seam every closed-form AddXxx factory uses.
func (dc *DimensionConstraints3D) create(kind DimKind, expression string, refs []Entity, measure adFunc1, vars []*math.Scalar) (*DimensionConstraint3D, error) {
	d, err := dc.newDimension(kind, expression, refs, func() float64 { return adMeasureValue(vars, measure) }, vars)
	if err != nil {
		return nil, err
	}
	d.measureAD = measure
	return d, nil
}

// createSampled builds a 3D dimension whose measure is a black-box float closure (the
// NURBS-sampled spline length) with no closed-form derivative — its measureAD stays nil
// and Partials finite-differences it.
func (dc *DimensionConstraints3D) createSampled(kind DimKind, expression string, refs []Entity, measure func() float64, vars []*math.Scalar) (*DimensionConstraint3D, error) {
	return dc.newDimension(kind, expression, refs, measure, vars)
}

// newDimension is the common construction step shared by create and createSampled.
func (dc *DimensionConstraints3D) newDimension(kind DimKind, expression string, refs []Entity, measure func() float64, vars []*math.Scalar) (*DimensionConstraint3D, error) {
	p, err := dc.params.AddModelParameter(dc.nextName(), expression)
	if err != nil {
		return nil, fmt.Errorf("sketch: 3D dimension parameter: %w", err)
	}
	d := &DimensionConstraint3D{constraintBase: newConstraint(), kind: kind, param: p, measure: measure, vars: vars, refs: refs}
	dc.items = append(dc.items, d)
	return d, nil
}

// AddDistance dimensions the distance between two 3D points.
func (dc *DimensionConstraints3D) AddDistance(a, b *Point3D, expression string) (*DimensionConstraint3D, error) {
	measure := func(v []ad.Number) ad.Number { return adV3(v, 0).Sub(adV3(v, 3)).Length() }
	return dc.create(DistanceDim, expression, []Entity{a, b}, measure,
		[]*math.Scalar{&a.X, &a.Y, &a.Z, &b.X, &b.Y, &b.Z})
}

// AddLineLength dimensions a 3D line's length (the distance between its endpoints).
func (dc *DimensionConstraints3D) AddLineLength(l *Line3D, expression string) (*DimensionConstraint3D, error) {
	measure := func(v []ad.Number) ad.Number { return adLine3DDir(v, 0).Length() }
	return dc.create(LengthDimKind3D, expression, []Entity{l}, measure,
		[]*math.Scalar{&l.A.X, &l.A.Y, &l.A.Z, &l.B.X, &l.B.Y, &l.B.Z})
}

// AddRadius dimensions a 3D circle's radius.
func (dc *DimensionConstraints3D) AddRadius(c *Circle3D, expression string) (*DimensionConstraint3D, error) {
	measure := func(v []ad.Number) ad.Number { return v[0] }
	return dc.create(RadiusDim, expression, []Entity{c}, measure, []*math.Scalar{&c.Radius})
}

// AddPointPlaneDistance dimensions the signed distance from a 3D point to the origin
// plane with the given unit normal (XY ⇒ +Z, XZ ⇒ +Y, YZ ⇒ +X): the point's coordinate
// along that normal.
func (dc *DimensionConstraints3D) AddPointPlaneDistance(p *Point3D, normal math.Vector3, expression string) (*DimensionConstraint3D, error) {
	measure := func(v []ad.Number) ad.Number { return adV3(v, 0).Dot(adConstVec3(normal)) }
	d, err := dc.create(PointPlaneDimKind3D, expression, []Entity{p}, measure, []*math.Scalar{&p.X, &p.Y, &p.Z})
	if err != nil {
		return nil, err
	}
	d.planeNormal = normal
	return d, nil
}

// AddTwoLineAngle dimensions the angle between two 3D lines (radians). The unsigned angle
// atan2(|d1×d2|, d1·d2) equals acos(cos θ) on [0,π] but differentiates cleanly.
func (dc *DimensionConstraints3D) AddTwoLineAngle(l1, l2 *Line3D, expression string) (*DimensionConstraint3D, error) {
	measure := func(v []ad.Number) ad.Number {
		d1, d2 := adLine3DDir(v, 0), adLine3DDir(v, 6)
		return d1.Cross(d2).Length().Atan2(d1.Dot(d2))
	}
	return dc.create(AngleDim, expression, []Entity{l1, l2}, measure, append(line3DVars(l1), line3DVars(l2)...))
}

// AddSplineLength dimensions a 3D spline's arc length, measured over the same
// representative Catmull-Rom polyline the spline draws and picks with (issue #144 —
// the last declared 3D dimension kind without a factory).
func (dc *DimensionConstraints3D) AddSplineLength(sp *Spline3D, expression string) (*DimensionConstraint3D, error) {
	if len(sp.Points) < 2 {
		return nil, fmt.Errorf("sketch: splineLength: spline %d has %d points, need at least 2", sp.EntityID(), len(sp.Points))
	}
	return dc.createSampled(SplineLengthDimKind3D, expression, []Entity{sp},
		func() float64 { return splineLength3D(sp) }, sp.smoothVars3D())
}

// splineLength3D returns the arc length of the spline's representative polyline.
func splineLength3D(sp *Spline3D) float64 {
	pts := sp.Sample()
	total := 0.0
	for i := 0; i+1 < len(pts); i++ {
		total += float64(pts[i].DistanceTo(pts[i+1]))
	}
	return total
}

// angleBetweenLines3D returns the unsigned angle (radians) between two 3D line directions.
func angleBetweenLines3D(l1, l2 *Line3D) float64 {
	d1 := l1.A.Position().VectorTo(l1.B.Position())
	d2 := l2.A.Position().VectorTo(l2.B.Position())
	n1, n2 := float64(d1.Length()), float64(d2.Length())
	if n1 == 0 || n2 == 0 {
		return 0
	}
	cos := float64(d1.Dot(d2)) / (n1 * n2)
	return stdmath.Acos(math.Clamp(cos, -1, 1))
}

func (dc *DimensionConstraints3D) nextName() string {
	for {
		name := fmt.Sprintf("d3_%d", dc.seq)
		dc.seq++
		if _, taken := dc.params.ByName(name); !taken {
			return name
		}
	}
}
