// SPDX-License-Identifier: GPL-2.0-only

package material

import "oblikovati/api/contract"

// AppearanceSpec is the editable content of an appearance — everything but its identity
// and source. Carrying the editable fields as one value keeps construction and edits
// (which must respect the read-only rule for built-ins) in one place.
type AppearanceSpec struct {
	DisplayName string
	Albedo      Rgba
	Metallic    float32
	Roughness   float32
	Emissive    Rgba
	Opacity     float32
}

// Appearance is one PBR appearance asset. It satisfies [contract.Appearance].
type Appearance struct {
	id     string
	source Source
	spec   AppearanceSpec
}

var _ contract.Appearance = (*Appearance)(nil)

// NewAppearance builds an appearance with a stable id, a source, and its spec.
func NewAppearance(id string, source Source, spec AppearanceSpec) *Appearance {
	return &Appearance{id: id, source: source, spec: spec}
}

// ID / DisplayName / Source / Albedo / Metallic / Roughness / Emissive / Opacity satisfy
// the read-only public contract.
func (a *Appearance) ID() string          { return a.id }
func (a *Appearance) DisplayName() string { return a.spec.DisplayName }
func (a *Appearance) Source() Source      { return a.source }
func (a *Appearance) Albedo() Rgba        { return a.spec.Albedo }
func (a *Appearance) Metallic() float32   { return a.spec.Metallic }
func (a *Appearance) Roughness() float32  { return a.spec.Roughness }
func (a *Appearance) Emissive() Rgba      { return a.spec.Emissive }
func (a *Appearance) Opacity() float32    { return a.spec.Opacity }

// Spec returns a copy of the editable fields (the editor reads it to populate controls).
func (a *Appearance) Spec() AppearanceSpec { return a.spec }

// SetSpec replaces the editable fields. It is a no-op on a built-in (read-only); only
// project/document assets are editable (see [types.AssetSource.Editable]).
func (a *Appearance) SetSpec(spec AppearanceSpec) {
	if !a.source.Editable() {
		return
	}
	a.spec = spec
}

// duplicate returns an independent copy under a new id/name and source — a full snapshot
// of the spec, so editing the copy never touches the original.
func (a *Appearance) duplicate(id, name string, source Source) *Appearance {
	spec := a.spec
	spec.DisplayName = name
	return &Appearance{id: id, source: source, spec: spec}
}
