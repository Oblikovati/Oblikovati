// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
)

// SheetMetalFaceData is the serialized form of a SheetMetalFaceFeature: the profile (sketch
// index + region index), the material side, and the boolean operation. Thickness is NOT
// stored — it is read live from the part's Thickness parameter on recompute (the sheet-metal
// invariant), so it can never round-trip a stale gauge.
type SheetMetalFaceData struct {
	Sketch    int   `yaml:"sketch"`
	Profile   int   `yaml:"profile"`
	Direction int32 `yaml:"direction,omitempty"`
	Operation int32 `yaml:"operation,omitempty"`
}

// serializeSheetMetalFace projects a Face recipe to its persisted form, erroring if its
// sketch is not in the part (so the profile cannot be lost silently).
func serializeSheetMetalFace(def *SheetMetalFaceDefinition, sk SketchIndexer) (*SheetMetalFaceData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal face references a sketch that is not in the part")
	}
	return &SheetMetalFaceData{
		Sketch: idx, Profile: def.ProfileIndex,
		Direction: int32(def.Direction), Operation: int32(def.Operation),
	}, nil
}

// restoreSheetMetalFace rebuilds a SheetMetalFaceFeature, erroring on a missing payload or
// unknown sketch.
func restoreSheetMetalFace(fs *PartFeatures, d *SheetMetalFaceData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal face feature is missing its payload")
	}
	s, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal face references sketch %d which is not in the part", d.Sketch)
	}
	def := &SheetMetalFaceDefinition{
		Sketch: s, ProfileIndex: d.Profile,
		Direction: ExtentDirection(d.Direction), Operation: ops.PartFeatureOperation(d.Operation),
	}
	return NewSheetMetalFaceFeatures(fs).Add(def), nil
}
