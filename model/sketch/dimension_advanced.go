// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// AddOffsetDim dimensions the perpendicular distance from point p to line l. With linearDiameter
// the value reads as a diameter — 2× that distance (Inventor's LinearDiameter, #1875).
func (dc *DimensionConstraints) AddOffsetDim(p *Point, l *Line, linearDiameter bool, expression string) (*DimensionConstraint, error) {
	measure := diameterScaled(func(v []ad.Number) ad.Number {
		pt, a, b := ad.V2(v[0], v[1]), ad.V2(v[2], v[3]), ad.V2(v[4], v[5])
		dir := b.Sub(a)
		length := dir.Length()
		if length.Val() == 0 {
			return pt.Sub(a).Length() // degenerate line: distance to its point
		}
		return dir.Cross(pt.Sub(a)).Div(length).Abs()
	}, linearDiameter)
	vars := []*math.Scalar{&p.X, &p.Y, &l.A.X, &l.A.Y, &l.B.X, &l.B.Y}
	d, err := dc.create(OffsetDim, expression, []Entity{p, l}, measure, vars)
	if err != nil {
		return nil, err
	}
	d.linearDiameter = linearDiameter
	return d, nil
}

// AddThreePointAngle dimensions the angle a–vertex–b (the included angle at vertex).
func (dc *DimensionConstraints) AddThreePointAngle(vertex, a, b *Point, expression string) (*DimensionConstraint, error) {
	measure := func(v []ad.Number) ad.Number {
		va := ad.V2(v[2], v[3]).Sub(ad.V2(v[0], v[1])) // a − vertex
		vb := ad.V2(v[4], v[5]).Sub(ad.V2(v[0], v[1])) // b − vertex
		return va.Cross(vb).Abs().Atan2(va.Dot(vb))
	}
	vars := []*math.Scalar{&vertex.X, &vertex.Y, &a.X, &a.Y, &b.X, &b.Y}
	return dc.create(ThreePointAngleDim, expression, []Entity{vertex, a, b}, measure, vars)
}

// AddEllipseRadius dimensions an ellipse's major radius.
func (dc *DimensionConstraints) AddEllipseRadius(e *Ellipse, expression string) (*DimensionConstraint, error) {
	measure := func(v []ad.Number) ad.Number { return v[0] }
	return dc.create(EllipseRadiusDim, expression, []Entity{e}, measure, []*math.Scalar{&e.MajorRadius})
}

// AddOffsetSplineDim drives an offset spline's offset distance from its parent (#1874). The
// measure is the magnitude of the signed offset, so a positive length expression preserves the
// side the offset was created on while setting its distance.
func (dc *DimensionConstraints) AddOffsetSplineDim(o *OffsetSpline, expression string) (*DimensionConstraint, error) {
	measure := func(v []ad.Number) ad.Number { return v[0].Abs() }
	return dc.create(OffsetSplineDim, expression, []Entity{o}, measure, []*math.Scalar{&o.Dist})
}
