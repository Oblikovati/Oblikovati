// SPDX-License-Identifier: GPL-2.0-only

package material

import "oblikovati.org/api/types"

// DefaultOpenPBRAppearanceID is the neutral OpenPBR appearance seeded into every library
// — the OpenPBR-side counterpart of [DefaultAppearanceID], present so a base id always
// exists for [Library.DuplicateOpenPBRAppearance].
const DefaultOpenPBRAppearanceID = "openpbr-default"

// defaultOpenPBRAppearance is the neutral gray fallback: every group at the OpenPBR spec's
// own default, with Base.Color set to the same neutral gray as [defaultAppearance] so the
// two defaults read as the same surface.
func defaultOpenPBRAppearance() *OpenPBRAppearance {
	spec := OpenPBRAppearanceSpec{
		DisplayName:  "Default (OpenPBR)",
		Base:         types.DefaultOpenPBRBase(),
		Specular:     types.DefaultOpenPBRSpecular(),
		Transmission: types.DefaultOpenPBRTransmission(),
		Subsurface:   types.DefaultOpenPBRSubsurface(),
		Coat:         types.DefaultOpenPBRCoat(),
		Fuzz:         types.DefaultOpenPBRFuzz(),
		ThinFilm:     types.DefaultOpenPBRThinFilm(),
		Emission:     types.DefaultOpenPBREmission(),
		Geometry:     types.DefaultOpenPBRGeometry(),
	}
	// Same channel values as defaultAppearance's "#b3b8bd" Albedo (0xb3/0xb8/0xbd ÷ 255); a
	// proper sRGB→ACEScg conversion is PBI-349's color-pipeline job, not this seed value.
	spec.Base.Color = Color3{R: 0.702, G: 0.722, B: 0.741}
	return NewOpenPBRAppearance(DefaultOpenPBRAppearanceID, SourceBuiltin, spec)
}
