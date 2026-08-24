// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"math"
	"testing"
)

// TestCatalogAppearancesHaveMigratedOpenPBRTwin is PBI-352's acceptance criterion: every
// built-in catalog Appearance must have a same-id OpenPBRAppearance whose mapped fields
// exactly reproduce the source's rendered color/metalness/roughness/emissive/opacity,
// with every lobe beyond Base/Specular/Emission/Geometry left at its off-state (zero
// weight) — an independently-recomputed expected value (not a call back into
// migrateAppearance/srgbToLinear, which would make the test tautological), matching
// mesh.frag's own toLinear(c) = pow(c, 2.2) gamma decode exactly.
func TestCatalogAppearancesHaveMigratedOpenPBRTwin(t *testing.T) {
	lib := NewLibrary()
	appearances := lib.Appearances()
	if len(appearances) == 0 {
		t.Fatal("catalog produced no built-in appearances — nothing to migrate")
	}

	for _, a := range appearances {
		if a.ID() == DefaultAppearanceID {
			continue // the synthetic neutral default isn't a catalog/*.yaml entry — OpenPBR already has its own (DefaultOpenPBRAppearanceID)
		}
		t.Run(a.ID(), func(t *testing.T) {
			p, ok := lib.OpenPBRAppearance(a.ID())
			if !ok {
				t.Fatalf("no migrated OpenPBRAppearance for catalog appearance %q", a.ID())
			}
			if p.Source() != SourceBuiltin {
				t.Errorf("migrated twin source = %q, want builtin", p.Source())
			}

			wantColor := independentLinear(a.Albedo())
			gotColor := p.Base().Color
			assertColorClose(t, "base_color", gotColor, wantColor)

			if p.Base().Metalness != a.Metallic() {
				t.Errorf("base_metalness = %v, want metallic %v", p.Base().Metalness, a.Metallic())
			}
			if p.Specular().Roughness != a.Roughness() {
				t.Errorf("specular_roughness = %v, want roughness %v", p.Specular().Roughness, a.Roughness())
			}
			if p.Geometry().Opacity != a.Opacity() {
				t.Errorf("geometry_opacity = %v, want opacity %v", p.Geometry().Opacity, a.Opacity())
			}

			wantEmissive := independentLinear(a.Emissive())
			gotLum, gotCol := p.Emission().Luminance, p.Emission().Color
			gotEmissive := Color3{R: gotCol.R * gotLum, G: gotCol.G * gotLum, B: gotCol.B * gotLum}
			assertColorClose(t, "emission_luminance*emission_color", gotEmissive, wantEmissive)

			// Every lobe the legacy appearance never had must be OFF (zero weight) — not
			// guessed at, not defaulted to "on".
			if p.Transmission().Weight != 0 {
				t.Errorf("transmission_weight = %v, want 0 (off)", p.Transmission().Weight)
			}
			if p.Subsurface().Weight != 0 {
				t.Errorf("subsurface_weight = %v, want 0 (off)", p.Subsurface().Weight)
			}
			if p.Coat().Weight != 0 {
				t.Errorf("coat_weight = %v, want 0 (off)", p.Coat().Weight)
			}
			if p.Fuzz().Weight != 0 {
				t.Errorf("fuzz_weight = %v, want 0 (off)", p.Fuzz().Weight)
			}
			if p.ThinFilm().Weight != 0 {
				t.Errorf("thin_film_weight = %v, want 0 (off)", p.ThinFilm().Weight)
			}
		})
	}
}

// independentLinear gamma-decodes an sRGB Rgba the same way mesh.frag's toLinear() does
// (pow(c, 2.2)), written fresh here rather than calling srgbToLinear so this test
// checks the production code against an independent reference, not itself.
func independentLinear(c Rgba) Color3 {
	dec := func(v float32) float32 { return float32(math.Pow(float64(v), 2.2)) }
	return Color3{R: dec(c.R), G: dec(c.G), B: dec(c.B)}
}

func assertColorClose(t *testing.T, label string, got, want Color3) {
	t.Helper()
	const tol = 1e-5
	if absF32(got.R-want.R) > tol || absF32(got.G-want.G) > tol || absF32(got.B-want.B) > tol {
		t.Errorf("%s = %+v, want %+v (within %v)", label, got, want, tol)
	}
}

func absF32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// TestMigrateAppearanceOffLobesAreZeroValue proves the migration baseline itself is
// correct in isolation (not just via the catalog), and that a fully-off emissive maps
// to (0, black) rather than a NaN from a zero-luminance division.
func TestMigrateAppearanceOffLobesAreZeroValue(t *testing.T) {
	a := NewAppearance("test", SourceBuiltin, AppearanceSpec{
		DisplayName: "Test", Albedo: mustColor("#804020ff"), Metallic: 0.3, Roughness: 0.6,
		Emissive: mustColor("#000000ff"), Opacity: 1,
	})
	spec := migrateAppearance(a)

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
}
