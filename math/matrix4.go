// SPDX-License-Identifier: GPL-2.0-only

package math

// Matrix4 is an immutable 4×4 affine transform in 3D space (contract: Matrix).
// Cells are stored row-major; the bottom row is always (0, 0, 0, 1) — Matrix4
// represents affine transforms (rotation, translation, scale, mirror, and their
// compositions), not projective ones. Perspective lives in the renderer layer
// in float32, never here.
type Matrix4 struct {
	m [16]Scalar
}

// linearBlock extracts the upper-left 3×3 (rotation/scale/shear) part.
func (t Matrix4) linearBlock() [9]Scalar {
	return [9]Scalar{
		t.m[0], t.m[1], t.m[2],
		t.m[4], t.m[5], t.m[6],
		t.m[8], t.m[9], t.m[10],
	}
}

// Identity4 returns the identity transform.
func Identity4() Matrix4 {
	return Matrix4{[16]Scalar{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}}
}

// Matrix4FromCells reconstructs a transform from its 16 row-major cells — the
// inverse of [Matrix4.Cells], used to restore a persisted transform (Inventor's
// PutMatrixData). The caller is responsible for the cells describing an affine
// transform (bottom row (0,0,0,1)); no normalization is applied.
func Matrix4FromCells(cells [16]Scalar) Matrix4 {
	return Matrix4{cells}
}

// Translation4 returns a transform that displaces every point by v.
func Translation4(v Vector3) Matrix4 {
	t := Identity4()
	t.m[3], t.m[7], t.m[11] = v.X, v.Y, v.Z
	return t
}

// Scale4 returns a transform that scales by (sx, sy, sz) about the origin.
func Scale4(sx, sy, sz Scalar) Matrix4 {
	t := Identity4()
	t.m[0], t.m[5], t.m[10] = sx, sy, sz
	return t
}

// Translation returns the translation component of the transform.
func (t Matrix4) Translation() Vector3 {
	return Vector3{t.m[3], t.m[7], t.m[11]}
}

// At returns the cell at the given row and column (both 0–3).
func (t Matrix4) At(row, col int) Scalar {
	return t.m[row*4+col]
}

// Cells returns the 16 cells in row-major order (the form used for
// serialization and the public contract's GetMatrixData).
func (t Matrix4) Cells() [16]Scalar {
	return t.m
}

// TransformPoint applies the full affine transform to p (translation included).
func (t Matrix4) TransformPoint(p Point3) Point3 {
	return Point3{
		t.m[0]*p.X + t.m[1]*p.Y + t.m[2]*p.Z + t.m[3],
		t.m[4]*p.X + t.m[5]*p.Y + t.m[6]*p.Z + t.m[7],
		t.m[8]*p.X + t.m[9]*p.Y + t.m[10]*p.Z + t.m[11],
	}
}

// TransformVector applies only the linear part to v (translation ignored,
// because a displacement has no position).
func (t Matrix4) TransformVector(v Vector3) Vector3 {
	return Vector3{
		t.m[0]*v.X + t.m[1]*v.Y + t.m[2]*v.Z,
		t.m[4]*v.X + t.m[5]*v.Y + t.m[6]*v.Z,
		t.m[8]*v.X + t.m[9]*v.Y + t.m[10]*v.Z,
	}
}

// TransformUnitVector transforms u and renormalizes the result. It returns an
// error when the transform collapses the direction to zero length (e.g. a
// degenerate scale).
func (t Matrix4) TransformUnitVector(u UnitVector3) (UnitVector3, error) {
	return UnitVector3FromVector(t.TransformVector(u.AsVector()))
}

// Mul returns the composed transform t·o: applying the result is the same as
// applying o first, then t.
func (t Matrix4) Mul(o Matrix4) Matrix4 {
	var out [16]Scalar
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			out[r*4+c] = t.m[r*4]*o.m[c] + t.m[r*4+1]*o.m[4+c] +
				t.m[r*4+2]*o.m[8+c] + t.m[r*4+3]*o.m[12+c]
		}
	}
	return Matrix4{out}
}

// Determinant returns the determinant of the linear part, which equals the
// determinant of the whole affine matrix (the bottom row is 0,0,0,1).
func (t Matrix4) Determinant() Scalar {
	return det3(t.linearBlock())
}

// Inverse returns the inverse transform and true, or the zero matrix and false
// when the transform is singular (non-invertible linear part). For an affine
// matrix [L|d], the inverse is [L⁻¹ | −L⁻¹d].
func (t Matrix4) Inverse() (Matrix4, bool) {
	inv, ok := invert3x3(t.linearBlock())
	if !ok {
		return Matrix4{}, false
	}
	d := t.Translation()
	return Matrix4{[16]Scalar{
		inv[0], inv[1], inv[2], -(inv[0]*d.X + inv[1]*d.Y + inv[2]*d.Z),
		inv[3], inv[4], inv[5], -(inv[3]*d.X + inv[4]*d.Y + inv[5]*d.Z),
		inv[6], inv[7], inv[8], -(inv[6]*d.X + inv[7]*d.Y + inv[8]*d.Z),
		0, 0, 0, 1,
	}}, true
}

// IsEqualTo reports whether all 16 cells are equal within tol. Pass tol <= 0 to
// use [DefaultTolerance].
func (t Matrix4) IsEqualTo(o Matrix4, tol Scalar) bool {
	tt := resolveTolerance(tol)
	for i := range t.m {
		if !approxEqual(t.m[i], o.m[i], tt) {
			return false
		}
	}
	return true
}

// IsRigid reports whether the transform preserves distances and angles: its
// linear part is orthonormal and its determinant is +1 (a pure rotation/
// translation, no scale, shear, or reflection) within tol.
func (t Matrix4) IsRigid(tol Scalar) bool {
	tt := resolveTolerance(tol)
	x := Vector3{t.m[0], t.m[4], t.m[8]}
	y := Vector3{t.m[1], t.m[5], t.m[9]}
	z := Vector3{t.m[2], t.m[6], t.m[10]}
	return columnsOrthonormal(x, y, z, tt) && approxEqual(t.Determinant(), 1, tt)
}

// columnsOrthonormal reports whether the three basis vectors are mutually
// perpendicular and unit length within tol.
func columnsOrthonormal(x, y, z Vector3, tol Scalar) bool {
	return approxEqual(x.LengthSquared(), 1, tol) &&
		approxEqual(y.LengthSquared(), 1, tol) &&
		approxEqual(z.LengthSquared(), 1, tol) &&
		approxEqual(x.Dot(y), 0, tol) &&
		approxEqual(y.Dot(z), 0, tol) &&
		approxEqual(z.Dot(x), 0, tol)
}
