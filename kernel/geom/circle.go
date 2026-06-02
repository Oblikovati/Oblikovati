// SPDX-License-Identifier: GPL-2.0-only

package geom

import "github.com/Oblikovati/oblikovati/math"

// Circle is a full 3D circle (contract: Circle) lying in the plane through
// Center with the given unit Normal. RefDir is a unit vector in that plane
// marking parameter angle 0; the circle is parameterized t∈[0,1] mapping to
// angle 2πt, counter-clockwise about Normal.
type Circle struct {
	Center math.Point3
	Normal math.UnitVector3
	RefDir math.UnitVector3
	Radius float64
}

// NewCircle builds a circle from a center, plane normal, and radius, choosing an
// arbitrary in-plane reference direction. It errors when normal is zero length.
func NewCircle(center math.Point3, normal math.Vector3, radius float64) (Circle, error) {
	n, err := math.UnitVector3FromVector(normal)
	if err != nil {
		return Circle{}, err
	}
	return Circle{Center: center, Normal: n, RefDir: perpendicularUnit(n), Radius: radius}, nil
}

// binormal returns the in-plane unit vector perpendicular to RefDir (Normal ×
// RefDir), completing the right-handed evaluation basis.
func (c Circle) binormal() math.Vector3 {
	return c.Normal.Cross(c.RefDir)
}

// PointAt returns the point at angle 2πt.
func (c Circle) PointAt(t float64) math.Point3 {
	return pointOnCircle(c.Center, c.RefDir.AsVector(), c.binormal(), c.Radius, twoPi*t)
}

// TangentAt returns the derivative dP/dt at parameter t.
func (c Circle) TangentAt(t float64) math.Vector3 {
	return circleTangent(c.RefDir.AsVector(), c.binormal(), c.Radius, twoPi*t).Scale(twoPi)
}

// Domain returns [0, 1].
func (c Circle) Domain() (lo, hi float64) { return 0, 1 }

// Circumference returns 2πr.
func (c Circle) Circumference() float64 { return twoPi * c.Radius }
