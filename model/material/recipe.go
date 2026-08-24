// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"sort"

	"oblikovati.org/api/types"
)

// AppearanceRecipe is the persisted YAML shape of a full OpenPBR Surface v1.1.1
// appearance (project library storage / a document's embedded assets, ADR-0020/0022) —
// every group inline. Colors stay raw ACEScg floats (not hex): emission_color is
// unbounded above, which hex cannot represent.
type AppearanceRecipe struct {
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

// MaterialRecipe / AssignmentRecipe are the other persisted YAML shapes of the materials
// data — embedded in a document's recipe and stored in the project library
// (ADR-0020/0022).
type MaterialRecipe struct {
	ID          string           `yaml:"id"`
	DisplayName string           `yaml:"name"`
	Density     float64          `yaml:"density"`
	Mechanical  types.Mechanical `yaml:"mechanical,omitempty"`
	Thermal     types.Thermal    `yaml:"thermal,omitempty"`
	Electrical  types.Electrical `yaml:"electrical,omitempty"`
	// Magnetic carries magnetostatics data; a pointer so the block is omitted entirely for
	// the overwhelmingly common non-magnetic material (no {class:"",...} noise on disk).
	Magnetic *types.Magnetic `yaml:"magnetic,omitempty"`
	// Isotropy / Orthotropic capture direction-dependent elasticity (ADR-0025). Orthotropic
	// is a pointer so the block is omitted entirely for ordinary isotropic materials.
	Isotropy     types.IsotropyClass       `yaml:"isotropy,omitempty"`
	Orthotropic  *types.AnisotropicElastic `yaml:"orthotropic,omitempty"`
	AppearanceID string                    `yaml:"appearance,omitempty"`
}

type AssignmentRecipe struct {
	PartMaterial   string            `yaml:"partMaterial,omitempty"`
	PartAppearance string            `yaml:"partAppearance,omitempty"`
	BodyMaterial   map[string]string `yaml:"bodyMaterial,omitempty"`
	BodyAppearance map[string]string `yaml:"bodyAppearance,omitempty"`
	FaceAppearance map[string]string `yaml:"faceAppearance,omitempty"`
}

// RecipeData is the materials section embedded in a part's recipe: the document's own
// assets plus its assignments.
type RecipeData struct {
	Appearances []AppearanceRecipe `yaml:"appearances,omitempty"`
	Materials   []MaterialRecipe   `yaml:"materials,omitempty"`
	Assignments *AssignmentRecipe  `yaml:"assignments,omitempty"`
}

// MarshalRecipe captures a document's embedded asset set and assignments as RecipeData,
// in a deterministic (id-sorted) order so the YAML diffs cleanly.
func MarshalRecipe(set *AssetSet, assign *AssignmentStore) RecipeData {
	var data RecipeData
	for _, a := range sortAppearances(set.Appearances()) {
		data.Appearances = append(data.Appearances, appearanceToRecipe(a))
	}
	for _, m := range sortMaterials(set.Materials()) {
		data.Materials = append(data.Materials, materialToRecipe(m))
	}
	data.Assignments = assignmentRecipe(assign)
	return data
}

// ApplyRecipe restores embedded assets (as document-source) into set and assignments into
// assign.
func ApplyRecipe(data RecipeData, set *AssetSet, assign *AssignmentStore) error {
	for _, ar := range data.Appearances {
		set.PutAppearance(recipeToAppearance(ar, SourceDocument))
	}
	for _, mr := range data.Materials {
		set.PutMaterial(recipeToMaterial(mr, SourceDocument))
	}
	if data.Assignments != nil {
		applyAssignmentRecipe(assign, data.Assignments)
	}
	return nil
}

func appearanceToRecipe(a *Appearance) AppearanceRecipe {
	s := a.spec
	return AppearanceRecipe{
		ID: a.id, DisplayName: s.DisplayName,
		Base: s.Base, Specular: s.Specular, Transmission: s.Transmission,
		Subsurface: s.Subsurface, Coat: s.Coat, Fuzz: s.Fuzz, ThinFilm: s.ThinFilm,
		Emission: s.Emission, Geometry: s.Geometry,
	}
}

func recipeToAppearance(r AppearanceRecipe, source Source) *Appearance {
	return NewAppearance(r.ID, source, AppearanceSpec{
		DisplayName: r.DisplayName,
		Base:        r.Base, Specular: r.Specular, Transmission: r.Transmission,
		Subsurface: r.Subsurface, Coat: r.Coat, Fuzz: r.Fuzz, ThinFilm: r.ThinFilm,
		Emission: r.Emission, Geometry: r.Geometry,
	})
}

func materialToRecipe(m *Material) MaterialRecipe {
	r := MaterialRecipe{
		ID: m.id, DisplayName: m.spec.DisplayName, Density: m.spec.Density,
		Mechanical: m.spec.Mechanical, Thermal: m.spec.Thermal, Electrical: m.spec.Electrical,
		Isotropy: m.spec.IsotropyClass, AppearanceID: m.spec.AppearanceID,
	}
	if m.spec.IsotropyClass.Anisotropic() {
		o := m.spec.Anisotropic
		r.Orthotropic = &o
	}
	if m.spec.Magnetic.IsMagnetic() {
		mag := m.spec.Magnetic
		r.Magnetic = &mag
	}
	return r
}

func recipeToMaterial(r MaterialRecipe, source Source) *Material {
	spec := MaterialSpec{
		DisplayName: r.DisplayName, Density: r.Density,
		Mechanical: r.Mechanical, Thermal: r.Thermal, Electrical: r.Electrical,
		IsotropyClass: r.Isotropy, AppearanceID: r.AppearanceID,
	}
	if r.Orthotropic != nil {
		spec.Anisotropic = *r.Orthotropic
	}
	if r.Magnetic != nil {
		spec.Magnetic = *r.Magnetic
	}
	return NewMaterial(r.ID, source, spec)
}

// assignmentRecipe captures the store, or nil when nothing is assigned (keeps the recipe
// minimal).
func assignmentRecipe(assign *AssignmentStore) *AssignmentRecipe {
	r := &AssignmentRecipe{
		PartMaterial:   assign.partMaterial,
		PartAppearance: assign.partAppearance,
		BodyMaterial:   nonEmpty(assign.bodyMaterial),
		BodyAppearance: nonEmpty(assign.bodyAppearance),
		FaceAppearance: nonEmpty(assign.faceAppearance),
	}
	if r.PartMaterial == "" && r.PartAppearance == "" &&
		r.BodyMaterial == nil && r.BodyAppearance == nil && r.FaceAppearance == nil {
		return nil
	}
	return r
}

// applyAssignmentRecipe writes a recipe's assignments back into a store.
func applyAssignmentRecipe(assign *AssignmentStore, r *AssignmentRecipe) {
	assign.partMaterial = r.PartMaterial
	assign.partAppearance = r.PartAppearance
	assign.bodyMaterial = orEmpty(r.BodyMaterial)
	assign.bodyAppearance = orEmpty(r.BodyAppearance)
	assign.faceAppearance = orEmpty(r.FaceAppearance)
}

func sortAppearances(in []*Appearance) []*Appearance {
	sort.Slice(in, func(i, j int) bool { return in[i].id < in[j].id })
	return in
}

func sortMaterials(in []*Material) []*Material {
	sort.Slice(in, func(i, j int) bool { return in[i].id < in[j].id })
	return in
}

// nonEmpty returns m, or nil when empty (so an empty map is omitted from YAML).
func nonEmpty(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	return copyMap(m)
}

// orEmpty returns m, or a fresh empty map when nil (so a store never holds a nil map).
func orEmpty(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}
