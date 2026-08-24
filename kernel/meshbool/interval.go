// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math"
	"math/big"
)

// An interval float filter for the orientation predicate on ARBITRARY rational
// Points — the constructed intersection vertices that dominate a refined mesh, which
// are not binary64 and so otherwise force the pure-rational (big.Rat) path. big.Rat
// normalizes every operation with an expensive GCD; on the Oblikovati#2084 coil that
// GCD is ~56% of the whole boolean, all inside the ray-cast classifier.
//
// The idea is standard (Shewchuk-style filtering): bracket every true coordinate by
// a float interval that provably contains it, propagate the intervals through the
// determinant with outward rounding, and read the sign off the result interval. When
// that interval excludes zero the sign is certain and no big.Rat runs; only a
// near-degenerate determinant (interval straddling zero) falls back to the exact
// path. The result is never a tolerance decision — an uncertain filter defers to the
// exact predicate, so the answer is identical to pure big.Rat (proven by
// TestOrient3DFastPathMatchesExact over exact, constructed and near-coplanar quads).

// interval is a closed float64 range [lo, hi] that provably contains the exact real
// value it stands for.
type interval struct{ lo, hi float64 }

// widenLo and widenHi move a just-computed endpoint outward by one ulp. A binary64
// operation rounds to nearest, so its result is within ½ ulp (< 1 ulp) of the exact
// value; stepping one ulp outward therefore keeps the exact value inside the bound.
func widenLo(x float64) float64 { return math.Nextafter(x, math.Inf(-1)) }
func widenHi(x float64) float64 { return math.Nextafter(x, math.Inf(1)) }

// iSub returns an interval containing a-b for every a in the first and b in the
// second: the extremes are a.lo-b.hi and a.hi-b.lo, each widened outward.
func iSub(a, b interval) interval {
	return interval{widenLo(a.lo - b.hi), widenHi(a.hi - b.lo)}
}

// iAdd returns an interval containing a+b.
func iAdd(a, b interval) interval {
	return interval{widenLo(a.lo + b.lo), widenHi(a.hi + b.hi)}
}

// iMul returns an interval containing a*b. A product of two ranges is bounded by the
// four corner products; the true extreme is their min and max. Each product is
// forced through rounded() so the compiler cannot fuse it with a neighbouring add
// into an FMA (Oblikovati#2020), which would defeat the per-operation ulp widening.
func iMul(a, b interval) interval {
	p1 := rounded(a.lo * b.lo)
	p2 := rounded(a.lo * b.hi)
	p3 := rounded(a.hi * b.lo)
	p4 := rounded(a.hi * b.hi)
	return interval{
		widenLo(min(min(p1, p2), min(p3, p4))),
		widenHi(max(max(p1, p2), max(p3, p4))),
	}
}

// rounded forces its (already binary64) argument through an explicit conversion so
// the Go compiler cannot fuse a preceding multiply with a following add/sub into an
// FMA (Oblikovati#2020: Go fuses a*b+c on arm64 but not amd64). The interval widening
// assumes each multiply is separately rounded.
func rounded(x float64) float64 { return float64(x) }

// coordInterval brackets one exact rational coordinate by a float interval. A
// coordinate that is exactly a binary64 collapses to a point; otherwise the exact
// value lies strictly between the neighbouring floats of its round-to-nearest image
// (round-to-nearest keeps it within ½ ulp), so [prevfloat, nextfloat] contains it.
func coordInterval(c *big.Rat) interval {
	f, exact := c.Float64()
	if exact {
		return interval{f, f}
	}
	return interval{widenLo(f), widenHi(f)}
}

// intervalsOf brackets a point's three coordinates.
func intervalsOf(p Point) [3]interval {
	return [3]interval{coordInterval(p.X), coordInterval(p.Y), coordInterval(p.Z)}
}

// orient3DInterval evaluates the orientation determinant in interval arithmetic,
// following the exact term grouping of orient3DVal. It returns the sign and true
// when the result interval excludes zero (the sign is then certain); otherwise it
// returns false and the caller must use the exact predicate.
func orient3DInterval(a, b, c, d Point) (int, bool) {
	ai, bi, ci, di := intervalsOf(a), intervalsOf(b), intervalsOf(c), intervalsOf(d)
	ad := subInterval(ai, di)
	bd := subInterval(bi, di)
	cd := subInterval(ci, di)
	t1 := iMul(ad[2], crossDiffInterval(bd[0], cd[1], cd[0], bd[1]))
	t2 := iMul(bd[2], crossDiffInterval(cd[0], ad[1], ad[0], cd[1]))
	t3 := iMul(cd[2], crossDiffInterval(ad[0], bd[1], bd[0], ad[1]))
	det := iAdd(iAdd(t1, t2), t3)
	switch {
	case det.lo > 0:
		return 1, true
	case det.hi < 0:
		return -1, true
	default:
		return 0, false
	}
}

// subInterval returns the componentwise a-b of two coordinate triples.
func subInterval(a, b [3]interval) [3]interval {
	return [3]interval{iSub(a[0], b[0]), iSub(a[1], b[1]), iSub(a[2], b[2])}
}

// crossDiffInterval returns an interval containing the 2x2 minor p*q - r*s.
func crossDiffInterval(p, q, r, s interval) interval {
	return iSub(iMul(p, q), iMul(r, s))
}
