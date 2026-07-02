// SPDX-License-Identifier: GPL-2.0-only

package math

import stdmath "math"

// Rotation4 returns a transform that rotates by angle radians about the line
// through center along axis (right-hand rule). Uses the Rodrigues formula on
// the unit axis, then conjugates by center so the axis line stays fixed.
func Rotation4(angle Scalar, axis UnitVector3, center Point3) Matrix4 {
	r := rodrigues(angle, axis)
	rc := transformLinear(r, center.AsVector())
	return Matrix4{[16]Scalar{
		r[0], r[1], r[2], center.X - rc.X,
		r[3], r[4], r[5], center.Y - rc.Y,
		r[6], r[7], r[8], center.Z - rc.Z,
		0, 0, 0, 1,
	}}
}

// rodrigues returns the row-major 3×3 rotation matrix for a rotation of angle
// radians about the given unit axis.
func rodrigues(angle Scalar, axis UnitVector3) [9]Scalar {
	c, s := stdmath.Cos(angle), stdmath.Sin(angle)
	t := 1 - c
	x, y, z := axis.X(), axis.Y(), axis.Z()
	return [9]Scalar{
		t*x*x + c, t*x*y - s*z, t*x*z + s*y,
		t*x*y + s*z, t*y*y + c, t*y*z - s*x,
		t*x*z - s*y, t*y*z + s*x, t*z*z + c,
	}
}

// transformLinear applies a row-major 3×3 matrix to v.
func transformLinear(m [9]Scalar, v Vector3) Vector3 {
	return Vector3{
		m[0]*v.X + m[1]*v.Y + m[2]*v.Z,
		m[3]*v.X + m[4]*v.Y + m[5]*v.Z,
		m[6]*v.X + m[7]*v.Y + m[8]*v.Z,
	}
}

// CoordinateSystem4 returns the transform that maps the standard coordinate
// system onto the one defined by origin and the three axis vectors (the axes
// become the matrix columns). It is the modern form of the contract's
// SetCoordinateSystem. The axes need not be unit length, allowing scaled or
// sheared frames.
func CoordinateSystem4(origin Point3, xAxis, yAxis, zAxis Vector3) Matrix4 {
	return Matrix4{[16]Scalar{
		xAxis.X, yAxis.X, zAxis.X, origin.X,
		xAxis.Y, yAxis.Y, zAxis.Y, origin.Y,
		xAxis.Z, yAxis.Z, zAxis.Z, origin.Z,
		0, 0, 0, 1,
	}}
}

// Reflection4 returns the mirror transform across the plane through origin with
// the given unit normal: reflect(p) = p − 2((p−origin)·n)n. Its determinant is −1
// (orientation-reversing), which a body-transform op uses to flip face winding so
// normals stay outward. Used by MirrorFeature and the mirror pattern.
func Reflection4(origin Point3, normal UnitVector3) Matrix4 {
	n := normal.AsVector()
	d := 2 * origin.AsVector().Dot(n) // translation component = 2(o·n)n
	return Matrix4{[16]Scalar{
		1 - 2*n.X*n.X, -2 * n.X * n.Y, -2 * n.X * n.Z, d * n.X,
		-2 * n.Y * n.X, 1 - 2*n.Y*n.Y, -2 * n.Y * n.Z, d * n.Y,
		-2 * n.Z * n.X, -2 * n.Z * n.Y, 1 - 2*n.Z*n.Z, d * n.Z,
		0, 0, 0, 1,
	}}
}

// RotateBetween returns the shortest-arc rotation that aligns direction from
// with direction to (both about the origin). When the two are antiparallel it
// rotates π about an arbitrary perpendicular axis, since the axis is otherwise
// undetermined.
func RotateBetween(from, to UnitVector3) Matrix4 {
	d := Clamp(from.Dot(to), -1, 1)
	if approxEqual(d, 1, AngleTolerance) {
		return Identity4()
	}
	if approxEqual(d, -1, AngleTolerance) {
		return Rotation4(stdmath.Pi, anyPerpendicular(from), P3(0, 0, 0))
	}
	axis, _ := UnitVector3FromVector(from.Cross(to))
	return Rotation4(stdmath.Acos(d), axis, P3(0, 0, 0))
}

// anyPerpendicular returns some unit vector perpendicular to u. It crosses u
// with whichever world axis is least aligned with it, so the cross product is
// never near zero.
func anyPerpendicular(u UnitVector3) UnitVector3 {
	seed := V3(1, 0, 0)
	if stdmath.Abs(u.X()) > 0.9 {
		seed = V3(0, 1, 0)
	}
	perp, _ := UnitVector3FromVector(u.Cross(seed.AsUnit()))
	return perp
}

// AsUnit normalizes v, panicking only on the programmer error of a zero vector
// used as a fixed world seed; callers here pass constant non-zero axes.
func (v Vector3) AsUnit() UnitVector3 {
	u, err := UnitVector3FromVector(v)
	if err != nil {
		panic(err)
	}
	return u
}
