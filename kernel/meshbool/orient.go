// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math/big"

	"oblikovati.org/kernel/predicates"
)

// Exact orientation on rational Points. The term grouping matches
// predicates.Orient3D exactly, so a Point built from float64 coordinates gives the
// identical sign to predicates.Orient3D on the same coordinates — the arrangement
// can mix original and constructed vertices without a sign-convention seam.

// Orient3D returns the exact sign of the orientation determinant of the rows
// (a-d),(b-d),(c-d): +1 if d is below plane(a,b,c) (a,b,c CCW from above), -1 if
// above, 0 if coplanar.
//
// It is filtered in two tiers before the exact path. When all four vertices are
// exact binary64 positions — the common case, since original tessellation vertices
// convert exactly — it delegates to the filtered-exact predicates.Orient3D, whose
// static float filter decides the non-degenerate majority without any big.Rat. When
// a constructed intersection vertex is involved (not a binary64), an interval filter
// (orient3DInterval) still resolves the sign in float arithmetic whenever the
// determinant is safely away from zero — the case that dominates the ray-cast
// classifier on a refined mesh. Only a near-degenerate determinant reaches the
// pure-rational path. Every tier is exact, so none can disagree with big.Rat (see
// TestOrient3DFastPathMatchesExact). This is the perf lever that makes the
// arrangement viable on refined meshes (Oblikovati#2084 coil).
func Orient3D(a, b, c, d Point) int {
	if f, ok := floatQuad(a, b, c, d); ok {
		return predicates.Orient3D(
			f[0][0], f[0][1], f[0][2], f[1][0], f[1][1], f[1][2],
			f[2][0], f[2][1], f[2][2], f[3][0], f[3][1], f[3][2])
	}
	if s, ok := orient3DInterval(a, b, c, d); ok {
		return s
	}
	return orient3DVal(a, b, c, d).Sign()
}

// floatQuad returns the four points as exact float64 triples, or ok=false as soon
// as any coordinate is a constructed rational that is not a binary64. A cheap
// power-of-two screen over all four vertices runs first, so a quad that includes even
// one constructed vertex is rejected before any expensive SetFloat64 round-trip on
// the exact vertices.
func floatQuad(a, b, c, d Point) ([4][3]float64, bool) {
	pts := [4]Point{a, b, c, d}
	for _, p := range pts {
		if !p.denomsArePowersOfTwo() {
			return [4][3]float64{}, false
		}
	}
	var f [4][3]float64
	for i, p := range pts {
		v, ok := p.float64Exact()
		if !ok {
			return [4][3]float64{}, false
		}
		f[i] = v
	}
	return f, true
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
