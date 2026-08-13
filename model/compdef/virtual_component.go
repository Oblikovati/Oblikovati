// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"oblikovati.org/math"
	"oblikovati.org/model/bom"
	"oblikovati.org/model/occurrence"
)

// Virtual components (#1979): a named occurrence with no backing document and no B-rep that still
// takes part in the bill of materials — paint, grease, labor, fasteners sold by weight. It satisfies
// occurrence.Definition with an empty range box (so it contributes no bounds and no mass) and
// bom.Component with its own part number/description/structure, so the BOM counts it as a line.

// VirtualComponentDefinition is a geometry-free, document-free component definition (#1979).
type VirtualComponentDefinition struct {
	displayName string
	partNumber  string
	description string
	structure   bom.Structure
	props       map[string]string
}

var (
	_ occurrence.Definition = (*VirtualComponentDefinition)(nil)
	_ bom.Component         = (*VirtualComponentDefinition)(nil)
)

// NewVirtualComponent builds a virtual component definition with a display name, part number and BOM
// structure (#1979).
func NewVirtualComponent(displayName, partNumber string, structure bom.Structure) *VirtualComponentDefinition {
	return &VirtualComponentDefinition{displayName: displayName, partNumber: partNumber, structure: structure}
}

// RangeBox returns the empty box: a virtual component has no geometry, so it contributes no bounds
// and no mass (occurrence.Definition).
func (v *VirtualComponentDefinition) RangeBox() math.Box { return math.EmptyBox() }

// DisplayName is the component's tree label.
func (v *VirtualComponentDefinition) DisplayName() string { return v.displayName }

// PartNumber / Description / BOMStructure / CustomProperties implement bom.Component (#1979).
func (v *VirtualComponentDefinition) PartNumber() string          { return v.partNumber }
func (v *VirtualComponentDefinition) Description() string         { return v.description }
func (v *VirtualComponentDefinition) BOMStructure() bom.Structure { return v.structure }
func (v *VirtualComponentDefinition) CustomProperties() map[string]string {
	return v.props
}

// AddVirtual places a new virtual component in the assembly: a named tree node with no geometry that
// appears in the BOM with its part number and structure (#1979).
func (a *AssemblyComponentDefinition) AddVirtual(name, partNumber string, structure bom.Structure, transform math.Matrix4) *occurrence.Occurrence {
	return a.Place(name, NewVirtualComponent(name, partNumber, structure), transform)
}
