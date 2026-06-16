// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SheetMetalFoldData is the serialized form of a SheetMetalFoldFeature: the fold line (sketch
// index + line index), the optional bend angle/radius (0 ⇒ default 90° / the rule's bend
// radius), the bend location, and the flip.
type SheetMetalFoldData struct {
	Sketch   int     `yaml:"sketch"`
	Line     int     `yaml:"line"`
	Angle    float64 `yaml:"angle,omitempty"`  // 0 ⇒ 90° default
	Radius   float64 `yaml:"radius,omitempty"` // 0 ⇒ rule bend radius
	Location int32   `yaml:"location,omitempty"`
	Flip     bool    `yaml:"flip,omitempty"`
}

// serializeSheetMetalFold projects a fold recipe to its persisted form, erroring if its
// sketch is not in the part.
func serializeSheetMetalFold(def *SheetMetalFoldDefinition, sk SketchIndexer) (*SheetMetalFoldData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal fold references a sketch that is not in the part")
	}
	return &SheetMetalFoldData{
		Sketch: idx, Line: def.LineIndex,
		Angle: evalFloat(def.Angle), Radius: evalFloat(def.Radius),
		Location: int32(def.Location), Flip: def.Flip,
	}, nil
}

// restoreSheetMetalFold rebuilds a fold feature, erroring on a missing payload or unknown
// sketch. A zero angle/radius restores as nil (default 90° / rule radius).
func restoreSheetMetalFold(fs *PartFeatures, d *SheetMetalFoldData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal fold feature is missing its payload")
	}
	s, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal fold references sketch %d which is not in the part", d.Sketch)
	}
	def := &SheetMetalFoldDefinition{Sketch: s, LineIndex: d.Line, Location: BendLocation(d.Location), Flip: d.Flip}
	if d.Angle != 0 {
		def.Angle = constFloat(d.Angle)
	}
	if d.Radius != 0 {
		def.Radius = constFloat(d.Radius)
	}
	return NewSheetMetalFoldFeatures(fs).Add(def), nil
}
