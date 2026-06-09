// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/material"
	"oblikovati.org/renderer"
)

// Materials returns the session's appearance/material library (built-ins on first use;
// the project library and document-embedded assets fold in as those land). It is shared
// by every open document of the session, the way the project asset library is shared
// across a project's documents (ADR-0022).
func (s *Session) Materials() *material.Library {
	if s.materials == nil {
		s.materials = material.NewLibrary()
	}
	return s.materials
}

// LoadProjectMaterials attaches a project asset store and folds its shared appearances and
// materials into the library. The head calls this with an OS-backed store rooted at the
// active project's design-data directory; later customizations persist back through it.
func (s *Session) LoadProjectMaterials(store *material.Store) error {
	s.materialStore = store
	return store.Load(s.Materials())
}

// DuplicateAppearance creates a project-scoped editable copy of an appearance and persists
// the project library.
func (s *Session) DuplicateAppearance(baseID, name string) (*material.Appearance, error) {
	a, err := s.Materials().DuplicateAppearance(baseID, name, material.SourceProject)
	if err != nil {
		return nil, err
	}
	s.saveProjectMaterials()
	return a, nil
}

// DuplicateMaterial creates a project-scoped editable copy of a material.
func (s *Session) DuplicateMaterial(baseID, name string) (*material.Material, error) {
	m, err := s.Materials().DuplicateMaterial(baseID, name, material.SourceProject)
	if err != nil {
		return nil, err
	}
	s.saveProjectMaterials()
	return m, nil
}

// UpdateAppearance / UpdateMaterial edit an asset's spec (a no-op for built-ins) and
// persist the project library.
func (s *Session) UpdateAppearance(id string, spec material.AppearanceSpec) {
	s.Materials().EditAppearance(id, spec)
	s.saveProjectMaterials()
}

func (s *Session) UpdateMaterial(id string, spec material.MaterialSpec) {
	s.Materials().EditMaterial(id, spec)
	s.saveProjectMaterials()
}

// saveProjectMaterials writes the project library when a store is attached (else a no-op,
// e.g. in headless tests).
func (s *Session) saveProjectMaterials() {
	if s.materialStore != nil {
		_ = s.materialStore.Save(s.Materials())
	}
}

// SurfaceLookup returns the per-body PBR resolver for the active part: it maps each body
// to its effective appearance (via the part's assignment store and the material library)
// and converts that to a [renderer.Surface]. It returns nil when there is no active part,
// so the renderer falls back to the neutral default.
func (s *Session) SurfaceLookup() renderer.SurfaceLookup {
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	look := material.MergedLookup{Embedded: part.Assets(), Catalog: s.Materials()}
	assign := part.Assignments()
	return func(b *topo.Body) renderer.Surface {
		appr := assign.EffectiveAppearance(look, material.RefKey(b.ReferenceKey()), "")
		return appearanceSurface(appr)
	}
}

// appearanceSurface converts a model appearance into the renderer's PBR surface value.
func appearanceSurface(a *material.Appearance) renderer.Surface {
	em := a.Emissive()
	return renderer.Surface{
		Albedo:    a.Albedo().Array(),
		Metallic:  a.Metallic(),
		Roughness: a.Roughness(),
		Emissive:  [3]float32{em.R, em.G, em.B},
		Opacity:   a.Opacity(),
	}
}
