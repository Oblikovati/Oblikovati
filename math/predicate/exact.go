// SPDX-License-Identifier: GPL-2.0-only

package predicate

import (
	"math/big"

	"oblikovati/math"
)

// The exact paths evaluate the same determinants in exact rational arithmetic
// (float64 values are exact rationals via big.Rat.SetFloat64), so the returned
// sign is definitive. They run only when the float64 result is too close to zero
// to trust — rare — so the big.Rat cost is acceptable.

func rat(x float64) *big.Rat { return new(big.Rat).SetFloat64(x) }

// sub returns a−b as a new rational.
func sub(a, b *big.Rat) *big.Rat { return new(big.Rat).Sub(a, b) }

// mul returns a·b as a new rational.
func mul(a, b *big.Rat) *big.Rat { return new(big.Rat).Mul(a, b) }

func orient2DExact(a, b, c math.Point2) int {
	ax, ay := rat(a.X), rat(a.Y)
	bx, by := rat(b.X), rat(b.Y)
	cx, cy := rat(c.X), rat(c.Y)
	left := mul(sub(ax, cx), sub(by, cy))
	right := mul(sub(ay, cy), sub(bx, cx))
	return sub(left, right).Sign()
}

func orient3DExact(a, b, c, d math.Point3) int {
	ax, ay, az := sub(rat(a.X), rat(d.X)), sub(rat(a.Y), rat(d.Y)), sub(rat(a.Z), rat(d.Z))
	bx, by, bz := sub(rat(b.X), rat(d.X)), sub(rat(b.Y), rat(d.Y)), sub(rat(b.Z), rat(d.Z))
	cx, cy, cz := sub(rat(c.X), rat(d.X)), sub(rat(c.Y), rat(d.Y)), sub(rat(c.Z), rat(d.Z))
	t1 := mul(ax, sub(mul(by, cz), mul(bz, cy)))
	t2 := mul(bx, sub(mul(ay, cz), mul(az, cy)))
	t3 := mul(cx, sub(mul(ay, bz), mul(az, by)))
	return sub(new(big.Rat).Add(t1, t3), t2).Sign()
}

func inCircleExact(a, b, c, d math.Point2) int {
	adx, ady := sub(rat(a.X), rat(d.X)), sub(rat(a.Y), rat(d.Y))
	bdx, bdy := sub(rat(b.X), rat(d.X)), sub(rat(b.Y), rat(d.Y))
	cdx, cdy := sub(rat(c.X), rat(d.X)), sub(rat(c.Y), rat(d.Y))
	alift := new(big.Rat).Add(mul(adx, adx), mul(ady, ady))
	blift := new(big.Rat).Add(mul(bdx, bdx), mul(bdy, bdy))
	clift := new(big.Rat).Add(mul(cdx, cdx), mul(cdy, cdy))
	t1 := mul(alift, sub(mul(bdx, cdy), mul(cdx, bdy)))
	t2 := mul(blift, sub(mul(adx, cdy), mul(cdx, ady)))
	t3 := mul(clift, sub(mul(adx, bdy), mul(bdx, ady)))
	return sub(new(big.Rat).Add(t1, t3), t2).Sign()
}
