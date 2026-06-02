// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"fmt"

	"github.com/Oblikovati/api/types"
)

// DefaultAppearanceID is the neutral appearance applied when a body has no material or
// appearance assigned — it reproduces the renderer's pre-materials default gray, so an
// un-assigned model looks exactly as it did before this subsystem.
const DefaultAppearanceID = "default"

// seedBuiltins fills a fresh library with the shipped, read-only catalog: a neutral
// default plus a few representative metals, plastics, and wood, each with a paired
// appearance. Property values follow common engineering references (Inventor-comparable).
func seedBuiltins(l *Library) {
	for _, a := range builtinAppearances() {
		l.AddAppearance(a)
	}
	for _, m := range builtinMaterials() {
		l.AddMaterial(m)
	}
}

// builtinAppearances returns the shipped appearance catalog.
func builtinAppearances() []*Appearance {
	a := func(id, name, albedo string, metallic, roughness float32) *Appearance {
		return NewAppearance(id, SourceBuiltin, AppearanceSpec{
			DisplayName: name, Albedo: mustColor(albedo), Metallic: metallic, Roughness: roughness,
			Emissive: mustColor("#000000ff"), Opacity: 1,
		})
	}
	return []*Appearance{
		a(DefaultAppearanceID, "Default", "#b3b8bdff", 0, 0.6),
		a("steel", "Steel", "#8a8d92ff", 0.9, 0.40),
		a("aluminum", "Aluminum", "#c8ccd0ff", 0.9, 0.35),
		a("abs-black", "ABS (Black)", "#1a1a1aff", 0, 0.50),
		a("abs-white", "ABS (White)", "#e8e8e8ff", 0, 0.50),
		a("oak", "Oak", "#b88a4fff", 0, 0.70),
	}
}

// builtinMaterials returns the shipped material catalog, each referencing a built-in
// appearance by id.
func builtinMaterials() []*Material {
	return []*Material{
		NewMaterial("steel", SourceBuiltin, MaterialSpec{
			DisplayName: "Steel", Density: 7.85, AppearanceID: "steel",
			Mechanical: Mechanical{YoungsModulus: 210, PoissonsRatio: 0.30, YieldStrength: 350, UltimateTensileStrength: 420},
			Thermal:    Thermal{Conductivity: 50, ExpansionCoeff: 12e-6, SpecificHeat: 480},
			Electrical: Electrical{Resistivity: 1.7e-7, RelativePermittivity: 1},
		}),
		NewMaterial("aluminum-6061", SourceBuiltin, MaterialSpec{
			DisplayName: "Aluminum 6061", Density: 2.70, AppearanceID: "aluminum",
			Mechanical: Mechanical{YoungsModulus: 68.9, PoissonsRatio: 0.33, YieldStrength: 276, UltimateTensileStrength: 310},
			Thermal:    Thermal{Conductivity: 167, ExpansionCoeff: 23.6e-6, SpecificHeat: 896},
			Electrical: Electrical{Resistivity: 3.99e-8, RelativePermittivity: 1},
		}),
		NewMaterial("abs-plastic", SourceBuiltin, MaterialSpec{
			DisplayName: "ABS Plastic", Density: 1.06, AppearanceID: "abs-black",
			Mechanical: Mechanical{YoungsModulus: 2.1, PoissonsRatio: 0.35, YieldStrength: 40, UltimateTensileStrength: 44},
			Thermal:    Thermal{Conductivity: 0.17, ExpansionCoeff: 90e-6, SpecificHeat: 1300},
			Electrical: Electrical{Resistivity: 1e15, RelativePermittivity: 3.0},
		}),
		NewMaterial("oak-wood", SourceBuiltin, MaterialSpec{
			DisplayName: "Oak Wood", Density: 0.75, AppearanceID: "oak",
			Mechanical: Mechanical{YoungsModulus: 11, PoissonsRatio: 0.35, YieldStrength: 40, UltimateTensileStrength: 50},
			Thermal:    Thermal{Conductivity: 0.17, ExpansionCoeff: 5e-6, SpecificHeat: 1700},
			Electrical: Electrical{Resistivity: 1e14, RelativePermittivity: 2.0},
		}),
	}
}

// hex parses an authored built-in color literal, panicking with the offending value on a
// malformed constant (a programming error caught by the package tests, not a runtime
// condition).
func mustColor(s string) Rgba {
	c, err := types.ParseHex(s)
	if err != nil {
		panic(fmt.Sprintf("material: malformed built-in color %q: %v", s, err))
	}
	return c
}
