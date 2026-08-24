// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import (
	"math"
	"testing"
)

// TestLayerFuzzZeroWeightReproducesCoatedBaseExactly is PBI-341's explicit regression
// guard: fuzz_weight=0 must reproduce the coat-only output exactly.
func TestLayerFuzzZeroWeightReproducesCoatedBaseExactly(t *testing.T) {
	coatedBase := NewColor3(0.31, 0.22, 0.05)
	up := Vec3{Z: 1}
	got := LayerFuzz(up, up, 0.5, Gray(1), 0, coatedBase)
	if got != coatedBase {
		t.Errorf("LayerFuzz(weight=0) = %+v, want coatedBase unchanged = %+v", got, coatedBase)
	}
}

// TestSpecularFuzzZeroBelowSurface checks the below-surface guard.
func TestSpecularFuzzZeroBelowSurface(t *testing.T) {
	got := SpecularFuzz(Vec3{Z: -1}, Vec3{Z: 1}, 0.5, Gray(1))
	if got != (Color3{}) {
		t.Errorf("SpecularFuzz(wi below surface) = %+v, want zero", got)
	}
}

// TestFuzzScalarAlbedoBounds checks the physical bound across a roughness sweep.
func TestFuzzScalarAlbedoBounds(t *testing.T) {
	for _, roughness := range []float64{0.05, 0.3, 0.5, 0.8, 1.0} {
		for _, cosTheta := range []float64{0.1, 0.5, 1.0} {
			e := fuzzScalarAlbedo(cosTheta, roughness)
			if e < 0 || e > 1.05 {
				t.Errorf("roughness=%v cos=%v: fuzzScalarAlbedo = %v, want in [0,1]", roughness, cosTheta, e)
			}
		}
	}
}

// TestLayerFuzzTintsWithFuzzColor checks that a colored fuzz measurably shifts the result
// toward its own color (not just adding an achromatic sheen). Charlie's sheen lobe is
// zero at exact normal incidence (sinθh=0, the grazing-angle effect sheen models), so wi
// and wo are both tilted off-normal here to land where the lobe actually contributes.
func TestLayerFuzzTintsWithFuzzColor(t *testing.T) {
	// wi and wo tilt the SAME direction by different amounts, so their half-vector isn't
	// exactly the macronormal (unlike a symmetric tilt, whose x/y components cancel).
	wi := Vec3{X: 0.9, Z: math.Sqrt(1 - 0.81)}
	wo := Vec3{Z: 1}
	coatedBase := Gray(0.2)
	red := NewColor3(1, 0.05, 0.05)
	got := LayerFuzz(wi, wo, 0.3, red, 1, coatedBase)
	if got.R <= got.G || got.R <= got.B {
		t.Errorf("LayerFuzz with red fuzz_color = %+v, want R channel to dominate", got)
	}
}
