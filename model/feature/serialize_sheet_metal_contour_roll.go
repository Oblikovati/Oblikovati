// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
)

// SheetMetalContourRollData is the serialized form of a SheetMetalContourRollFeature: the
// profile sketch index, the axis line index in that sketch, the optional sweep angle (0 ⇒
// full 360°), and the boolean operation. Thickness is read live from the rule.
type SheetMetalContourRollData struct {
	Profile   int     `yaml:"profile"`
	AxisLine  int     `yaml:"axisLine"`
	Angle     float64 `yaml:"angle,omitempty"`
	Operation int32   `yaml:"operation,omitempty"`
}

// serializeSheetMetalContourRoll projects a contour-roll recipe to its persisted form,
// erroring if its profile sketch is not in the part.
func serializeSheetMetalContourRoll(def *SheetMetalContourRollDefinition, sk SketchIndexer) (*SheetMetalContourRollData, error) {
	idx, ok := sk.IndexOf(def.Profile)
	if !ok {
		return nil, fmt.Errorf("sheet-metal contour roll references a profile sketch not in the part")
	}
	return &SheetMetalContourRollData{
		Profile: idx, AxisLine: def.AxisLine, Angle: evalFloat(def.Angle), Operation: int32(def.Operation),
	}, nil
}

// restoreSheetMetalContourRoll rebuilds a contour-roll feature, erroring on a missing payload
// or unknown profile sketch. A zero angle restores as nil (a full revolution).
func restoreSheetMetalContourRoll(fs *PartFeatures, d *SheetMetalContourRollData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal contour roll feature is missing its payload")
	}
	profile, ok := sk.At(d.Profile)
	if !ok {
		return nil, fmt.Errorf("sheet-metal contour roll references sketch %d which is not in the part", d.Profile)
	}
	def := &SheetMetalContourRollDefinition{Profile: profile, AxisLine: d.AxisLine, Operation: ops.PartFeatureOperation(d.Operation)}
	if d.Angle != 0 {
		def.Angle = constFloat(d.Angle)
	}
	return NewSheetMetalContourRollFeatures(fs).Add(def), nil
}
