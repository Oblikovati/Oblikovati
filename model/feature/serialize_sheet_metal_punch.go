// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SheetMetalPunchData is the serialized form of a SheetMetalPunchFeature: the sketch index, the
// punch direction, and the optional depth (0 ⇒ through all).
type SheetMetalPunchData struct {
	Sketch    int     `yaml:"sketch"`
	Direction int32   `yaml:"direction,omitempty"`
	Depth     float64 `yaml:"depth,omitempty"` // 0 ⇒ through all
}

// serializeSheetMetalPunch projects a punch recipe to its persisted form, erroring if its
// sketch is not in the part.
func serializeSheetMetalPunch(def *SheetMetalPunchDefinition, sk SketchIndexer) (*SheetMetalPunchData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal punch references a sketch that is not in the part")
	}
	return &SheetMetalPunchData{Sketch: idx, Direction: int32(def.Direction), Depth: evalFloat(def.Depth)}, nil
}

// restoreSheetMetalPunch rebuilds a punch feature, erroring on a missing payload or unknown
// sketch. A zero depth restores as nil (through all).
func restoreSheetMetalPunch(fs *PartFeatures, d *SheetMetalPunchData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal punch feature is missing its payload")
	}
	s, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal punch references sketch %d which is not in the part", d.Sketch)
	}
	def := &SheetMetalPunchDefinition{Sketch: s, Direction: ExtentDirection(d.Direction)}
	if d.Depth != 0 {
		def.Depth = constFloat(d.Depth)
	}
	return NewSheetMetalPunchFeatures(fs).Add(def), nil
}
