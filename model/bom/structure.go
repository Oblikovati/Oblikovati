// SPDX-License-Identifier: GPL-2.0-only

// Package bom derives a bill of materials from an assembly's occurrence structure: a
// structured (hierarchical) and a parts-only (flat, totalled) view of the components,
// honoring each component's BOM structure (phantoms collapse, purchased/inseparable
// sub-assemblies count as one), with quantities and stable item numbers that feed
// parts lists and balloons (M14). It is the reference API's BOM / BOMView / BOMRow
// (M11-F05, #349).
package bom

// Structure classifies how a component participates in the bill of materials — the
// reference API's BOMStructureEnum. The default is Normal.
type Structure int

const (
	// Normal is a counted row whose sub-assembly children are expanded.
	Normal Structure = iota
	// Phantom is not a row of its own: its children are promoted into its parent, so
	// a purely organizational sub-assembly collapses out of the BOM.
	Phantom
	// Reference is shown for context but never counted (construction/skeleton geometry).
	Reference
	// Purchased is a bought item counted as a single line; its children are not broken out.
	Purchased
	// Inseparable is a welded/glued sub-assembly counted as a single line; not broken out.
	Inseparable
	// Default inherits the definition's structure — a per-occurrence override that defers to the
	// shared definition (#1978). It never appears on a resolved row (it resolves to the definition).
	Default
	// Varies marks a structured row whose grouped occurrences carry differing structures (iAssembly)
	// — a computed value, never one you set on a single component (#1978).
	Varies
)

// String returns the lowercase name of the structure, used in diagnostics and export.
func (s Structure) String() string {
	switch s {
	case Normal:
		return "normal"
	case Phantom:
		return "phantom"
	case Reference:
		return "reference"
	case Purchased:
		return "purchased"
	case Inseparable:
		return "inseparable"
	case Default:
		return "default"
	case Varies:
		return "varies"
	default:
		return "unknown"
	}
}

// ParseStructure resolves a structure's lowercase name back to its value, reporting false on an
// unknown name (so a stored "BOM Structure" property maps to the enum; #718).
func ParseStructure(s string) (Structure, bool) {
	for _, v := range []Structure{Normal, Phantom, Reference, Purchased, Inseparable, Default, Varies} {
		if v.String() == s {
			return v, true
		}
	}
	return Normal, false
}

// expands reports whether a sub-assembly of this structure is traversed into its
// children (Normal/Phantom) rather than counted as one opaque line
// (Purchased/Inseparable). Reference is neither — it is dropped from counts entirely.
func (s Structure) expands() bool {
	return s == Normal || s == Phantom
}

// Component is the metadata a BOM needs from a placed component definition: a part
// number, a description, its BOM structure, and custom properties for export columns.
// A definition that does not implement it is treated as a [Normal] component with no
// part metadata, so the BOM still reflects its structure and quantity.
type Component interface {
	// PartNumber is the component's identifying number (shared across its instances).
	PartNumber() string
	// Description is the component's human-readable description.
	Description() string
	// BOMStructure is how the component participates in the BOM.
	BOMStructure() Structure
	// CustomProperties are the component's properties (iProperties), keyed by name,
	// available as export columns.
	CustomProperties() map[string]string
}

// defaultComponent is the metadata used for a placed definition that does not
// implement [Component]: a normal, unnamed line.
type defaultComponent struct{}

func (defaultComponent) PartNumber() string                  { return "" }
func (defaultComponent) Description() string                 { return "" }
func (defaultComponent) BOMStructure() Structure             { return Normal }
func (defaultComponent) CustomProperties() map[string]string { return nil }
