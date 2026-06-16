// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
)

// SheetMetalLoftedFlangeData is the serialized form of a SheetMetalLoftedFlangeFeature: the
// two profile sketch indices and the boolean operation. Thickness is read live from the rule.
type SheetMetalLoftedFlangeData struct {
	ProfileA  int   `yaml:"profileA"`
	ProfileB  int   `yaml:"profileB"`
	Operation int32 `yaml:"operation,omitempty"`
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
	return &SheetMetalLoftedFlangeData{ProfileA: a, ProfileB: b, Operation: int32(def.Operation)}, nil
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
	return NewSheetMetalLoftedFlangeFeatures(fs).Add(&SheetMetalLoftedFlangeDefinition{
		ProfileA: a, ProfileB: b, Operation: ops.PartFeatureOperation(d.Operation),
	}), nil
}
