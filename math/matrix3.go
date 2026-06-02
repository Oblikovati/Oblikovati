// SPDX-License-Identifier: GPL-2.0-only

package math

import stdmath "math"

// Matrix3 is an immutable 3×3 affine transform in 2D (sketch) space (contract:
// Matrix2d). Cells are row-major; the bottom row is always (0, 0, 1).
type Matrix3 struct {
	m [9]Scalar
}

// Identity3 returns the 2D identity transform.
func Identity3() Matrix3 {
	return Matrix3{[9]Scalar{
		1, 0, 0,
		0, 1, 0,
		0, 0, 1,
	}}
}

// Translation3 returns a transform that displaces every point by v.
func Translation3(v Vector2) Matrix3 {
	t := Identity3()
	t.m[2], t.m[5] = v.X, v.Y
	return t
}

// Scale3 returns a transform that scales by (sx, sy) about the origin.
func Scale3(sx, sy Scalar) Matrix3 {
	t := Identity3()
	t.m[0], t.m[4] = sx, sy
	return t
}

// Rotation3 returns a transform that rotates by angle radians about center
// (counter-clockwise for positive angle).
func Rotation3(angle Scalar, center Point2) Matrix3 {
	c, s := stdmath.Cos(angle), stdmath.Sin(angle)
	// translation keeps center fixed: center − R·center.
	tx := center.X - (c*center.X - s*center.Y)
	ty := center.Y - (s*center.X + c*center.Y)
	return Matrix3{[9]Scalar{
		c, -s, tx,
		s, c, ty,
		0, 0, 1,
	}}
}

// CoordinateSystem3 maps the standard 2D frame onto the one defined by origin
// and the two axis vectors (axes become the matrix columns).
func CoordinateSystem3(origin Point2, xAxis, yAxis Vector2) Matrix3 {
	return Matrix3{[9]Scalar{
		xAxis.X, yAxis.X, origin.X,
		xAxis.Y, yAxis.Y, origin.Y,
		0, 0, 1,
	}}
}

// Translation returns the translation component.
func (t Matrix3) Translation() Vector2 {
	return Vector2{t.m[2], t.m[5]}
}

// At returns the cell at the given row and column (both 0–2).
func (t Matrix3) At(row, col int) Scalar {
	return t.m[row*3+col]
}

// Cells returns the 9 cells in row-major order.
func (t Matrix3) Cells() [9]Scalar {
	return t.m
}

// TransformPoint applies the full affine transform to p (translation included).
func (t Matrix3) TransformPoint(p Point2) Point2 {
	return Point2{
		t.m[0]*p.X + t.m[1]*p.Y + t.m[2],
		t.m[3]*p.X + t.m[4]*p.Y + t.m[5],
	}
}

// TransformVector applies only the linear part to v (translation ignored).
func (t Matrix3) TransformVector(v Vector2) Vector2 {
	return Vector2{
		t.m[0]*v.X + t.m[1]*v.Y,
		t.m[3]*v.X + t.m[4]*v.Y,
	}
}

// Mul returns the composed transform t·o: applying the result applies o first,
// then t.
func (t Matrix3) Mul(o Matrix3) Matrix3 {
	return Matrix3{mul3x3(t.m, o.m)}
}

// Determinant returns the determinant of the matrix.
func (t Matrix3) Determinant() Scalar {
	return det3(t.m)
}

// Inverse returns the inverse transform and true, or the zero matrix and false
// when the transform is singular.
func (t Matrix3) Inverse() (Matrix3, bool) {
	inv, ok := invert3x3(t.m)
	if !ok {
		return Matrix3{}, false
	}
	return Matrix3{inv}, true
}

// IsEqualTo reports whether all 9 cells are equal within tol. Pass tol <= 0 to
// use [DefaultTolerance].
func (t Matrix3) IsEqualTo(o Matrix3, tol Scalar) bool {
	tt := resolveTolerance(tol)
	for i := range t.m {
		if !approxEqual(t.m[i], o.m[i], tt) {
			return false
		}
	}
	return true
}

// IsRigid reports whether the transform preserves distances and angles: its 2×2
// linear part is orthonormal with determinant +1 (rotation/translation, no
// scale, shear, or reflection) within tol.
func (t Matrix3) IsRigid(tol Scalar) bool {
	tt := resolveTolerance(tol)
	x := Vector2{t.m[0], t.m[3]}
	y := Vector2{t.m[1], t.m[4]}
	return approxEqual(x.LengthSquared(), 1, tt) &&
		approxEqual(y.LengthSquared(), 1, tt) &&
		approxEqual(x.Dot(y), 0, tt) &&
		approxEqual(t.Determinant(), 1, tt)
}
