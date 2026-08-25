// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import (
	"math"
	"testing"
)

// TestFresnelWithThinFilmZeroWeightReproducesPlainFresnel is PBI-342's explicit
// regression guard: thin_film_weight=0 must reproduce the plain (PBI-341) Fresnel term
// exactly.
func TestFresnelWithThinFilmZeroWeightReproducesPlainFresnel(t *testing.T) {
	want := Gray(DielectricFresnel(1.5, 0.7))
	got := FresnelWithThinFilm(0.7, 1.5, 1.4, 0.5, 0)
	if got != want {
		t.Errorf("FresnelWithThinFilm(weight=0) = %+v, want plain Fresnel %+v", got, want)
	}
}

func TestThinFilmReflectanceZeroThicknessIsZero(t *testing.T) {
	got := ThinFilmReflectanceDielectric(0.8, 1, 1.4, 1.5, 0)
	if got != (Color3{}) {
		t.Errorf("ThinFilmReflectanceDielectric(thickness=0) = %+v, want zero", got)
	}
}

// TestThinFilmReflectanceBounded checks the physical bound (a power reflectance can't
// exceed 1) across a sweep of thickness and incidence angle.
func TestThinFilmReflectanceBounded(t *testing.T) {
	for _, thickness := range []float64{0.1, 0.3, 0.5, 0.8, 1.2} {
		for _, cosTheta := range []float64{0.05, 0.3, 0.6, 1.0} {
			got := ThinFilmReflectanceDielectric(cosTheta, 1, 1.4, 1.5, thickness)
			for _, c := range []float64{got.R, got.G, got.B} {
				if c < 0 || c > 1.001 {
					t.Errorf("thickness=%v cos=%v: reflectance = %+v, want channels in [0,1]", thickness, cosTheta, got)
				}
			}
		}
	}
}

// TestThinFilmReflectanceShowsColorShift is PBI-342's core acceptance criterion: at a
// representative thickness/angle, interference must separate the R/G/B channels
// measurably — the signature of iridescence — not collapse to a gray (achromatic) result,
// which would indicate the Airy summation degenerated into a plain, wavelength-independent
// Fresnel value.
func TestThinFilmReflectanceShowsColorShift(t *testing.T) {
	// Sample across several thicknesses rather than pinning one: the R/G/B spread itself
	// oscillates (TestThinFilmReflectanceOscillatesWithThickness), so at least one of a
	// representative spread of thicknesses must land near a fringe with strong color
	// separation for a genuinely working Airy summation.
	maxSpread := 0.0
	for _, thickness := range []float64{0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8} {
		got := ThinFilmReflectanceDielectric(0.9, 1, 1.4, 1.5, thickness)
		spread := math.Max(got.R, math.Max(got.G, got.B)) - math.Min(got.R, math.Min(got.G, got.B))
		maxSpread = math.Max(maxSpread, spread)
	}
	if maxSpread < 0.02 {
		t.Errorf("max R/G/B spread across a 0.2-0.8µm thickness sweep = %v, want a measurable spread (iridescence) somewhere in range", maxSpread)
	}
}

// TestThinFilmReflectanceOscillatesWithThickness checks the other defining signature of
// thin-film interference: unlike a plain Fresnel term (angle-dependent only), reflectance
// at a FIXED angle must vary non-monotonically as thickness sweeps through several
// interference fringes.
func TestThinFilmReflectanceOscillatesWithThickness(t *testing.T) {
	const cosTheta = 1.0
	var g []float64
	for thicknessNM := 50.0; thicknessNM <= 2000; thicknessNM += 25 {
		got := ThinFilmReflectanceDielectric(cosTheta, 1, 1.4, 1.5, thicknessNM/1000)
		g = append(g, got.G)
	}
	risingToFalling, fallingToRising := 0, 0
	for i := 2; i < len(g); i++ {
		prevDelta := g[i-1] - g[i-2]
		delta := g[i] - g[i-1]
		if prevDelta > 0 && delta < 0 {
			risingToFalling++
		}
		if prevDelta < 0 && delta > 0 {
			fallingToRising++
		}
	}
	if risingToFalling < 2 || fallingToRising < 2 {
		t.Errorf("G channel over a 50nm-2000nm thickness sweep had %d peaks and %d troughs, want several of each (interference fringes)",
			risingToFalling, fallingToRising)
	}
}

// TestThinFilmReflectanceTotalInternalReflectionIsWhite checks the TIR short-circuit: at
// grazing incidence into a higher-IOR film, the film's own outer interface totally
// internally reflects, and the whole result must saturate toward white (all channels 1,
// scaled by the thin-film presence multiplier).
func TestThinFilmReflectanceTotalInternalReflectionIsWhite(t *testing.T) {
	// eta_exterior > eta_film forces sin(theta_t) >= 1 at a shallow enough incidence.
	got := ThinFilmReflectanceDielectric(0.05, 2.0, 1.0, 1.5, 0.5)
	if got.R < 0.95 || got.G < 0.95 || got.B < 0.95 {
		t.Errorf("ThinFilmReflectanceDielectric under TIR = %+v, want ≈ white", got)
	}
}
