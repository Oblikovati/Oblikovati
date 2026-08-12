// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
)

// SheetMetalPunchData is the serialized form of a SheetMetalPunchFeature: the sketch index, the
// punch direction, the optional depth (0 ⇒ through all), and the die-tool settings (#1968) — the
// rotation angle, the across-bends / unfold-in-flat flags, the tool id, and the representation type.
type SheetMetalPunchData struct {
	Sketch         int     `yaml:"sketch"`
	Direction      int32   `yaml:"direction,omitempty"`
	Depth          float64 `yaml:"depth,omitempty"` // 0 ⇒ through all
	Angle          float64 `yaml:"angle,omitempty"`
	AcrossBends    bool    `yaml:"acrossBends,omitempty"`
	UnfoldInFlat   bool    `yaml:"unfoldInFlat,omitempty"`
	ToolID         string  `yaml:"toolId,omitempty"`
	Representation int32   `yaml:"representation,omitempty"`
}

// serializeSheetMetalPunch projects a punch recipe to its persisted form, erroring if its
// sketch is not in the part.
func serializeSheetMetalPunch(def *SheetMetalPunchDefinition, sk SketchIndexer) (*SheetMetalPunchData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal punch references a sketch that is not in the part")
	}
	return &SheetMetalPunchData{
		Sketch: idx, Direction: int32(def.Direction), Depth: evalFloat(def.Depth), Angle: evalFloat(def.Angle),
		AcrossBends: def.AcrossBends, UnfoldInFlat: def.UnfoldInFlat, ToolID: def.ToolID,
		Representation: int32(def.Representation),
	}, nil
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
	def := &SheetMetalPunchDefinition{
		Sketch: s, Direction: ExtentDirection(d.Direction), AcrossBends: d.AcrossBends,
		UnfoldInFlat: d.UnfoldInFlat, ToolID: d.ToolID, Representation: types.PunchRepresentationType(d.Representation),
	}
	if d.Depth != 0 {
		def.Depth = constFloat(d.Depth)
	}
	if d.Angle != 0 {
		def.Angle = constFloat(d.Angle)
	}
	return NewSheetMetalPunchFeatures(fs).Add(def), nil
}
