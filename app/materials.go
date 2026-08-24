// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/compdef"
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
	if asm, err := activeAssembly(s); err == nil {
		return s.assemblySurfaceLookup(asm)
	}
	if part, err := activePart(s); err == nil {
		return s.partSurfaceLookup(part)
	}
	return nil
}

// partSurfaceLookup resolves a part's bodies to their assigned appearance (face →
// body → body-material → part-material → default), with a color-style assignment
// winning over the appearance (M16-F02 #403/#408).
func (s *Session) partSurfaceLookup(part *compdef.PartComponentDefinition) renderer.SurfaceLookup {
	look := material.MergedLookup{Embedded: part.Assets(), Catalog: s.Materials()}
	assign := part.Assignments()
	return func(b *topo.Body) renderer.Surface {
		if name, ok := s.BodyColorStyle(string(b.ReferenceKey())); ok {
			if cs, found := s.styles.ByName(name); found {
				return styleSurface(cs)
			}
		}
		key := material.RefKey(b.ReferenceKey())
		// An assigned OpenPBRAppearance (M45-F05, ADR-0053) wins over the legacy chain — a
		// user who explicitly assigned one via the OpenPBR editor/API means it to be the
		// body's real material, not an inert side record (#2150). Falls through to the
		// legacy Appearance chain when nothing is assigned in the OpenPBR chain.
		if opbr, ok := assign.EffectiveOpenPBRAppearance(look, key, ""); ok {
			return openPBRAppearanceSurface(opbr)
		}
		appr := assign.EffectiveAppearance(look, key, "")
		return appearanceSurface(appr)
	}
}

// assemblySurfaceLookup resolves every placed occurrence body to the appearance from
// its OWN source part (that part's assignment store + merged material lookup), so an
// assembly view renders each component with its assigned material instead of the
// neutral default (Oblikovati#1103). The part graph already carries the linkage —
// each PlacedBody knows its source occurrence, whose definition is the part.
func (s *Session) assemblySurfaceLookup(asm *compdef.AssemblyComponentDefinition) renderer.SurfaceLookup {
	byBody := make(map[*topo.Body]renderer.SurfaceLookup)
	perPart := make(map[*compdef.PartComponentDefinition]renderer.SurfaceLookup)
	for _, pb := range asm.PlacedBodies() {
		part, ok := pb.Source.Definition().(*compdef.PartComponentDefinition)
		if !ok {
			continue // a sub-assembly occurrence's own placed bodies already carry their part
		}
		look, ok := perPart[part]
		if !ok {
			look = s.partSurfaceLookup(part)
			perPart[part] = look
		}
		byBody[pb.Body] = look
	}
	fallback := appearanceSurface(material.MergedLookup{Catalog: s.Materials()}.DefaultAppearance())
	return func(b *topo.Body) renderer.Surface {
		if look, ok := byBody[b]; ok {
			return look(b)
		}
		return fallback
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

// openPBRAppearanceSurface converts an OpenPBR appearance's Base/Specular/Emission/
// Geometry groups into the renderer's PBR surface value. renderer.Surface.Albedo/
// Emissive are sRGB-encoded (mesh.frag's toLinear() decodes them at shade time — see
// appearanceSurface above), but an OpenPBRAppearance's colors are already LINEAR
// (types.Color3, ACEScg working space, PBI-335/349) — so this, unlike appearanceSurface,
// encodes rather than passes the values straight through, to land on the same
// sRGB-encoded convention every consumer (raster and Realistic mode alike) expects.
func openPBRAppearanceSurface(a *material.OpenPBRAppearance) renderer.Surface {
	base, spec, geo := a.Base(), a.Specular(), a.Geometry()
	em := a.Emission()
	emissive := material.Color3{R: em.Color.R * em.Luminance, G: em.Color.G * em.Luminance, B: em.Color.B * em.Luminance}
	return renderer.Surface{
		Albedo:    encodeSRGBColor(base.Color),
		Metallic:  base.Metalness,
		Roughness: spec.Roughness,
		Emissive:  encodeSRGB3(emissive),
		Opacity:   geo.Opacity,
	}
}

// encodeSRGBChannel is the inverse of mesh.frag's toLinear(c) = pow(c, 2.2).
func encodeSRGBChannel(c float32) float32 {
	if c < 0 {
		return 0
	}
	return float32(math.Pow(float64(c), 1/2.2))
}

func encodeSRGB3(c material.Color3) [3]float32 {
	return [3]float32{encodeSRGBChannel(c.R), encodeSRGBChannel(c.G), encodeSRGBChannel(c.B)}
}

func encodeSRGBColor(c material.Color3) [4]float32 {
	return [4]float32{encodeSRGBChannel(c.R), encodeSRGBChannel(c.G), encodeSRGBChannel(c.B), 1}
}
