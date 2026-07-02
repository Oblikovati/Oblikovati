// SPDX-License-Identifier: GPL-2.0-only

package math

// Point3 is an immutable 3D position (contract: Point). It is distinct from
// [Vector3] because the two transform differently: a point is affected by the
// translation component of a [Matrix4], a vector is not.
type Point3 struct {
	X, Y, Z Scalar
}

// P3 constructs a Point3. Example: origin := P3(0, 0, 0).
func P3(x, y, z Scalar) Point3 {
	return Point3{X: x, Y: y, Z: z}
}

// TranslateBy returns this point displaced by v.
func (p Point3) TranslateBy(v Vector3) Point3 {
	return Point3{p.X + v.X, p.Y + v.Y, p.Z + v.Z}
}

// VectorTo returns the displacement from this point to o (o - this).
func (p Point3) VectorTo(o Point3) Vector3 {
	return Vector3{o.X - p.X, o.Y - p.Y, o.Z - p.Z}
}

// AsVector reinterprets the position as a displacement from the origin.
func (p Point3) AsVector() Vector3 {
	return Vector3(p)
}

// DistanceTo returns the Euclidean distance between this point and o.
func (p Point3) DistanceTo(o Point3) Scalar {
	return p.VectorTo(o).Length()
}

// DistanceSquaredTo returns the squared distance to o, avoiding the square root.
func (p Point3) DistanceSquaredTo(o Point3) Scalar {
	return p.VectorTo(o).LengthSquared()
}

// IsEqualTo reports whether this point and o coincide within tol. Pass tol <= 0
// to use [DefaultTolerance].
func (p Point3) IsEqualTo(o Point3, tol Scalar) bool {
	t := resolveTolerance(tol)
	return approxEqual(p.X, o.X, t) && approxEqual(p.Y, o.Y, t) && approxEqual(p.Z, o.Z, t)
}

// Midpoint returns the point halfway between this point and o.
func (p Point3) Midpoint(o Point3) Point3 {
	return Point3{(p.X + o.X) / 2, (p.Y + o.Y) / 2, (p.Z + o.Z) / 2}
}

// Lerp interpolates this point toward q at t, one [Lerp] per coordinate — the
// single evaluation order of #1654 (exact at t=0 and t=1).
//
//	quarter := a.Lerp(b, 0.25)
func (p Point3) Lerp(q Point3, t Scalar) Point3 {
	return P3(Lerp(p.X, q.X, t), Lerp(p.Y, q.Y, t), Lerp(p.Z, q.Z, t))
}
