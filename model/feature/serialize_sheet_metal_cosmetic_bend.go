// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SheetMetalCosmeticBendData is the serialized form of a SheetMetalCosmeticBendFeature: the
// bend line (sketch index + line index) and the optional angle/radius (0 ⇒ default 90° / the
// rule's bend radius). Thickness is read live from the rule; there is no flip (a cosmetic bend
// does not fold).
type SheetMetalCosmeticBendData struct {
	Sketch int     `yaml:"sketch"`
	Line   int     `yaml:"line"`
	Angle  float64 `yaml:"angle,omitempty"`  // 0 ⇒ 90° default
	Radius float64 `yaml:"radius,omitempty"` // 0 ⇒ rule bend radius
}

// serializeSheetMetalCosmeticBend projects a cosmetic-bend recipe to its persisted form,
// erroring if its sketch is not in the part (so the bend line cannot be lost silently).
func serializeSheetMetalCosmeticBend(def *SheetMetalCosmeticBendDefinition, sk SketchIndexer) (*SheetMetalCosmeticBendData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal cosmetic bend references a sketch that is not in the part")
	}
	return &SheetMetalCosmeticBendData{
		Sketch: idx, Line: def.LineIndex,
		Angle: evalFloat(def.Angle), Radius: evalFloat(def.Radius),
	}, nil
}

// restoreSheetMetalCosmeticBend rebuilds a cosmetic-bend feature, erroring on a missing payload
// or unknown sketch. A zero angle/radius restores as nil (default 90° / rule radius).
func restoreSheetMetalCosmeticBend(fs *PartFeatures, d *SheetMetalCosmeticBendData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal cosmetic bend feature is missing its payload")
	}
	s, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal cosmetic bend references sketch %d which is not in the part", d.Sketch)
	}
	def := &SheetMetalCosmeticBendDefinition{Sketch: s, LineIndex: d.Line}
	if d.Angle != 0 {
		def.Angle = constFloat(d.Angle)
	}
	if d.Radius != 0 {
		def.Radius = constFloat(d.Radius)
	}
	return NewSheetMetalCosmeticBendFeatures(fs).Add(def), nil
}
