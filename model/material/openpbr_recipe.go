// SPDX-License-Identifier: GPL-2.0-only

package material

import "sort"

// OpenPBRAppearanceRecipe is the persisted YAML shape of an OpenPBR appearance (project
// library storage, ADR-0020/0022) — every group inline, mirroring [AppearanceRecipe]'s
// role for the metallic-roughness subset. Colors stay raw ACEScg floats (not hex): unlike
// [AppearanceRecipe]'s Albedo/Emissive, OpenPBR's emission_color is unbounded above.
type OpenPBRAppearanceRecipe struct {
	ID           string              `yaml:"id"`
	DisplayName  string              `yaml:"name"`
	Base         OpenPBRBase         `yaml:"base"`
	Specular     OpenPBRSpecular     `yaml:"specular"`
	Transmission OpenPBRTransmission `yaml:"transmission"`
	Subsurface   OpenPBRSubsurface   `yaml:"subsurface"`
	Coat         OpenPBRCoat         `yaml:"coat"`
	Fuzz         OpenPBRFuzz         `yaml:"fuzz"`
	ThinFilm     OpenPBRThinFilm     `yaml:"thinFilm"`
	Emission     OpenPBREmission     `yaml:"emission"`
	Geometry     OpenPBRGeometry     `yaml:"geometry"`
}

func openPBRAppearanceToRecipe(a *OpenPBRAppearance) OpenPBRAppearanceRecipe {
	s := a.spec
	return OpenPBRAppearanceRecipe{
		ID: a.id, DisplayName: s.DisplayName,
		Base: s.Base, Specular: s.Specular, Transmission: s.Transmission,
		Subsurface: s.Subsurface, Coat: s.Coat, Fuzz: s.Fuzz, ThinFilm: s.ThinFilm,
		Emission: s.Emission, Geometry: s.Geometry,
	}
}

func recipeToOpenPBRAppearance(r OpenPBRAppearanceRecipe, source Source) *OpenPBRAppearance {
	return NewOpenPBRAppearance(r.ID, source, OpenPBRAppearanceSpec{
		DisplayName: r.DisplayName,
		Base:        r.Base, Specular: r.Specular, Transmission: r.Transmission,
		Subsurface: r.Subsurface, Coat: r.Coat, Fuzz: r.Fuzz, ThinFilm: r.ThinFilm,
		Emission: r.Emission, Geometry: r.Geometry,
	})
}

func sortOpenPBRAppearances(in []*OpenPBRAppearance) []*OpenPBRAppearance {
	sort.Slice(in, func(i, j int) bool { return in[i].id < in[j].id })
	return in
}
