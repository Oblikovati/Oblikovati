// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"github.com/Oblikovati/oblikovati/build"
	"github.com/Oblikovati/oblikovati/math"
)

// TransformCurve maps a curve by m, which must be a similarity transform
// (rotation, translation, reflection, and/or uniform scale). Non-uniform scale or
// shear is rejected because it would turn an analytic curve (a circle, an arc)
// into a different curve type that the analytic representation cannot hold.
//
// Example: rotating a circle's edge curve when its body is patterned:
//
//	c2, err := geom.TransformCurve(edge.Geometry(), math.Rotation4(a, axis, ctr))
func TransformCurve(c Curve3, m math.Matrix4) (Curve3, error) {
	scale, err := similarityScale(m)
	if err != nil {
		return nil, err
	}
	switch g := c.(type) {
	case LineSegment:
		return NewLineSegment(m.TransformPoint(g.StartPoint), m.TransformPoint(g.EndPoint)), nil
	case Line:
		return NewLine(m.TransformPoint(g.Origin), m.TransformVector(g.Dir.AsVector()))
	case Circle:
		return transformCircle(g, m, scale)
	case Arc3d:
		return transformArc(g, m, scale)
	case BSplineCurve:
		return transformBSplineCurve(g, m), nil
	default:
		return nil, build.NotYetImplemented(fmt.Sprintf("geom.TransformCurve: %T", c))
	}
}

// TransformSurface maps a surface by m under the same similarity constraint as
// [TransformCurve].
func TransformSurface(s Surface, m math.Matrix4) (Surface, error) {
	scale, err := similarityScale(m)
	if err != nil {
		return nil, err
	}
	switch g := s.(type) {
	case Plane:
		return NewPlaneFromAxes(m.TransformPoint(g.Origin),
			m.TransformVector(g.UAxis.AsVector()), m.TransformVector(g.VAxis.AsVector()))
	case Cylinder:
		return transformCylinder(g, m, scale)
	case Cone:
		return transformCone(g, m)
	case Sphere:
		return NewSphere(m.TransformPoint(g.Center), g.Radius*scale)
	case Torus:
		return transformTorus(g, m, scale)
	case BSplineSurface:
		return transformBSplineSurface(g, m), nil
	default:
		return nil, build.NotYetImplemented(fmt.Sprintf("geom.TransformSurface: %T", s))
	}
}

// transformCircle preserves the circle's reference frame (Normal, RefDir) so its
// parameterization carries over, scaling the radius by the similarity factor.
func transformCircle(c Circle, m math.Matrix4, scale float64) (Circle, error) {
	n, err := math.UnitVector3FromVector(m.TransformVector(c.Normal.AsVector()))
	if err != nil {
		return Circle{}, err
	}
	ref, err := math.UnitVector3FromVector(m.TransformVector(c.RefDir.AsVector()))
	if err != nil {
		return Circle{}, err
	}
	return Circle{Center: m.TransformPoint(c.Center), Normal: n, RefDir: ref, Radius: c.Radius * scale}, nil
}

func transformArc(a Arc3d, m math.Matrix4, scale float64) (Arc3d, error) {
	n, err := math.UnitVector3FromVector(m.TransformVector(a.Normal.AsVector()))
	if err != nil {
		return Arc3d{}, err
	}
	ref, err := math.UnitVector3FromVector(m.TransformVector(a.RefDir.AsVector()))
	if err != nil {
		return Arc3d{}, err
	}
	return Arc3d{
		Center: m.TransformPoint(a.Center), Normal: n, RefDir: ref,
		Radius: a.Radius * scale, StartAngle: a.StartAngle, SweepAngle: a.SweepAngle,
	}, nil
}

func transformCylinder(c Cylinder, m math.Matrix4, scale float64) (Cylinder, error) {
	out, err := NewCylinder(m.TransformPoint(c.Origin), m.TransformVector(c.AxisDir.AsVector()), c.Radius*scale)
	return out, err
}

func transformCone(c Cone, m math.Matrix4) (Cone, error) {
	out, err := NewCone(m.TransformPoint(c.Apex), m.TransformVector(c.AxisDir.AsVector()), c.HalfAngle)
	return out, err
}

func transformTorus(t Torus, m math.Matrix4, scale float64) (Torus, error) {
	out, err := NewTorus(m.TransformPoint(t.Center), m.TransformVector(t.AxisDir.AsVector()),
		t.MajorRadius*scale, t.MinorRadius*scale)
	return out, err
}

// transformBSplineCurve maps a rational B-spline by transforming its control points.
// This is exact for any affine m: the rational basis functions are a partition of
// unity, so T(C(t)) = Σ Rᵢ(t)·T(Pᵢ); weights, knots and degree are invariant. (The
// public entry point still gates on similarity for consistency with the analytic
// types, but the control-point map itself needs no such restriction.) It cannot
// fail — the control count, weights and knots are unchanged, so the value stays valid.
func transformBSplineCurve(c BSplineCurve, m math.Matrix4) BSplineCurve {
	ctrl := make([]math.Point3, len(c.Ctrl))
	for i, p := range c.Ctrl {
		ctrl[i] = m.TransformPoint(p)
	}
	return BSplineCurve{
		Degree:  c.Degree,
		Ctrl:    ctrl,
		Weights: append([]float64(nil), c.Weights...),
		Knots:   append([]float64(nil), c.Knots...),
	}
}

// transformBSplineSurface maps a rational B-spline surface by transforming its control
// net; see [transformBSplineCurve] for why this is exact. Weights, knots and degrees
// are invariant.
func transformBSplineSurface(s BSplineSurface, m math.Matrix4) BSplineSurface {
	ctrl := make([][]math.Point3, len(s.Ctrl))
	for i, row := range s.Ctrl {
		ctrl[i] = make([]math.Point3, len(row))
		for j, p := range row {
			ctrl[i][j] = m.TransformPoint(p)
		}
	}
	weights := make([][]float64, len(s.Weights))
	for i, row := range s.Weights {
		weights[i] = append([]float64(nil), row...)
	}
	return BSplineSurface{
		UDegree: s.UDegree, VDegree: s.VDegree,
		Ctrl:    ctrl,
		Weights: weights,
		UKnots:  append([]float64(nil), s.UKnots...),
		VKnots:  append([]float64(nil), s.VKnots...),
	}
}

// similarityScale returns the uniform scale factor of m and rejects non-uniform
// scale or shear (the analytic types cannot represent the result). The factor is
// always positive; reflection (negative determinant) is a valid similarity whose
// orientation flip is handled by the caller, not folded into the scale.
func similarityScale(m math.Matrix4) (float64, error) {
	sx := m.TransformVector(math.V3(1, 0, 0)).Length()
	sy := m.TransformVector(math.V3(0, 1, 0)).Length()
	sz := m.TransformVector(math.V3(0, 0, 1)).Length()
	const tol = 1e-9
	if sx <= tol || sy <= tol || sz <= tol {
		return 0, fmt.Errorf("geom.similarityScale: degenerate transform (axis lengths %g,%g,%g)", sx, sy, sz)
	}
	if stdmath.Abs(sx-sy) > tol*sx || stdmath.Abs(sx-sz) > tol*sx {
		return 0, fmt.Errorf("geom.similarityScale: non-uniform scale (axis lengths %g,%g,%g); only similarity transforms are supported", sx, sy, sz)
	}
	return (sx + sy + sz) / 3, nil
}
