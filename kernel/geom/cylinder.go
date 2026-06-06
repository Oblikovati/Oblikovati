// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati/math"

// Cylinder is an infinite circular cylinder (contract: Cylinder) about the axis
// line through Origin along AxisDir, with the given Radius. Parameters are
// (u = angle around the axis in [0,2π], v = signed distance along the axis):
// P(u,v) = Origin + v·AxisDir + Radius·(cos u·Ref + sin u·Binormal).
type Cylinder struct {
	Origin   math.Point3
	AxisDir  math.UnitVector3
	Ref      math.UnitVector3
	Radius   float64
	binormal math.Vector3
}

// NewCylinder builds a cylinder from an axis point, axis direction, and radius.
// Errors on a zero axis direction.
func NewCylinder(origin math.Point3, axisDir math.Vector3, radius float64) (Cylinder, error) {
	a, err := math.UnitVector3FromVector(axisDir)
	if err != nil {
		return Cylinder{}, err
	}
	ref, bi := axisFrame(a)
	return Cylinder{Origin: origin, AxisDir: a, Ref: ref.AsUnit(), Radius: radius, binormal: bi}, nil
}

// PointAt returns the point at (u, v).
func (c Cylinder) PointAt(u, v float64) math.Point3 {
	radial := c.radial(u)
	return c.Origin.TranslateBy(c.AxisDir.AsVector().Scale(v)).TranslateBy(radial.Scale(c.Radius))
}

// radial returns the outward unit direction at angle u: cos u·Ref + sin u·Binormal.
func (c Cylinder) radial(u float64) math.Vector3 {
	cos, sin := cosSin(u)
	return c.Ref.AsVector().Scale(cos).Add(c.binormal.Scale(sin))
}

// DerivativesAt returns ∂P/∂u (tangent around) and ∂P/∂v (the axis direction).
func (c Cylinder) DerivativesAt(u, _ float64) (du, dv math.Vector3) {
	cos, sin := cosSin(u)
	du = c.Ref.AsVector().Scale(-c.Radius * sin).Add(c.binormal.Scale(c.Radius * cos))
	return du, c.AxisDir.AsVector()
}

// NormalAt returns the outward radial unit normal at angle u.
func (c Cylinder) NormalAt(u, _ float64) math.Vector3 { return c.radial(u) }

// UDomain returns the periodic angular range [0, 2π].
func (c Cylinder) UDomain() (lo, hi float64) { return fullCircleDomain() }

// VDomain returns the unbounded axial range.
func (c Cylinder) VDomain() (lo, hi float64) { return unboundedDomain() }
