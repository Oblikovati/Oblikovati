// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/model/material"
	"github.com/Oblikovati/oblikovati/renderer"
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

// SurfaceLookup returns the per-body PBR resolver for the active part: it maps each body
// to its effective appearance (via the part's assignment store and the material library)
// and converts that to a [renderer.Surface]. It returns nil when there is no active part,
// so the renderer falls back to the neutral default.
func (s *Session) SurfaceLookup() renderer.SurfaceLookup {
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	lib := s.Materials()
	assign := part.Assignments()
	return func(b *topo.Body) renderer.Surface {
		appr := assign.EffectiveAppearance(lib, material.RefKey(b.ReferenceKey()), "")
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
