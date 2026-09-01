// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import (
	"math"
	"testing"
)

// grayParityTolerance is the documented tolerance for PBI-349's grey/white parity
// check: the ACEScg->linear-sRGB matrix's rows sum to 1 only up to the precision of its
// published constants (row 2 sums to 0.99999, not exactly 1), so an achromatic value
// carried through it picks up a tiny relative error before tone mapping even starts.
const grayParityTolerance = 1e-4

// TestACEScgToLinearSRGBPreservesAchromaticValues checks the matrix's defining property
// for this PBI: a grey/white ACEScg color must map to (approximately) the identical
// linear-sRGB value, channel for channel — the whole reason the two pipelines can agree
// on a neutral reference material.
func TestACEScgToLinearSRGBPreservesAchromaticValues(t *testing.T) {
	t.Parallel()
	for _, v := range []float64{0, 0.04, 0.18, 0.5, 1.0, 2.5} {
		got := ACEScgToLinearSRGB(Gray(v))
		for name, ch := range map[string]float64{"R": got.R, "G": got.G, "B": got.B} {
			if diff := math.Abs(ch - v); diff > grayParityTolerance {
				t.Errorf("Gray(%v).%s = %v after ACEScgToLinearSRGB, want ~%v (diff %v > %v)", v, name, ch, v, diff, grayParityTolerance)
			}
		}
	}
}

// TestACEScgToLinearSRGBIsChromatic confirms the matrix is NOT just an identity in
// general — a saturated color must actually change primaries, or the "conversion"
// would be a no-op that defeats the point of ACEScg working space.
func TestACEScgToLinearSRGBIsChromatic(t *testing.T) {
	t.Parallel()
	red := NewColor3(1, 0, 0)
	got := ACEScgToLinearSRGB(red)
	if got.G == 0 && got.B == 0 {
		t.Errorf("ACEScgToLinearSRGB(pure ACEScg red) = %+v, want cross-talk into G/B (AP1 red is not BT.709 red)", got)
	}
}

// TestACESFilmicTonemapMatchesMeshFragBehavior pins the Go port's numeric behavior to
// the same properties mesh.frag's aces() has: 0 stays 0, output never exceeds 1, and it
// is monotonically increasing over the working HDR range (no highlight banding/reversal).
func TestACESFilmicTonemapMatchesMeshFragBehavior(t *testing.T) {
	t.Parallel()
	if got := ACESFilmicTonemap(Gray(0)); got != (Color3{}) {
		t.Errorf("ACESFilmicTonemap(0) = %+v, want zero", got)
	}
	prev := 0.0
	for _, v := range []float64{0.01, 0.1, 0.5, 1, 2, 5, 20, 100} {
		got := ACESFilmicTonemap(Gray(v)).R
		if got < prev {
			t.Errorf("ACESFilmicTonemap not monotonic at v=%v: got %v < previous %v", v, got, prev)
		}
		if got > 1 || got < 0 {
			t.Errorf("ACESFilmicTonemap(%v) = %v, want in [0,1]", v, got)
		}
		prev = got
	}
}

func TestEncodeSRGBBoundaryValues(t *testing.T) {
	t.Parallel()
	if got := EncodeSRGB(Gray(0)); got != (Color3{}) {
		t.Errorf("EncodeSRGB(0) = %+v, want zero", got)
	}
	got := EncodeSRGB(Gray(1))
	if math.Abs(got.R-1) > 1e-9 {
		t.Errorf("EncodeSRGB(1).R = %v, want 1", got.R)
	}
	// mesh.frag's toSRGB clamps its input to [0,1] before the gamma curve, so an
	// out-of-range (HDR, already-tonemapped-incorrectly-or-not) input must not produce
	// an out-of-range or NaN output.
	if got := EncodeSRGB(Gray(3)); got.R > 1 || math.IsNaN(got.R) {
		t.Errorf("EncodeSRGB(3).R = %v, want clamped to <=1 and finite", got.R)
	}
}

// TestToDisplayGrayWhiteMatchesRasterPipeline is PBI-349's headline acceptance
// criterion: a calibrated grey/white reference material must render to the same
// display RGB whether shaded by the old raster pipeline (which authors and shades
// directly in linear sRGB, tone-mapping with mesh.frag's aces()+toSRGB()) or the new
// OpenPBR path tracer (which authors in ACEScg, converts to linear sRGB, then reuses
// that same tone-map+encode chain) — within grayParityTolerance, documented above.
func TestToDisplayGrayWhiteMatchesRasterPipeline(t *testing.T) {
	t.Parallel()
	rasterPipeline := func(v, exposure float64) Color3 {
		return EncodeSRGB(ACESFilmicTonemap(Gray(v).Scale(exposure)))
	}
	for _, v := range []float64{0.18, 0.5, 0.9, 1.0, 2.0} {
		for _, exposure := range []float64{0.5, 1.0, 1.5} {
			raster := rasterPipeline(v, exposure)
			pathTraced := ToDisplay(Gray(v), exposure)
			for name, pair := range map[string][2]float64{
				"R": {raster.R, pathTraced.R}, "G": {raster.G, pathTraced.G}, "B": {raster.B, pathTraced.B},
			} {
				if diff := math.Abs(pair[0] - pair[1]); diff > grayParityTolerance {
					t.Errorf("v=%v exposure=%v channel %s: raster=%v path-traced=%v, diff %v > %v",
						v, exposure, name, pair[0], pair[1], diff, grayParityTolerance)
				}
			}
		}
	}
}
