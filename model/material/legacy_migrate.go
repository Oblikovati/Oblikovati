// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"math"

	"oblikovati.org/api/types"
)

// Legacy-to-OpenPBR migration (M46-F04): mechanical, value-preserving conversion of a
// pre-consolidation metallic-roughness appearance into an equivalent (native-shaped)
// AppearanceSpec. No longer runs at catalog-load time (the built-in catalog is natively
// OpenPBR-authored YAML, M46-F03) — it survives only for the one-time migration of old
// .obk/project-library data still holding the pre-consolidation 5-scalar shape. Every
// lobe beyond Base/Specular/Emission/Geometry (Transmission/Subsurface/Coat/Fuzz/
// ThinFilm) is left at its Go zero value, which is the spec's own off-state (every
// lobe's Weight defaults to 0) — a plain metallic-roughness appearance never had a
// coat, fuzz, transmission, or subsurface term, so the honest migration is to leave
// those OFF, not guess a value for them.

// legacyAppearanceSpec is the pre-M46 metallic-roughness appearance shape (fields
// mirror the deleted material.Appearance's old accessors), used only by the one-time
// migration of old .obk/project-library data — never constructed from live code.
type legacyAppearanceSpec struct {
	DisplayName string
	Albedo      Rgba
	Metallic    float32
	Roughness   float32
	Emissive    Rgba
	Opacity     float32
}

// legacyAppearanceToSpec converts a into the AppearanceSpec that renders the same
// surface: albedo -> base_color (gamma-decoded to linear, matching mesh.frag's own
// toLinear() so the two pipelines agree on what "the same color" means), metallic ->
// base_metalness, roughness -> specular_roughness, emissive -> emission_luminance/color,
// opacity -> geometry_opacity.
func legacyAppearanceToSpec(a legacyAppearanceSpec) AppearanceSpec {
	spec := DefaultOpenPBRAppearanceSpecForMigration()
	spec.DisplayName = a.DisplayName
	spec.Base.Color = srgbToLinearColor3(a.Albedo)
	spec.Base.Metalness = a.Metallic
	spec.Specular.Roughness = a.Roughness
	spec.Emission.Luminance, spec.Emission.Color = emissiveToLuminanceColor(a.Emissive)
	spec.Geometry.Opacity = a.Opacity
	return spec
}

// DefaultOpenPBRAppearanceSpecForMigration is the starting point legacyAppearanceToSpec
// edits: Base/Specular at the spec's own defaults (fully-weighted, Lambertian base;
// fully-weighted dielectric specular at IOR 1.5), every other lobe at its zero-value
// off-state. Exported for the migration-correctness test, which needs the exact same
// baseline to compute an independent expected spec.
func DefaultOpenPBRAppearanceSpecForMigration() AppearanceSpec {
	return AppearanceSpec{
		Base:     types.DefaultOpenPBRBase(),
		Specular: types.DefaultOpenPBRSpecular(),
		// Transmission, Subsurface, Coat, Fuzz, ThinFilm: zero value = off (Weight 0).
		Geometry: OpenPBRGeometry{Opacity: 1},
	}
}

// srgbToLinearColor3 gamma-decodes an sRGB-encoded [0,1] Rgba (the pre-consolidation
// hex-color convention) into a linear Color3, exactly matching mesh.frag's
// toLinear(c) = pow(c, 2.2) so a migrated appearance's base_color represents the
// identical linear surface color the legacy raster pipeline already rendered from the
// same hex value.
func srgbToLinearColor3(c Rgba) Color3 {
	return Color3{R: srgbToLinear(c.R), G: srgbToLinear(c.G), B: srgbToLinear(c.B)}
}

func srgbToLinear(v float32) float32 {
	return float32(math.Pow(float64(v), 2.2))
}

// emissiveToLuminanceColor decomposes a legacy sRGB emissive color into OpenPBR's
// luminance*color split: Luminance is the linear value's largest channel, Color is the
// linear value normalized by it (so Color*Luminance reconstructs the exact original
// linear RGB) — a fully off (black) emissive maps to (0, black), never a divide by
// zero.
func emissiveToLuminanceColor(c Rgba) (luminance float32, color Color3) {
	linear := srgbToLinearColor3(c)
	luminance = maxOf3(linear.R, linear.G, linear.B)
	if luminance <= 0 {
		return 0, Color3{}
	}
	return luminance, Color3{R: linear.R / luminance, G: linear.G / luminance, B: linear.B / luminance}
}

func maxOf3(a, b, c float32) float32 {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}
