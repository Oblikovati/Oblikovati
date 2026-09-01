// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import (
	"math"
	"testing"
)

// TestLayerCoatZeroWeightReproducesBaseExactly is PBI-341's explicit regression guard:
// coat_weight=0 must reproduce PBI-340's output exactly.
func TestLayerCoatZeroWeightReproducesBaseExactly(t *testing.T) {
	t.Parallel()
	fSub := NewColor3(0.31, 0.22, 0.05)
	got := LayerCoat(0.9, fSub, Gray(0.5), 0, 1, Vec3{Z: 1}, 1.6)
	if got != fSub {
		t.Errorf("LayerCoat(weight=0) = %+v, want fSub unchanged = %+v", got, fSub)
	}
}

// TestCoatDarkeningFactorNoOpCases checks the two documented identity cases: weight=0 or
// coat_darkening=0 must apply no darkening (factor exactly 1).
func TestCoatDarkeningFactorNoOpCases(t *testing.T) {
	t.Parallel()
	if got := CoatDarkeningFactor(0.5, 0, 1, 1.6); got != 1 {
		t.Errorf("CoatDarkeningFactor(weight=0) = %v, want 1", got)
	}
	if got := CoatDarkeningFactor(0.5, 1, 0, 1.6); got != 1 {
		t.Errorf("CoatDarkeningFactor(coatDarkening=0) = %v, want 1", got)
	}
}

// TestCoatDarkeningFactorFullyReflectiveBaseIsUndarkened checks the spec's own stated
// limit (index.html line 975): as the base albedo → 1, no energy is lost so darkening → 1
// (no darkening), regardless of weight/coat_darkening.
func TestCoatDarkeningFactorFullyReflectiveBaseIsUndarkened(t *testing.T) {
	t.Parallel()
	got := CoatDarkeningFactor(1, 1, 1, 1.6)
	if math.Abs(got-1) > 1e-6 {
		t.Errorf("CoatDarkeningFactor(baseAlbedo=1) = %v, want 1 (no darkening for a fully reflective base)", got)
	}
}

// TestCoatDarkeningFactorDarkensPartlyAbsorptiveBase checks the qualitative direction of
// the effect: a base that absorbs some energy (albedo < 1) must be darkened (factor < 1)
// when coat_darkening=1.
func TestCoatDarkeningFactorDarkensPartlyAbsorptiveBase(t *testing.T) {
	t.Parallel()
	got := CoatDarkeningFactor(0.5, 1, 1, 1.6)
	if got >= 1 || got <= 0 {
		t.Errorf("CoatDarkeningFactor(baseAlbedo=0.5) = %v, want in (0,1)", got)
	}
}

// TestLayerCoatAddsCoatReflectionAtNormalIncidence sanity-checks the layer combination at
// normal incidence: the result must be at least the coat's own single-scatter reflection
// (never less, since the base term only adds energy).
func TestLayerCoatAddsCoatReflectionAtNormalIncidence(t *testing.T) {
	t.Parallel()
	up := Vec3{Z: 1}
	fCoat := SpecularCoat(up, up, 0.01, 1.6)
	fSub := Gray(0.3)
	got := LayerCoat(fCoat, fSub, Gray(1), 1, 1, up, 1.6)
	if got.R < fCoat-1e-9 {
		t.Errorf("LayerCoat result R=%v, want at least the coat term %v", got.R, fCoat)
	}
}
