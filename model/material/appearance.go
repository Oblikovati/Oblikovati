// SPDX-License-Identifier: GPL-2.0-only

package material

import "oblikovati.org/api/contract"

// AppearanceSpec is the editable content of an appearance — every OpenPBR Surface
// v1.1.1 lobe group, but not its identity/source.
type AppearanceSpec struct {
	DisplayName  string
	Base         OpenPBRBase
	Specular     OpenPBRSpecular
	Transmission OpenPBRTransmission
	Subsurface   OpenPBRSubsurface
	Coat         OpenPBRCoat
	Fuzz         OpenPBRFuzz
	ThinFilm     OpenPBRThinFilm
	Emission     OpenPBREmission
	Geometry     OpenPBRGeometry
}

// Appearance is one full OpenPBR Surface v1.1.1 appearance asset. It satisfies
// [contract.Appearance].
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

// ID / DisplayName / Source satisfy the shared asset identity contract.
func (a *Appearance) ID() string          { return a.id }
func (a *Appearance) DisplayName() string { return a.spec.DisplayName }
func (a *Appearance) Source() Source      { return a.source }

// Base / Specular / Transmission / Subsurface / Coat / Fuzz / ThinFilm / Emission /
// Geometry satisfy the read-only public contract's grouped accessors.
func (a *Appearance) Base() OpenPBRBase                 { return a.spec.Base }
func (a *Appearance) Specular() OpenPBRSpecular         { return a.spec.Specular }
func (a *Appearance) Transmission() OpenPBRTransmission { return a.spec.Transmission }
func (a *Appearance) Subsurface() OpenPBRSubsurface     { return a.spec.Subsurface }
func (a *Appearance) Coat() OpenPBRCoat                 { return a.spec.Coat }
func (a *Appearance) Fuzz() OpenPBRFuzz                 { return a.spec.Fuzz }
func (a *Appearance) ThinFilm() OpenPBRThinFilm         { return a.spec.ThinFilm }
func (a *Appearance) Emission() OpenPBREmission         { return a.spec.Emission }
func (a *Appearance) Geometry() OpenPBRGeometry         { return a.spec.Geometry }

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
