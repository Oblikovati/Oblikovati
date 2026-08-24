// SPDX-License-Identifier: GPL-2.0-only

package material

import "oblikovati.org/api/contract"

// OpenPBRAppearanceSpec is the editable content of an OpenPBR appearance — every lobe
// group, but not its identity/source (mirrors [AppearanceSpec]'s split).
type OpenPBRAppearanceSpec struct {
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

// OpenPBRAppearance is one full OpenPBR Surface v1.1.1 appearance asset. It satisfies
// [contract.OpenPBRAppearance] — additive alongside [Appearance], not a replacement.
type OpenPBRAppearance struct {
	id     string
	source Source
	spec   OpenPBRAppearanceSpec
}

var _ contract.OpenPBRAppearance = (*OpenPBRAppearance)(nil)

// NewOpenPBRAppearance builds an OpenPBR appearance with a stable id, a source, and its
// spec.
func NewOpenPBRAppearance(id string, source Source, spec OpenPBRAppearanceSpec) *OpenPBRAppearance {
	return &OpenPBRAppearance{id: id, source: source, spec: spec}
}

// ID / DisplayName / Source satisfy the shared asset identity contract.
func (a *OpenPBRAppearance) ID() string          { return a.id }
func (a *OpenPBRAppearance) DisplayName() string { return a.spec.DisplayName }
func (a *OpenPBRAppearance) Source() Source      { return a.source }

// Base / Specular / Transmission / Subsurface / Coat / Fuzz / ThinFilm / Emission /
// Geometry satisfy the read-only public contract's grouped accessors.
func (a *OpenPBRAppearance) Base() OpenPBRBase                 { return a.spec.Base }
func (a *OpenPBRAppearance) Specular() OpenPBRSpecular         { return a.spec.Specular }
func (a *OpenPBRAppearance) Transmission() OpenPBRTransmission { return a.spec.Transmission }
func (a *OpenPBRAppearance) Subsurface() OpenPBRSubsurface     { return a.spec.Subsurface }
func (a *OpenPBRAppearance) Coat() OpenPBRCoat                 { return a.spec.Coat }
func (a *OpenPBRAppearance) Fuzz() OpenPBRFuzz                 { return a.spec.Fuzz }
func (a *OpenPBRAppearance) ThinFilm() OpenPBRThinFilm         { return a.spec.ThinFilm }
func (a *OpenPBRAppearance) Emission() OpenPBREmission         { return a.spec.Emission }
func (a *OpenPBRAppearance) Geometry() OpenPBRGeometry         { return a.spec.Geometry }

// Spec returns a copy of the editable fields (the editor reads it to populate controls).
func (a *OpenPBRAppearance) Spec() OpenPBRAppearanceSpec { return a.spec }

// SetSpec replaces the editable fields. It is a no-op on a built-in (read-only); only
// project/document assets are editable (see [types.AssetSource.Editable]).
func (a *OpenPBRAppearance) SetSpec(spec OpenPBRAppearanceSpec) {
	if !a.source.Editable() {
		return
	}
	a.spec = spec
}

// duplicate returns an independent copy under a new id/name and source — a full snapshot
// of the spec, so editing the copy never touches the original.
func (a *OpenPBRAppearance) duplicate(id, name string, source Source) *OpenPBRAppearance {
	spec := a.spec
	spec.DisplayName = name
	return &OpenPBRAppearance{id: id, source: source, spec: spec}
}
