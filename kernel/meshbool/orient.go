// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "math/big"

// Exact orientation on rational Points. The term grouping matches
// predicates.Orient3D exactly, so a Point built from float64 coordinates gives the
// identical sign to predicates.Orient3D on the same coordinates — the arrangement
// can mix original and constructed vertices without a sign-convention seam.

// Orient3D returns the exact sign of the orientation determinant of the rows
// (a-d),(b-d),(c-d): +1 if d is below plane(a,b,c) (a,b,c CCW from above), -1 if
// above, 0 if coplanar.
func Orient3D(a, b, c, d Point) int {
	return orient3DVal(a, b, c, d).Sign()
}

// orient3DVal returns the exact orientation determinant value.
func orient3DVal(a, b, c, d Point) *big.Rat {
	ad, bd, cd := a.sub(d), b.sub(d), c.sub(d)
	t1 := new(big.Rat).Mul(ad[2], crossDiff(bd[0], cd[1], cd[0], bd[1]))
	t2 := new(big.Rat).Mul(bd[2], crossDiff(cd[0], ad[1], ad[0], cd[1]))
	t3 := new(big.Rat).Mul(cd[2], crossDiff(ad[0], bd[1], bd[0], ad[1]))
	return t1.Add(t1, t2).Add(t1, t3)
}

// crossDiff returns the exact 2x2 minor p*q - r*s.
func crossDiff(p, q, r, s *big.Rat) *big.Rat {
	pq := new(big.Rat).Mul(p, q)
	rs := new(big.Rat).Mul(r, s)
	return pq.Sub(pq, rs)
}
