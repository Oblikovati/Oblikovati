// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SheetMetalBendData is the serialized form of a SheetMetalBendFeature: the bend line (sketch
// index + line index), the optional bend angle/radius (0 ⇒ default 90° / the rule's bend
// radius), and the flip. Thickness is read live from the rule.
type SheetMetalBendData struct {
	Sketch int     `yaml:"sketch"`
	Line   int     `yaml:"line"`
	Angle  float64 `yaml:"angle,omitempty"`  // 0 ⇒ 90° default
	Radius float64 `yaml:"radius,omitempty"` // 0 ⇒ rule bend radius
	Flip   bool    `yaml:"flip,omitempty"`
}

// serializeSheetMetalBend projects a bend recipe to its persisted form, erroring if its
// sketch is not in the part (so the bend line cannot be lost silently).
func serializeSheetMetalBend(def *SheetMetalBendDefinition, sk SketchIndexer) (*SheetMetalBendData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal bend references a sketch that is not in the part")
	}
	return &SheetMetalBendData{
		Sketch: idx, Line: def.LineIndex,
		Angle: evalFloat(def.Angle), Radius: evalFloat(def.Radius), Flip: def.Flip,
	}, nil
}

// restoreSheetMetalBend rebuilds a bend feature, erroring on a missing payload or unknown
// sketch. A zero angle/radius restores as nil (default 90° / rule radius).
func restoreSheetMetalBend(fs *PartFeatures, d *SheetMetalBendData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal bend feature is missing its payload")
	}
	s, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal bend references sketch %d which is not in the part", d.Sketch)
	}
	def := &SheetMetalBendDefinition{Sketch: s, LineIndex: d.Line, Flip: d.Flip}
	if d.Angle != 0 {
		def.Angle = constFloat(d.Angle)
	}
	if d.Radius != 0 {
		def.Radius = constFloat(d.Radius)
	}
	return NewSheetMetalBendFeatures(fs).Add(def), nil
}
