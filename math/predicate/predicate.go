// SPDX-License-Identifier: GPL-2.0-only

package predicate

import (
	stdmath "math"

	"oblikovati.org/math"
)

// errScale is a conservative multiple of the float64 unit roundoff (2⁻⁵³). When a
// determinant's magnitude exceeds errScale·(magnitude of its terms), the float64
// sign is trustworthy; otherwise the exact path decides.
const errScale = 1e-14

// Orient2D returns a value with the sign of the signed area of triangle (a, b, c):
// positive if a→b→c turns counter-clockwise, negative if clockwise, exactly zero
// if collinear. The sign is always correct.
func Orient2D(a, b, c math.Point2) float64 {
	left := (a.X - c.X) * (b.Y - c.Y)
	right := (a.Y - c.Y) * (b.X - c.X)
	det := left - right
	bound := errScale * (stdmath.Abs(left) + stdmath.Abs(right))
	if det > bound || -det > bound {
		return det
	}
	return float64(orient2DExact(a, b, c))
}

// Orient3D returns a value with the sign of the signed volume of tetrahedron
// (a, b, c, d): positive if d is below the plane a→b→c by the right-hand rule,
// negative if above, zero if coplanar. The sign is always correct.
func Orient3D(a, b, c, d math.Point3) float64 {
	det, mag := orient3DFloat(a, b, c, d)
	bound := errScale * mag
	if det > bound || -det > bound {
		return det
	}
	return float64(orient3DExact(a, b, c, d))
}

// InCircle returns a value with the sign telling whether d lies inside the circle
// through (a, b, c): positive if inside (for a CCW triangle), negative if outside,
// zero if cocircular. The sign is always correct.
func InCircle(a, b, c, d math.Point2) float64 {
	det, mag := inCircleFloat(a, b, c, d)
	bound := errScale * mag
	if det > bound || -det > bound {
		return det
	}
	return float64(inCircleExact(a, b, c, d))
}

// orient3DFloat computes the 3×3 determinant of (a−d, b−d, c−d) in float64 and a
// magnitude estimate of its terms for the error bound.
func orient3DFloat(a, b, c, d math.Point3) (det, mag float64) {
	ax, ay, az := a.X-d.X, a.Y-d.Y, a.Z-d.Z
	bx, by, bz := b.X-d.X, b.Y-d.Y, b.Z-d.Z
	cx, cy, cz := c.X-d.X, c.Y-d.Y, c.Z-d.Z
	t1 := ax * (by*cz - bz*cy)
	t2 := bx * (ay*cz - az*cy)
	t3 := cx * (ay*bz - az*by)
	return t1 - t2 + t3, stdmath.Abs(t1) + stdmath.Abs(t2) + stdmath.Abs(t3)
}

// inCircleFloat computes the in-circle determinant in float64 plus a term magnitude.
func inCircleFloat(a, b, c, d math.Point2) (det, mag float64) {
	adx, ady := a.X-d.X, a.Y-d.Y
	bdx, bdy := b.X-d.X, b.Y-d.Y
	cdx, cdy := c.X-d.X, c.Y-d.Y
	alift := adx*adx + ady*ady
	blift := bdx*bdx + bdy*bdy
	clift := cdx*cdx + cdy*cdy
	t1 := alift * (bdx*cdy - cdx*bdy)
	t2 := blift * (adx*cdy - cdx*ady)
	t3 := clift * (adx*bdy - bdx*ady)
	return t1 - t2 + t3, stdmath.Abs(t1) + stdmath.Abs(t2) + stdmath.Abs(t3)
}
