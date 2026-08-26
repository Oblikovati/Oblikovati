// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import "math"

// hemisphericalReflectanceScalar numerically integrates a scalar BRDF value(wi,wo) over
// the incoming hemisphere at a fixed outgoing polar cosine, E(wo) = ∫ value(wi,wo)·cosθi
// dωi — the general form of the energy-conservation property test (PBI-340's acceptance
// criteria) applied to any lobe, not just DistributionGGX·SmithG2 (multiscatter.go's
// DirectionalAlbedoGGX is the same integral specialized to the single-scatter GGX kernel
// with Fresnel≡1). Test-only: production code never integrates at runtime.
func hemisphericalReflectanceScalar(value func(wi, wo Vec3) float64, cosThetaO float64) float64 {
	sinThetaO := math.Sqrt(1 - cosThetaO*cosThetaO)
	wo := Vec3{X: sinThetaO, Z: cosThetaO}

	const nTheta, nPhi = 32, 32
	var sum float64
	for i := range nTheta {
		thetaI := (float64(i) + 0.5) / nTheta * (math.Pi / 2)
		sinThetaI, cosThetaI := math.Sin(thetaI), math.Cos(thetaI)
		for j := range nPhi {
			phiI := (float64(j) + 0.5) / nPhi * (2 * math.Pi)
			wi := Vec3{X: sinThetaI * math.Cos(phiI), Y: sinThetaI * math.Sin(phiI), Z: cosThetaI}
			sum += value(wi, wo) * cosThetaI * sinThetaI
		}
	}
	dTheta := (math.Pi / 2) / nTheta
	dPhi := (2 * math.Pi) / nPhi
	return sum * dTheta * dPhi
}

// hemisphericalReflectanceColor is [hemisphericalReflectanceScalar] for a Color3-valued
// lobe, returning the per-channel maximum (the strictest channel for an energy bound).
func hemisphericalReflectanceColor(value func(wi, wo Vec3) Color3, cosThetaO float64) float64 {
	r := hemisphericalReflectanceScalar(func(wi, wo Vec3) float64 { return value(wi, wo).R }, cosThetaO)
	g := hemisphericalReflectanceScalar(func(wi, wo Vec3) float64 { return value(wi, wo).G }, cosThetaO)
	b := hemisphericalReflectanceScalar(func(wi, wo Vec3) float64 { return value(wi, wo).B }, cosThetaO)
	return math.Max(r, math.Max(g, b))
}
