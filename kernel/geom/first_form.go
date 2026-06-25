// SPDX-License-Identifier: GPL-2.0-only

package geom

// The first fundamental form of a surface at (u, v) is the 2×2 metric
// [[a, b], [b, c]] with a = Sᵤ·Sᵤ, b = Sᵤ·Sᵥ, c = Sᵥ·Sᵥ. Its determinant
// det = a·c − b² carries units of length⁴, so a FIXED cutoff on det is
// scale-blind: the same patch is judged "degenerate" at mm scale and well-formed
// at km scale (#1402). The scale-invariant conditioning measure is the squared
// sine of the angle θ between the two tangents — purely geometric, unit-free.

// firstFormRelTol is the relative floor on sin²θ below which the tangent frame is
// treated as degenerate (tangents collapsed or parallel). It is a pure ratio, so
// it means the same thing at every model scale, unlike an absolute length⁴ cutoff.
const firstFormRelTol = 1e-12

// firstFormParallelity returns sin²θ = 1 − (Sᵤ·Sᵥ)²/((Sᵤ·Sᵤ)(Sᵥ·Sᵥ)) — the
// dimensionless squared sine of the angle between the surface tangents (1 when
// orthogonal, 0 when parallel or collapsed). By Cauchy–Schwarz b² ≤ a·c, so the
// value lies in [0, 1]; a tiny negative from roundoff near parallel is clamped to
// 0. Returns 0 when either tangent has zero length (a·c ≤ 0).
//
//	firstFormParallelity(1, 0, 1) // 1.0 — orthogonal unit tangents
//	firstFormParallelity(1, 1, 1) // 0.0 — identical tangents, degenerate
func firstFormParallelity(a, b, c float64) float64 {
	ac := a * c
	if ac <= 0 {
		return 0
	}
	s := 1 - b*b/ac
	if s < 0 {
		return 0
	}
	return s
}

// degenerateFirstForm reports whether the first fundamental form [[a,b],[b,c]] is
// degenerate — its tangents parallel or collapsed — by the scale-invariant
// parallelity falling below firstFormRelTol. Replaces the legacy absolute cutoffs
// on the length⁴ determinant (#1402), which misjudged tiny or huge patches.
func degenerateFirstForm(a, b, c float64) bool {
	return firstFormParallelity(a, b, c) < firstFormRelTol
}
