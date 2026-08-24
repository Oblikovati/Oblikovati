// SPDX-License-Identifier: GPL-2.0-only

package predicates

import "math/big"

// The exact path. Every binary64 coordinate converts to a big.Rat with no
// rounding (a float is a dyadic rational, exactly representable), so the sign of
// the rational determinant is the exact mathematical sign of the predicate. This
// is the whole correctness guarantee; it runs only when the static filter in
// filter.go cannot certify the floating-point sign.

// ratOf converts a finite binary64 to its exact rational value. Coordinates are
// finite geometry positions by precondition; SetFloat64 returns nil only for
// NaN/Inf, which would be a caller bug upstream.
func ratOf(x float64) *big.Rat {
	return new(big.Rat).SetFloat64(x)
}

// ratDiff returns a-b exactly.
func ratDiff(a, b float64) *big.Rat {
	return new(big.Rat).Sub(ratOf(a), ratOf(b))
}

// crossDiff returns p*q - r*s exactly, the 2x2 minor these predicates are built
// from.
func crossDiff(p, q, r, s *big.Rat) *big.Rat {
	pq := new(big.Rat).Mul(p, q)
	rs := new(big.Rat).Mul(r, s)
	return pq.Sub(pq, rs)
}

// exactOrient2D returns the exact sign of the 2D orientation determinant
// (ax-cx)(by-cy) - (ay-cy)(bx-cx): +1 CCW, -1 CW, 0 collinear.
func exactOrient2D(ax, ay, bx, by, cx, cy float64) int {
	det := crossDiff(ratDiff(ax, cx), ratDiff(by, cy), ratDiff(ay, cy), ratDiff(bx, cx))
	return det.Sign()
}

// exactOrient3D returns the exact sign of the 3D orientation determinant of the
// rows (a-d), (b-d), (c-d), using the same term grouping as the filter so the two
// paths share one sign convention: +1 if d is below plane(a,b,c) (a,b,c CCW from
// above), -1 if above, 0 coplanar.
func exactOrient3D(ax, ay, az, bx, by, bz, cx, cy, cz, dx, dy, dz float64) int {
	adx, ady, adz := ratDiff(ax, dx), ratDiff(ay, dy), ratDiff(az, dz)
	bdx, bdy, bdz := ratDiff(bx, dx), ratDiff(by, dy), ratDiff(bz, dz)
	cdx, cdy, cdz := ratDiff(cx, dx), ratDiff(cy, dy), ratDiff(cz, dz)

	t1 := new(big.Rat).Mul(adz, crossDiff(bdx, cdy, cdx, bdy))
	t2 := new(big.Rat).Mul(bdz, crossDiff(cdx, ady, adx, cdy))
	t3 := new(big.Rat).Mul(cdz, crossDiff(adx, bdy, bdx, ady))

	det := t1.Add(t1, t2)
	det.Add(det, t3)
	return det.Sign()
}
