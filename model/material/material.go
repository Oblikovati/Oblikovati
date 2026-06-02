// SPDX-License-Identifier: GPL-2.0-only

package material

import "github.com/Oblikovati/api/contract"

// MaterialSpec is the editable content of a material — density, the property groups, and
// the id of the appearance it renders with.
type MaterialSpec struct {
	DisplayName  string
	Density      float64 // g/cm³
	Mechanical   Mechanical
	Thermal      Thermal
	Electrical   Electrical
	AppearanceID string
}

// Material is one physical-world material asset. It satisfies [contract.Material].
type Material struct {
	id     string
	source Source
	spec   MaterialSpec
}

var _ contract.Material = (*Material)(nil)

// NewMaterial builds a material with a stable id, a source, and its spec.
func NewMaterial(id string, source Source, spec MaterialSpec) *Material {
	return &Material{id: id, source: source, spec: spec}
}

// ID / DisplayName / Source / Density / Mechanical / Thermal / Electrical / AppearanceID
// satisfy the read-only public contract.
func (m *Material) ID() string             { return m.id }
func (m *Material) DisplayName() string    { return m.spec.DisplayName }
func (m *Material) Source() Source         { return m.source }
func (m *Material) Density() float64       { return m.spec.Density }
func (m *Material) Mechanical() Mechanical { return m.spec.Mechanical }
func (m *Material) Thermal() Thermal       { return m.spec.Thermal }
func (m *Material) Electrical() Electrical { return m.spec.Electrical }
func (m *Material) AppearanceID() string   { return m.spec.AppearanceID }

// Spec returns a copy of the editable fields.
func (m *Material) Spec() MaterialSpec { return m.spec }

// SetSpec replaces the editable fields; a no-op on a built-in (read-only).
func (m *Material) SetSpec(spec MaterialSpec) {
	if !m.source.Editable() {
		return
	}
	m.spec = spec
}

// duplicate returns an independent copy under a new id/name and source.
func (m *Material) duplicate(id, name string, source Source) *Material {
	spec := m.spec
	spec.DisplayName = name
	return &Material{id: id, source: source, spec: spec}
}
