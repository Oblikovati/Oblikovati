// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import "math"

// AlphaFromRoughness maps the user-facing roughness r ∈ [0,1] to the GGX NDF's α
// parameter (spec eq. GGX, "Following [Burley2012]..."): α = r², chosen for a
// perceptually more linear roughness response than α = r.
func AlphaFromRoughness(roughness float64) float64 { return roughness * roughness }

// DistributionGGX is the isotropic GGX (Trowbridge-Reitz) normal distribution function,
// evaluated at the half-vector h in local shading space (h.Z = cos of the angle to the
// macrosurface normal) — exact port of Adobe's openpbr_eval_aniso_ggx with alpha.x =
// alpha.y = alpha (the isotropic case).
func DistributionGGX(h Vec3, alpha float64) float64 {
	denom := sq(h.X/alpha) + sq(h.Y/alpha) + sq(h.Z)
	return 1 / (math.Pi * alpha * alpha * denom * denom)
}

// SmithG1 is the (isotropic) Smith masking/shadowing term for one direction v in local
// shading space — exact port of Adobe's openpbr_eval_aniso_smith_g1 with alpha.x =
// alpha.y = alpha, using the cancellation-avoiding algebraic form the reference notes is
// numerically stable at low roughness/near-grazing angles. v.Z may be negative
// (transmission paths); the caller decides whether that's meaningful.
func SmithG1(v Vec3, alpha float64) float64 {
	vzSq := v.Z * v.Z
	if vzSq == 0 {
		return 0
	}
	return 2 / (1 + math.Sqrt(1+(sq(alpha*v.X)+sq(alpha*v.Y))/vzSq))
}

// SmithG2 is the separable (uncorrelated) Smith joint masking-shadowing term for the
// incident/outgoing direction pair — exact port of Adobe's openpbr_eval_aniso_smith_g2
// (G1(v1) * G1(v2); the reference does not use the height-correlated joint form here).
func SmithG2(wi, wo Vec3, alpha float64) float64 { return SmithG1(wi, alpha) * SmithG1(wo, alpha) }
