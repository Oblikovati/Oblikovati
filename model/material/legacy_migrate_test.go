// SPDX-License-Identifier: GPL-2.0-only

package material

import "testing"

// TestLegacyAppearanceToSpecOffLobesAreZeroValue proves the migration baseline is
// correct: converting a pre-consolidation 5-scalar appearance produces every lobe
// beyond Base/Specular/Emission/Geometry at its off-state (zero weight), and a
// fully-off emissive maps to (0, black) rather than a NaN from a zero-luminance
// division.
func TestLegacyAppearanceToSpecOffLobesAreZeroValue(t *testing.T) {
	legacy := legacyAppearanceSpec{
		DisplayName: "Test", Albedo: mustColor("#804020ff"), Metallic: 0.3, Roughness: 0.6,
		Emissive: mustColor("#000000ff"), Opacity: 1,
	}
	spec := legacyAppearanceToSpec(legacy)

	if spec.Emission.Luminance != 0 || spec.Emission.Color != (Color3{}) {
		t.Errorf("fully-off emissive migrated to luminance=%v color=%+v, want 0/black", spec.Emission.Luminance, spec.Emission.Color)
	}
	if spec.Transmission.Weight != 0 || spec.Subsurface.Weight != 0 || spec.Coat.Weight != 0 ||
		spec.Fuzz.Weight != 0 || spec.ThinFilm.Weight != 0 {
		t.Error("an added lobe was not left at its off-state")
	}
	if spec.Specular.Weight == 0 {
		t.Error("specular_weight was zeroed — it is part of the mapped base reflection layer, not an added lobe")
	}
	if spec.Base.Metalness != legacy.Metallic {
		t.Errorf("base_metalness = %v, want %v", spec.Base.Metalness, legacy.Metallic)
	}
	if spec.Specular.Roughness != legacy.Roughness {
		t.Errorf("specular_roughness = %v, want %v", spec.Specular.Roughness, legacy.Roughness)
	}
	if spec.Geometry.Opacity != legacy.Opacity {
		t.Errorf("geometry_opacity = %v, want %v", spec.Geometry.Opacity, legacy.Opacity)
	}
}
