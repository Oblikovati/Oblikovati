// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"strconv"

	"oblikovati.org/model/attr"
	"oblikovati.org/model/bom"
)

// Document BOM metadata (#718): the part and assembly definitions satisfy [bom.Component] by
// reading their iProperties (#156), so a BOM over a real assembly shows each component's part
// number, description, and custom property columns — instead of the blank default a definition
// without metadata falls back to. The BOM structure rides a Design Tracking property, so it
// persists through the same recipe channel as the rest of the iProperties.
//
// A per-OCCURRENCE structure override (the reference API allows one) is a follow-up: occurrence
// cannot return a bom.Structure without an import cycle (bom imports occurrence).

var (
	_ bom.Component = (*PartComponentDefinition)(nil)
	_ bom.Component = (*AssemblyComponentDefinition)(nil)
)

// The Design Tracking property names the BOM reads.
const (
	propPartNumber   = "Part Number"
	propDescription  = "Description"
	propBOMStructure = "BOM Structure"
)

// PartNumber / Description / BOMStructure / CustomProperties implement [bom.Component] for a part
// from its iProperties.
func (d *PartComponentDefinition) PartNumber() string {
	return designTrackingProperty(d.props, propPartNumber)
}
func (d *PartComponentDefinition) Description() string {
	return designTrackingProperty(d.props, propDescription)
}
func (d *PartComponentDefinition) BOMStructure() bom.Structure { return bomStructureOf(d.props) }
func (d *PartComponentDefinition) CustomProperties() map[string]string {
	return customPropertiesOf(d.props)
}

// PartNumber / Description / BOMStructure / CustomProperties implement [bom.Component] for an
// assembly (a sub-assembly carries metadata too) from its iProperties.
func (a *AssemblyComponentDefinition) PartNumber() string {
	return designTrackingProperty(a.props, propPartNumber)
}
func (a *AssemblyComponentDefinition) Description() string {
	return designTrackingProperty(a.props, propDescription)
}
func (a *AssemblyComponentDefinition) BOMStructure() bom.Structure { return bomStructureOf(a.props) }
func (a *AssemblyComponentDefinition) CustomProperties() map[string]string {
	return customPropertiesOf(a.props)
}

// designTrackingProperty reads a string-valued property from the Design Tracking set, returning ""
// when the set or property is absent or not a string.
func designTrackingProperty(ps *attr.PropertySets, name string) string {
	s, ok := ps.Lookup(attr.DesignTracking)
	if !ok {
		return ""
	}
	p, ok := s.Property(name)
	if !ok {
		return ""
	}
	v, _ := p.Value().Str()
	return v
}

// bomStructureOf maps the "BOM Structure" Design Tracking property to a [bom.Structure], defaulting
// to Normal when it is unset or unrecognized.
func bomStructureOf(ps *attr.PropertySets) bom.Structure {
	if st, ok := bom.ParseStructure(designTrackingProperty(ps, propBOMStructure)); ok {
		return st
	}
	return bom.Normal
}

// customPropertiesOf flattens every property across the document's sets into a name→display-string
// map, the source for a BOM's custom export columns. Returns nil when there are no properties.
func customPropertiesOf(ps *attr.PropertySets) map[string]string {
	out := map[string]string{}
	for _, set := range ps.Sets() {
		for _, p := range set.Properties() {
			out[p.Name()] = propertyDisplayString(p.Value())
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// propertyDisplayString renders a typed value for a BOM column (a string property is verbatim;
// numbers/booleans format; bytes are not column data and render empty).
func propertyDisplayString(v attr.Value) string {
	switch v.Type() {
	case attr.Integer:
		i, _ := v.Int()
		return strconv.FormatInt(i, 10)
	case attr.Double:
		f, _ := v.Float()
		return strconv.FormatFloat(f, 'g', -1, 64)
	case attr.Boolean:
		b, _ := v.Bool()
		return strconv.FormatBool(b)
	case attr.Bytes:
		return ""
	default:
		s, _ := v.Str()
		return s
	}
}
