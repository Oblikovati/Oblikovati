// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// Sphere is a sphere surface (contract: Sphere) of the given Center and Radius.
// Parameters are (u = longitude in [0,2π], v = latitude in [−π/2, π/2]):
// P(u,v) = Center + R·(cos v cos u, cos v sin u, sin v).
type Sphere struct {
	Center math.Point3
	Radius float64
}

// NewSphere builds a sphere; errors on a non-positive radius.
func NewSphere(center math.Point3, radius float64) (Sphere, error) {
	if radius <= 0 {
		return Sphere{}, fmt.Errorf("geom: sphere radius %g must be > 0", radius)
	}
	return Sphere{Center: center, Radius: radius}, nil
}

// direction returns the outward unit direction at (u, v) — the surface normal.
func (s Sphere) direction(u, v float64) math.Vector3 {
	cu, su := cosSin(u)
	cv, sv := cosSin(v)
	return math.V3(cv*cu, cv*su, sv)
}

// PointAt returns the point at (u, v).
func (s Sphere) PointAt(u, v float64) math.Point3 {
	return s.Center.TranslateBy(s.direction(u, v).Scale(s.Radius))
}

// DerivativesAt returns ∂P/∂u (eastward) and ∂P/∂v (northward).
func (s Sphere) DerivativesAt(u, v float64) (du, dv math.Vector3) {
	cu, su := cosSin(u)
	cv, sv := cosSin(v)
	du = math.V3(-s.Radius*cv*su, s.Radius*cv*cu, 0)
	dv = math.V3(-s.Radius*sv*cu, -s.Radius*sv*su, s.Radius*cv)
	return du, dv
}

// NormalAt returns the outward radial unit normal (defined even at the poles).
func (s Sphere) NormalAt(u, v float64) math.Vector3 { return s.direction(u, v) }

// UDomain returns the periodic longitude range [0, 2π].
func (s Sphere) UDomain() (lo, hi float64) { return fullCircleDomain() }

// VDomain returns the latitude range [−π/2, π/2].
func (s Sphere) VDomain() (lo, hi float64) { return -stdmath.Pi / 2, stdmath.Pi / 2 }
