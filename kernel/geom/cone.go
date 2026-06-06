// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati/math"
)

// Cone is a circular cone (contract: Cone) with its Apex on the axis along
// AxisDir and the given HalfAngle (radians, between the axis and the surface).
// Parameters are (u = angle around the axis in [0,2π], v = distance from the
// apex along the axis, v ≥ 0). At distance v the radius is v·tan(HalfAngle):
// P(u,v) = Apex + v·AxisDir + v·tan(HalfAngle)·(cos u·Ref + sin u·Binormal).
type Cone struct {
	Apex      math.Point3
	AxisDir   math.UnitVector3
	Ref       math.UnitVector3
	HalfAngle float64
	binormal  math.Vector3
}

// NewCone builds a cone from an apex, axis direction, and half angle. Errors on
// a zero axis or a half angle outside the open interval (0, π/2).
func NewCone(apex math.Point3, axisDir math.Vector3, halfAngle float64) (Cone, error) {
	if halfAngle <= 0 || halfAngle >= stdmath.Pi/2 {
		return Cone{}, fmt.Errorf("geom: cone half angle %g out of range (0, π/2)", halfAngle)
	}
	a, err := math.UnitVector3FromVector(axisDir)
	if err != nil {
		return Cone{}, err
	}
	ref, bi := axisFrame(a)
	return Cone{Apex: apex, AxisDir: a, Ref: ref.AsUnit(), HalfAngle: halfAngle, binormal: bi}, nil
}

// radial returns the outward in-plane unit direction at angle u.
func (c Cone) radial(u float64) math.Vector3 {
	cos, sin := cosSin(u)
	return c.Ref.AsVector().Scale(cos).Add(c.binormal.Scale(sin))
}

// PointAt returns the point at (u, v).
func (c Cone) PointAt(u, v float64) math.Point3 {
	r := v * stdmath.Tan(c.HalfAngle)
	return c.Apex.TranslateBy(c.AxisDir.AsVector().Scale(v)).TranslateBy(c.radial(u).Scale(r))
}

// DerivativesAt returns ∂P/∂u (around) and ∂P/∂v (along the slant).
func (c Cone) DerivativesAt(u, v float64) (du, dv math.Vector3) {
	t := stdmath.Tan(c.HalfAngle)
	cos, sin := cosSin(u)
	du = c.Ref.AsVector().Scale(-v * t * sin).Add(c.binormal.Scale(v * t * cos))
	dv = c.AxisDir.AsVector().Add(c.radial(u).Scale(t))
	return du, dv
}

// NormalAt returns the outward unit normal cos(HalfAngle)·radial − sin(HalfAngle)·AxisDir.
func (c Cone) NormalAt(u, _ float64) math.Vector3 {
	cosH, sinH := cosSin(c.HalfAngle)
	return c.radial(u).Scale(cosH).Sub(c.AxisDir.AsVector().Scale(sinH))
}

// UDomain returns the periodic angular range [0, 2π].
func (c Cone) UDomain() (lo, hi float64) { return fullCircleDomain() }

// VDomain returns the half-open axial range [0, +Inf).
func (c Cone) VDomain() (lo, hi float64) { return 0, stdmath.Inf(1) }
