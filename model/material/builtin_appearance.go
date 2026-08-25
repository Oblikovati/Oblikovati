// SPDX-License-Identifier: GPL-2.0-only

package material

import "oblikovati.org/api/types"

// DefaultAppearanceID is the neutral appearance applied when a body has no material or
// appearance assigned — the resolver's last resort. It is always present (seeded here),
// so a base id always exists for [Library.DuplicateAppearance].
const DefaultAppearanceID = "default"

// defaultAppearance is the neutral gray fallback: every group at the OpenPBR spec's own
// default.
func defaultAppearance() *Appearance {
	spec := AppearanceSpec{
		DisplayName:  "Default",
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
	// Same channel values as the pre-consolidation default's "#b3b8bd" Albedo
	// (0xb3/0xb8/0xbd ÷ 255, gamma-decoded); reproduces the renderer's pre-materials
	// default gray, so an un-assigned model looks exactly as it did before this
	// subsystem.
	spec.Base.Color = Color3{R: 0.702, G: 0.722, B: 0.741}
	return NewAppearance(DefaultAppearanceID, SourceBuiltin, spec)
}
