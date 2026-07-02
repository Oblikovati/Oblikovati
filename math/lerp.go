// SPDX-License-Identifier: GPL-2.0-only

package math

// Lerp interpolates a→b at t — the ONE evaluation order for the whole kernel
// (#1654). The codebase had forked into a+(b−a)*t and a+t*(b−a) spellings;
// in a B-rep kernel two packages interpolating the "same" point along a shared
// edge must agree to the last bit, so every lerp now routes through here.
//
// Form: the fused a + t*(b−a) (bit-identical to both historical spellings —
// IEEE-754 multiplication commutes), which is exact at t=0 and monotone, but
// not always exact at t=1 (a + (b−a) can round past b, e.g. a=0.1, b=0.3).
// The t==1 pin restores endpoint exactness, mirroring C++ std::lerp (P0811).
//
//	mid := math.Lerp(lo, hi, 0.5)
func Lerp(a, b, t Scalar) Scalar {
	if t == 1 {
		return b
	}
	return a + t*(b-a)
}
