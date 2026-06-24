// SPDX-License-Identifier: GPL-2.0-only

package predicate

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Per-predicate static float filters. Each is a small multiple of the unit roundoff u = 2⁻⁵³, set
// safely ABOVE Shewchuk's PROVEN A-level forward-error bounds for these adaptive determinants
// (orient2d (3+16u)u, orient3d (7+56u)u, incircle (10+96u)u; "Robust Adaptive Floating-Point
// Geometric Predicates", 1997) yet far below the previous uniform 1e-14 ≈ 45u. Being above the proven
// bound keeps the sign exact whenever the float magnitude clears it; being below 1e-14 sends only the
// genuinely uncertain (near-degenerate) determinants to the exact big.Rat path instead of every
// modestly small one (#1323 L4). Correctness is unchanged — only how often the slow path runs.
const (
	unitRoundoff   = 1.1102230246251565e-16                // u = 2⁻⁵³
	orient2DFilter = (3 + 16*unitRoundoff) * unitRoundoff  // Shewchuk ccwerrboundA ≈ 3.3e-16
	orient3DFilter = (7 + 56*unitRoundoff) * unitRoundoff  // Shewchuk o3derrboundA ≈ 7.8e-16
	inCircleFilter = (10 + 96*unitRoundoff) * unitRoundoff // Shewchuk iccerrboundA ≈ 1.1e-15
)

// Orient2D returns a value with the sign of the signed area of triangle (a, b, c):
// positive if a→b→c turns counter-clockwise, negative if clockwise, exactly zero
// if collinear. The sign is always correct.
func Orient2D(a, b, c math.Point2) float64 {
	left := (a.X - c.X) * (b.Y - c.Y)
	right := (a.Y - c.Y) * (b.X - c.X)
	det := left - right
	bound := orient2DFilter * (stdmath.Abs(left) + stdmath.Abs(right))
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
	bound := orient3DFilter * mag
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
	bound := inCircleFilter * mag
	if det > bound || -det > bound {
		return det
	}
	return float64(inCircleExact(a, b, c, d))
}

// orient3DFloat computes the 3×3 determinant of (a−d, b−d, c−d) in float64 and Shewchuk's PERMANENT
// for it — the sum of the absolute products BEFORE the minors' internal subtraction can cancel. (A
// plain |t₁|+|t₂|+|t₃| over the cofactor terms underestimates the rounding error when a minor nearly
// cancels, which is why the orient3DFilter must pair with this permanent, not that — #1323 L4.)
func orient3DFloat(a, b, c, d math.Point3) (det, perm float64) {
	adx, ady, adz := float64(a.X-d.X), float64(a.Y-d.Y), float64(a.Z-d.Z)
	bdx, bdy, bdz := float64(b.X-d.X), float64(b.Y-d.Y), float64(b.Z-d.Z)
	cdx, cdy, cdz := float64(c.X-d.X), float64(c.Y-d.Y), float64(c.Z-d.Z)
	bdxcdy, cdxbdy := bdx*cdy, cdx*bdy
	cdxady, adxcdy := cdx*ady, adx*cdy
	adxbdy, bdxady := adx*bdy, bdx*ady
	det = adz*(bdxcdy-cdxbdy) + bdz*(cdxady-adxcdy) + cdz*(adxbdy-bdxady)
	perm = (stdmath.Abs(bdxcdy)+stdmath.Abs(cdxbdy))*stdmath.Abs(adz) +
		(stdmath.Abs(cdxady)+stdmath.Abs(adxcdy))*stdmath.Abs(bdz) +
		(stdmath.Abs(adxbdy)+stdmath.Abs(bdxady))*stdmath.Abs(cdz)
	return det, perm
}

// inCircleFloat computes the in-circle determinant in float64 plus Shewchuk's permanent for it. The
// lifts (adx²+ady², …) are non-negative, so the permanent weights each cross-product magnitude by the
// lift, again capturing the rounding error before the minor subtractions cancel.
func inCircleFloat(a, b, c, d math.Point2) (det, perm float64) {
	adx, ady := float64(a.X-d.X), float64(a.Y-d.Y)
	bdx, bdy := float64(b.X-d.X), float64(b.Y-d.Y)
	cdx, cdy := float64(c.X-d.X), float64(c.Y-d.Y)
	alift := adx*adx + ady*ady
	blift := bdx*bdx + bdy*bdy
	clift := cdx*cdx + cdy*cdy
	bdxcdy, cdxbdy := bdx*cdy, cdx*bdy
	cdxady, adxcdy := cdx*ady, adx*cdy
	adxbdy, bdxady := adx*bdy, bdx*ady
	det = alift*(bdxcdy-cdxbdy) + blift*(cdxady-adxcdy) + clift*(adxbdy-bdxady)
	perm = (stdmath.Abs(bdxcdy)+stdmath.Abs(cdxbdy))*alift +
		(stdmath.Abs(cdxady)+stdmath.Abs(adxcdy))*blift +
		(stdmath.Abs(adxbdy)+stdmath.Abs(bdxady))*clift
	return det, perm
}
