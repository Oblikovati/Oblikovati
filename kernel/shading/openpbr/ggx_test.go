// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import (
	"math"
	"testing"
)

// TestDistributionGGXNormalIncidencePeak checks the closed-form value at the microfacet
// normal m=(0,0,1): the spec's unnormalized form D ∝ (1+tan²θm/α²)⁻² is 1 at θm=0, and
// the normalization constant 1/(π α²) is hand-derivable from the isotropic NDF integral.
func TestDistributionGGXNormalIncidencePeak(t *testing.T) {
	const alpha = 0.5
	got := DistributionGGX(Vec3{Z: 1}, alpha)
	want := 1 / (math.Pi * alpha * alpha)
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("DistributionGGX(normal, alpha=0.5) = %v, want 1/(π α²) = %v", got, want)
	}
}

// TestDistributionGGXIntegratesToOne is the NDF normalization property (∫ D(m) cosθm dωm
// = 1 over the hemisphere, the defining property of a valid microfacet distribution):
// checked numerically since there is no simpler closed form for the full integral.
func TestDistributionGGXIntegratesToOne(t *testing.T) {
	// alpha=0.1's peak is narrower than the shared 32x32 test quadrature's grid spacing
	// (the same resolution limit documented on multiscatter.go's minAlphaForMultiScatter),
	// so it needs more slack than the wider, well-resolved peaks at higher alpha.
	tolerance := map[float64]float64{0.1: 0.03, 0.3: 0.02, 0.6: 0.02, 1.0: 0.02}
	for _, alpha := range []float64{0.1, 0.3, 0.6, 1.0} {
		got := hemisphericalReflectanceScalar(func(h, _ Vec3) float64 { return DistributionGGX(h, alpha) }, 1)
		if math.Abs(got-1) > tolerance[alpha] {
			t.Errorf("alpha=%v: ∫ D(m) cosθm dωm = %v, want ≈ 1", alpha, got)
		}
	}
}

// TestSmithG1NormalIncidenceIsOne checks no self-shadowing straight up: vzSq=1 gives
// G1 = 2/(1+1) = 1 for any alpha.
func TestSmithG1NormalIncidenceIsOne(t *testing.T) {
	for _, alpha := range []float64{0.1, 0.5, 1.0} {
		if got := SmithG1(Vec3{Z: 1}, alpha); math.Abs(got-1) > 1e-12 {
			t.Errorf("alpha=%v: SmithG1(normal) = %v, want 1", alpha, got)
		}
	}
}

// TestSmithG1ApproachesZeroAtGrazing checks heavy masking near the tangent plane.
func TestSmithG1ApproachesZeroAtGrazing(t *testing.T) {
	v := Vec3{X: 0.9999, Z: 0.01}.Normalize()
	if got := SmithG1(v, 0.5); got > 0.1 {
		t.Errorf("SmithG1(near-grazing, alpha=0.5) = %v, want close to 0", got)
	}
}

// TestSmithG1ZeroAtTangentPlane checks the vzSq==0 guard.
func TestSmithG1ZeroAtTangentPlane(t *testing.T) {
	if got := SmithG1(Vec3{X: 1}, 0.5); got != 0 {
		t.Errorf("SmithG1(tangent, alpha=0.5) = %v, want exactly 0", got)
	}
}
