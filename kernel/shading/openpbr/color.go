// SPDX-License-Identifier: GPL-2.0-only

package openpbr

import "math"

// Color pipeline (M45-F04 PBI-349, ADR-0053): OpenPBR's spec default working color
// space is ACEScg (AP1 primaries, D60 white) — see api/types.Color3's doc comment. The
// existing raster Realistic-mode shader (mesh.frag) instead authors and shades directly
// in linear sRGB, then tone-maps with a Narkowicz ACES-filmic approximation and encodes
// with a simple 2.2 gamma (toSRGB/aces in mesh.frag). Rather than diverge, the path
// tracer converts ACEScg to linear sRGB with one 3x3 matrix and then reuses that exact
// same tone-map + encode chain, so a neutral (achromatic) material renders identically
// whichever pipeline shades it, and the two visual styles never visibly disagree on
// exposure or white point.
//
// GLSL wiring of this same math into the live pathtrace/swpathtrace compute shaders is
// the F05 wiring PBI's job (PBI-350) — those shaders currently output raw linear
// radiance for their CPU-oracle cross-checks (PBI-345/346), not final display pixels.

// acesCgToLinearSRGB is the standard ACEScg (AP1, D60) -> linear sRGB (D65) 3x3 matrix
// (Academy Color Encoding System / OpenColorIO's published aces-to-srgb primaries
// conversion, including the D60->D65 Bradford chromatic adaptation). Its rows sum to
// ~1.0, so it maps an achromatic ACEScg color to the identical linear-sRGB value —
// exactly the property this PBI's grey/white parity check relies on.
var acesCgToLinearSRGB = [3][3]float64{
	{1.70505, -0.62179, -0.08326},
	{-0.13026, 1.14080, -0.01055},
	{-0.02400, -0.12897, 1.15297},
}

// ACEScgToLinearSRGB converts a Color3 from ACEScg (AP1) to linear sRGB (BT.709)
// primaries, both still scene-referred/unbounded (no tone mapping or gamma yet).
func ACEScgToLinearSRGB(c Color3) Color3 {
	m := acesCgToLinearSRGB
	return Color3{
		R: m[0][0]*c.R + m[0][1]*c.G + m[0][2]*c.B,
		G: m[1][0]*c.R + m[1][1]*c.G + m[1][2]*c.B,
		B: m[2][0]*c.R + m[2][1]*c.G + m[2][2]*c.B,
	}
}

// ACESFilmicTonemap is the Narkowicz ACES-filmic approximation — a direct Go port of
// mesh.frag's aces(), kept bit-for-bit equivalent (same constants, same clamp) so the
// two pipelines' highlight roll-off matches exactly, not just approximately.
func ACESFilmicTonemap(c Color3) Color3 {
	const a, b, cc, d, e = 2.51, 0.03, 2.43, 0.59, 0.14
	tonemap := func(x float64) float64 {
		return clamp01((x * (a*x + b)) / (x*(cc*x+d) + e))
	}
	return Color3{R: tonemap(c.R), G: tonemap(c.G), B: tonemap(c.B)}
}

// EncodeSRGB applies the same simplified 2.2-gamma display encode mesh.frag's toSRGB()
// uses (not the precise piecewise sRGB OETF) — kept identical deliberately, so this is
// an exact parity match rather than a close-but-different approximation.
func EncodeSRGB(c Color3) Color3 {
	const invGamma = 1.0 / 2.2
	encode := func(x float64) float64 { return powClamped01(x, invGamma) }
	return Color3{R: encode(c.R), G: encode(c.G), B: encode(c.B)}
}

// ToDisplay is the full ACEScg working-space color's path to a display-ready sRGB
// pixel: primaries conversion, exposure, ACES-filmic tone map, gamma encode — mirroring
// mesh.frag's `toSRGB(aces(color * scene.header.z))` exactly from the linear-sRGB step
// onward.
func ToDisplay(c Color3, exposure float64) Color3 {
	linear := ACEScgToLinearSRGB(c).Scale(exposure)
	return EncodeSRGB(ACESFilmicTonemap(linear))
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func powClamped01(x, exp float64) float64 {
	return math.Pow(clamp01(x), exp)
}
