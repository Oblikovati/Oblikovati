// SPDX-License-Identifier: GPL-2.0-only

package math

// Point2 is an immutable 2D position (contract: Point2d), used in sketch space.
// Distinct from [Vector2]: only a point is affected by the translation part of
// a [Matrix3].
type Point2 struct {
	X, Y Scalar
}

// P2 constructs a Point2. Example: origin := P2(0, 0).
func P2(x, y Scalar) Point2 {
	return Point2{X: x, Y: y}
}

// TranslateBy returns this point displaced by v.
func (p Point2) TranslateBy(v Vector2) Point2 {
	return Point2{p.X + v.X, p.Y + v.Y}
}

// VectorTo returns the displacement from this point to o (o - this).
func (p Point2) VectorTo(o Point2) Vector2 {
	return Vector2{o.X - p.X, o.Y - p.Y}
}

// AsVector reinterprets the position as a displacement from the origin.
func (p Point2) AsVector() Vector2 {
	return Vector2(p)
}

// DistanceTo returns the Euclidean distance between this point and o.
func (p Point2) DistanceTo(o Point2) Scalar {
	return p.VectorTo(o).Length()
}

// DistanceSquaredTo returns the squared distance to o, avoiding the square root.
func (p Point2) DistanceSquaredTo(o Point2) Scalar {
	return p.VectorTo(o).LengthSquared()
}

// IsEqualTo reports whether this point and o coincide within tol. Pass tol <= 0
// to use [DefaultTolerance].
func (p Point2) IsEqualTo(o Point2, tol Scalar) bool {
	t := resolveTolerance(tol)
	return approxEqual(p.X, o.X, t) && approxEqual(p.Y, o.Y, t)
}

// Midpoint returns the point halfway between this point and o.
func (p Point2) Midpoint(o Point2) Point2 {
	return Point2{(p.X + o.X) / 2, (p.Y + o.Y) / 2}
}

// Lerp interpolates this point toward q at t, one [Lerp] per coordinate — the
// single evaluation order of #1654 (exact at t=0 and t=1).
//
//	quarter := a.Lerp(b, 0.25)
func (p Point2) Lerp(q Point2, t Scalar) Point2 {
	return P2(Lerp(p.X, q.X, t), Lerp(p.Y, q.Y, t))
}
