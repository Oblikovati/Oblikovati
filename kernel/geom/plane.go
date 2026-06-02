// SPDX-License-Identifier: GPL-2.0-only

package geom

import "github.com/Oblikovati/oblikovati/math"

// Plane is an unbounded planar surface (contract: Plane) through Origin with the
// orthonormal in-plane basis (UAxis, VAxis); its Normal is UAxis×VAxis.
// P(u, v) = Origin + u·UAxis + v·VAxis.
type Plane struct {
	Origin       math.Point3
	UAxis, VAxis math.UnitVector3
}

// NewPlane builds a plane through origin with the given normal, choosing an
// arbitrary in-plane U/V basis. Errors on a zero normal.
func NewPlane(origin math.Point3, normal math.Vector3) (Plane, error) {
	n, err := math.UnitVector3FromVector(normal)
	if err != nil {
		return Plane{}, err
	}
	ref, bi := axisFrame(n)
	return Plane{Origin: origin, UAxis: ref.AsUnit(), VAxis: bi.AsUnit()}, nil
}

// NewPlaneFromAxes builds a plane from an origin and two in-plane direction
// vectors. The V axis is orthogonalized against U (Gram-Schmidt). Errors when
// either axis is zero or the two are parallel.
func NewPlaneFromAxes(origin math.Point3, uAxis, vAxis math.Vector3) (Plane, error) {
	u, err := math.UnitVector3FromVector(uAxis)
	if err != nil {
		return Plane{}, err
	}
	vOrtho := vAxis.Sub(u.AsVector().Scale(vAxis.Dot(u.AsVector())))
	v, err := math.UnitVector3FromVector(vOrtho)
	if err != nil {
		return Plane{}, err
	}
	return Plane{Origin: origin, UAxis: u, VAxis: v}, nil
}

// Normal returns the unit plane normal UAxis×VAxis.
func (p Plane) Normal() math.Vector3 {
	return p.UAxis.Cross(p.VAxis)
}

// PointAt returns Origin + u·UAxis + v·VAxis.
func (p Plane) PointAt(u, v float64) math.Point3 {
	return p.Origin.TranslateBy(p.UAxis.AsVector().Scale(u)).TranslateBy(p.VAxis.AsVector().Scale(v))
}

// DerivativesAt returns the constant partials (UAxis, VAxis).
func (p Plane) DerivativesAt(_, _ float64) (du, dv math.Vector3) {
	return p.UAxis.AsVector(), p.VAxis.AsVector()
}

// NormalAt returns the constant unit normal.
func (p Plane) NormalAt(_, _ float64) math.Vector3 { return p.Normal() }

// UDomain returns the unbounded U range.
func (p Plane) UDomain() (lo, hi float64) { return unboundedDomain() }

// VDomain returns the unbounded V range.
func (p Plane) VDomain() (lo, hi float64) { return unboundedDomain() }
