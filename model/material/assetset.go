// SPDX-License-Identifier: GPL-2.0-only

package material

// AssetSet is the appearances and materials a single document embeds — a copy of every
// non-built-in asset it uses, so the .obk is self-contained and portable even without the
// project library it was authored against (ADR-0022). Built-in assets are not embedded
// (they are always reproducible from the catalog), so the set stays small.
type AssetSet struct {
	appearances map[string]*Appearance
	materials   map[string]*Material
}

// NewAssetSet returns an empty document asset set.
func NewAssetSet() *AssetSet {
	return &AssetSet{
		appearances: map[string]*Appearance{},
		materials:   map[string]*Material{},
	}
}

// PutAppearance / PutMaterial embed (or replace) an asset under its id.
func (s *AssetSet) PutAppearance(a *Appearance) { s.appearances[a.id] = a }
func (s *AssetSet) PutMaterial(m *Material)     { s.materials[m.id] = m }

// Appearance / Material look up an embedded asset by id.
func (s *AssetSet) Appearance(id string) (*Appearance, bool) {
	a, ok := s.appearances[id]
	return a, ok
}
func (s *AssetSet) Material(id string) (*Material, bool) { m, ok := s.materials[id]; return m, ok }

// Appearances / Materials return the embedded assets (order is unspecified; callers
// that need determinism sort by id).
func (s *AssetSet) Appearances() []*Appearance {
	out := make([]*Appearance, 0, len(s.appearances))
	for _, a := range s.appearances {
		out = append(out, a)
	}
	return out
}

func (s *AssetSet) Materials() []*Material {
	out := make([]*Material, 0, len(s.materials))
	for _, m := range s.materials {
		out = append(out, m)
	}
	return out
}

// MergedLookup resolves asset ids over a document's embedded assets first, then the
// session catalog (built-ins + project). It satisfies [AssetLookup], so the renderer and
// browser see the document's own copies before the shared catalog — the document-embedded
// asset is authoritative for portability.
type MergedLookup struct {
	Embedded *AssetSet
	Catalog  *Library
}

// Appearance / Material look up an id, embedded set first.
func (m MergedLookup) Appearance(id string) (*Appearance, bool) {
	if m.Embedded != nil {
		if a, ok := m.Embedded.Appearance(id); ok {
			return a, true
		}
	}
	return m.Catalog.Appearance(id)
}

func (m MergedLookup) Material(id string) (*Material, bool) {
	if m.Embedded != nil {
		if mat, ok := m.Embedded.Material(id); ok {
			return mat, true
		}
	}
	return m.Catalog.Material(id)
}

// DefaultAppearance defers to the catalog's neutral default.
func (m MergedLookup) DefaultAppearance() *Appearance { return m.Catalog.DefaultAppearance() }
