// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// Torus is a torus surface (contract: Torus) centered at Center about AxisDir.
// MajorRadius is the distance from the center to the tube center; MinorRadius
// is the tube radius. Parameters are (u = angle around the axis, v = angle
// around the tube), both in [0,2π]:
// P(u,v) = Center + (Major + Minor·cos v)·Radial(u) + Minor·sin v·AxisDir.
type Torus struct {
	Center      math.Point3
	AxisDir     math.UnitVector3
	Ref         math.UnitVector3
	MajorRadius float64
	MinorRadius float64
	binormal    math.Vector3
}

// NewTorus builds a torus; errors on a zero axis or a non-positive radius.
func NewTorus(center math.Point3, axisDir math.Vector3, majorR, minorR float64) (Torus, error) {
	if majorR <= 0 || minorR <= 0 {
		return Torus{}, fmt.Errorf("geom: torus radii (major %g, minor %g) must be > 0", majorR, minorR)
	}
	a, err := math.UnitVector3FromVector(axisDir)
	if err != nil {
		return Torus{}, err
	}
	ref, bi := axisFrame(a)
	return Torus{Center: center, AxisDir: a, Ref: ref.AsUnit(), MajorRadius: majorR, MinorRadius: minorR, binormal: bi}, nil
}

// NewTorusWithRef builds a torus whose angle-zero reference is refHint (projected perpendicular to
// the axis) rather than an arbitrary axis-derived frame — so a toroidal fillet face shares the frame
// of its neighbouring cylinder/cap/circle faces and the seams line up at angle 0
// (Oblikovati/Oblikovati#129). Mirrors NewCylinderWithRef / NewConeWithRef.
func NewTorusWithRef(center math.Point3, axisDir, refHint math.Vector3, majorR, minorR float64) (Torus, error) {
	if majorR <= 0 || minorR <= 0 {
		return Torus{}, fmt.Errorf("geom: torus radii (major %g, minor %g) must be > 0", majorR, minorR)
	}
	a, err := math.UnitVector3FromVector(axisDir)
	if err != nil {
		return Torus{}, err
	}
	av := a.AsVector()
	ref, err := math.UnitVector3FromVector(refHint.Sub(av.Scale(refHint.Dot(av))))
	if err != nil {
		return Torus{}, fmt.Errorf("geom: torus refHint %v is parallel to axis %v", refHint, axisDir)
	}
	return Torus{Center: center, AxisDir: a, Ref: ref, MajorRadius: majorR, MinorRadius: minorR, binormal: a.Cross(ref)}, nil
}

// radial returns the in-plane unit direction toward the tube center at angle u.
func (t Torus) radial(u float64) math.Vector3 {
	cos, sin := cosSin(u)
	return t.Ref.AsVector().Scale(cos).Add(t.binormal.Scale(sin))
}

// PointAt returns the point at (u, v).
func (t Torus) PointAt(u, v float64) math.Point3 {
	cv, sv := cosSin(v)
	planar := t.radial(u).Scale(t.MajorRadius + t.MinorRadius*cv)
	return t.Center.TranslateBy(planar).TranslateBy(t.AxisDir.AsVector().Scale(t.MinorRadius * sv))
}

// DerivativesAt returns ∂P/∂u (around the axis) and ∂P/∂v (around the tube).
func (t Torus) DerivativesAt(u, v float64) (du, dv math.Vector3) {
	cu, su := cosSin(u)
	cv, sv := cosSin(v)
	dRadial := t.Ref.AsVector().Scale(-su).Add(t.binormal.Scale(cu))
	du = dRadial.Scale(t.MajorRadius + t.MinorRadius*cv)
	dv = t.radial(u).Scale(-t.MinorRadius * sv).Add(t.AxisDir.AsVector().Scale(t.MinorRadius * cv))
	return du, dv
}

// NormalAt returns the outward unit normal cos v·Radial(u) + sin v·AxisDir.
func (t Torus) NormalAt(u, v float64) math.Vector3 {
	cv, sv := cosSin(v)
	return t.radial(u).Scale(cv).Add(t.AxisDir.AsVector().Scale(sv))
}

// UDomain returns the periodic range [0, 2π] around the axis.
func (t Torus) UDomain() (lo, hi float64) { return fullCircleDomain() }

// VDomain returns the periodic range [0, 2π] around the tube.
func (t Torus) VDomain() (lo, hi float64) { return fullCircleDomain() }
