// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
)

// SheetMetalLoftedFlangeData is the serialized form of a SheetMetalLoftedFlangeFeature: the two
// profile sketch indices, the boolean operation, and the output settings (#1966) — how the
// transition is calculated, the facet tolerance for the press-brake modes, whether the corners
// converge, and the end-bend radius. Thickness is read live from the rule.
type SheetMetalLoftedFlangeData struct {
	ProfileA       int     `yaml:"profileA"`
	ProfileB       int     `yaml:"profileB"`
	Operation      int32   `yaml:"operation,omitempty"`
	Output         int32   `yaml:"output,omitempty"`
	FacetTolerance float64 `yaml:"facetTolerance,omitempty"`
	Converge       bool    `yaml:"converge,omitempty"`
	Radius         float64 `yaml:"radius,omitempty"`
}

// serializeSheetMetalLoftedFlange projects a lofted-flange recipe to its persisted form,
// erroring if either profile sketch is not in the part.
func serializeSheetMetalLoftedFlange(def *SheetMetalLoftedFlangeDefinition, sk SketchIndexer) (*SheetMetalLoftedFlangeData, error) {
	a, ok := sk.IndexOf(def.ProfileA)
	if !ok {
		return nil, fmt.Errorf("sheet-metal lofted flange references a profile A sketch not in the part")
	}
	b, ok := sk.IndexOf(def.ProfileB)
	if !ok {
		return nil, fmt.Errorf("sheet-metal lofted flange references a profile B sketch not in the part")
	}
	return &SheetMetalLoftedFlangeData{
		ProfileA: a, ProfileB: b, Operation: int32(def.Operation), Output: int32(def.Output),
		FacetTolerance: def.FacetTolerance, Converge: def.Converge, Radius: evalFloat(def.Radius),
	}, nil
}

// restoreSheetMetalLoftedFlange rebuilds a lofted-flange feature, erroring on a missing
// payload or an unknown profile sketch.
func restoreSheetMetalLoftedFlange(fs *PartFeatures, d *SheetMetalLoftedFlangeData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal lofted flange feature is missing its payload")
	}
	a, ok := sk.At(d.ProfileA)
	if !ok {
		return nil, fmt.Errorf("sheet-metal lofted flange references sketch %d which is not in the part", d.ProfileA)
	}
	b, ok := sk.At(d.ProfileB)
	if !ok {
		return nil, fmt.Errorf("sheet-metal lofted flange references sketch %d which is not in the part", d.ProfileB)
	}
	def := &SheetMetalLoftedFlangeDefinition{
		ProfileA: a, ProfileB: b, Operation: ops.PartFeatureOperation(d.Operation),
		Output: LoftedFlangeOutputType(d.Output), FacetTolerance: d.FacetTolerance, Converge: d.Converge,
	}
	if d.Radius != 0 {
		def.Radius = constFloat(d.Radius)
	}
	return NewSheetMetalLoftedFlangeFeatures(fs).Add(def), nil
}
