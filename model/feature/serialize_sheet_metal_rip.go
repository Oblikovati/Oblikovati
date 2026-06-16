// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SheetMetalRipData is the serialized form of a SheetMetalRipFeature: the rip line (sketch index
// + line index) and the gap width.
type SheetMetalRipData struct {
	Sketch int     `yaml:"sketch"`
	Line   int     `yaml:"line"`
	Gap    float64 `yaml:"gap"`
}

// serializeSheetMetalRip projects a rip recipe to its persisted form, erroring if its sketch is
// not in the part (so the rip line cannot be lost silently).
func serializeSheetMetalRip(def *SheetMetalRipDefinition, sk SketchIndexer) (*SheetMetalRipData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal rip references a sketch that is not in the part")
	}
	return &SheetMetalRipData{Sketch: idx, Line: def.LineIndex, Gap: evalFloat(def.Gap)}, nil
}

// restoreSheetMetalRip rebuilds a rip feature, erroring on a missing payload or unknown sketch.
func restoreSheetMetalRip(fs *PartFeatures, d *SheetMetalRipData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal rip feature is missing its payload")
	}
	s, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("sheet-metal rip references sketch %d which is not in the part", d.Sketch)
	}
	def := &SheetMetalRipDefinition{Sketch: s, LineIndex: d.Line}
	if d.Gap != 0 {
		def.Gap = constFloat(d.Gap)
	}
	return NewSheetMetalRipFeatures(fs).Add(def), nil
}
