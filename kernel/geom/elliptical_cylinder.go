// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati/math"
)

// EllipticalCylinder is an infinite cylinder of elliptical cross-section (contract:
// EllipticalCylinder) about the axis line through Origin along AxisDir. Ref is the in-plane
// major-axis direction (perpendicular to AxisDir) with semi-axis MajorRadius; the minor
// semi-axis MinorRadius runs along AxisDir×Ref. Parameters are (u = angle in [0,2π], v =
// signed distance along the axis):
// P(u,v) = Origin + v·AxisDir + MajorRadius·cos u·Ref + MinorRadius·sin u·(AxisDir×Ref).
type EllipticalCylinder struct {
	Origin      math.Point3
	AxisDir     math.UnitVector3
	Ref         math.UnitVector3
	MajorRadius float64
	MinorRadius float64
	binormal    math.Vector3
}

// NewEllipticalCylinder builds an elliptical cylinder from an axis point and direction, the
// major-axis direction (projected onto the plane perpendicular to the axis), and the two
// semi-radii. Errors on a zero axis direction.
func NewEllipticalCylinder(origin math.Point3, axisDir, majorAxis math.Vector3, majorR, minorR float64) (EllipticalCylinder, error) {
	a, err := math.UnitVector3FromVector(axisDir)
	if err != nil {
		return EllipticalCylinder{}, err
	}
	ref := planarRef(a, majorAxis)
	return EllipticalCylinder{
		Origin: origin, AxisDir: a, Ref: ref, MajorRadius: majorR, MinorRadius: minorR,
		binormal: a.Cross(ref),
	}, nil
}

// PointAt returns the point at (u, v).
func (c EllipticalCylinder) PointAt(u, v float64) math.Point3 {
	cos, sin := cosSin(u)
	radial := c.Ref.AsVector().Scale(c.MajorRadius * cos).Add(c.binormal.Scale(c.MinorRadius * sin))
	return c.Origin.TranslateBy(c.AxisDir.AsVector().Scale(v)).TranslateBy(radial)
}

// DerivativesAt returns ∂P/∂u (around the ellipse) and ∂P/∂v (the axis direction).
func (c EllipticalCylinder) DerivativesAt(u, _ float64) (du, dv math.Vector3) {
	cos, sin := cosSin(u)
	du = c.Ref.AsVector().Scale(-c.MajorRadius * sin).Add(c.binormal.Scale(c.MinorRadius * cos))
	return du, c.AxisDir.AsVector()
}

// NormalAt returns the outward unit normal (du×dv normalized).
func (c EllipticalCylinder) NormalAt(u, v float64) math.Vector3 {
	du, dv := c.DerivativesAt(u, v)
	return normalFromPartials(du, dv)
}

// UDomain returns the periodic angular range [0, 2π].
func (c EllipticalCylinder) UDomain() (lo, hi float64) { return fullCircleDomain() }

// VDomain returns the unbounded axial range.
func (c EllipticalCylinder) VDomain() (lo, hi float64) { return unboundedDomain() }

// ParamAt inverts PointAt: the angle of the ellipse and the signed distance along the
// axis. The radii rescale the projected components before the atan2 so an on-surface point
// reproduces its u exactly.
func (c EllipticalCylinder) ParamAt(q math.Point3) (u, v float64) {
	d := c.Origin.VectorTo(q)
	v = d.Dot(c.AxisDir.AsVector())
	r := d.Sub(c.AxisDir.AsVector().Scale(v))
	return wrap2pi(stdmath.Atan2(r.Dot(c.binormal)/c.MinorRadius, r.Dot(c.Ref.AsVector())/c.MajorRadius)), v
}

var _ Surface = EllipticalCylinder{}
