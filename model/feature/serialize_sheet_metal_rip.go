// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SheetMetalRipData is the serialized form of a SheetMetalRipFeature. A sketch-line rip stores its
// sketch + line index; a face-based rip (#1965) stores the RipFace and any point vertices as
// base64 reference keys, with Sketch = -1 to mark that there is no sketch line. Type and GapSide
// ride both forms; a record without GapSide (legacy) restores as the symmetric rip it was.
type SheetMetalRipData struct {
	Sketch   int     `yaml:"sketch"`
	Line     int     `yaml:"line"`
	Gap      float64 `yaml:"gap"`
	Type     int32   `yaml:"type,omitempty"`
	GapSide  string  `yaml:"gapSide,omitempty"`
	Face     string  `yaml:"face,omitempty"`
	Point    string  `yaml:"point,omitempty"`
	PointTwo string  `yaml:"pointTwo,omitempty"`
}

// serializeSheetMetalRip projects a rip recipe to its persisted form. A sketch-line rip errors if
// its sketch is not in the part (so the rip line cannot be lost silently); a face-based rip carries
// no sketch and marks that with Sketch = -1.
func serializeSheetMetalRip(def *SheetMetalRipDefinition, sk SketchIndexer) (*SheetMetalRipData, error) {
	d := &SheetMetalRipData{
		Sketch: -1, Line: def.LineIndex, Gap: evalFloat(def.Gap), Type: int32(def.Type),
		GapSide: ripGapSideName(def.GapSide), Face: encodeKey(def.FaceKey),
		Point: encodeKey(def.PointKey), PointTwo: encodeKey(def.PointTwoKey),
	}
	if def.Sketch != nil {
		idx, ok := sk.IndexOf(def.Sketch)
		if !ok {
			return nil, fmt.Errorf("sheet-metal rip references a sketch that is not in the part")
		}
		d.Sketch = idx
	}
	return d, nil
}

// restoreSheetMetalRip rebuilds a rip feature, erroring on a missing payload, an unknown sketch,
// or an undecodable reference key.
func restoreSheetMetalRip(fs *PartFeatures, d *SheetMetalRipData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal rip feature is missing its payload")
	}
	def, err := ripDefinitionFromData(d, sk)
	if err != nil {
		return nil, err
	}
	return NewSheetMetalRipFeatures(fs).Add(def), nil
}

// ripDefinitionFromData rebuilds a rip recipe from its persisted form, resolving the sketch (when
// present) and the face/point reference keys.
func ripDefinitionFromData(d *SheetMetalRipData, sk SketchIndexer) (*SheetMetalRipDefinition, error) {
	side, _ := ParseRipGapSide(d.GapSide) // a bad spelling falls back to symmetric, never errors on load
	def := &SheetMetalRipDefinition{LineIndex: d.Line, Type: RipType(d.Type), GapSide: side}
	if d.Gap != 0 {
		def.Gap = constFloat(d.Gap)
	}
	if d.Sketch >= 0 {
		s, ok := sk.At(d.Sketch)
		if !ok {
			return nil, fmt.Errorf("sheet-metal rip references sketch %d which is not in the part", d.Sketch)
		}
		def.Sketch = s
	}
	for name, dst := range map[string]*[]byte{"face": &def.FaceKey, "point": &def.PointKey, "pointTwo": &def.PointTwoKey} {
		key, err := decodeRipKey(d, name)
		if err != nil {
			return nil, err
		}
		*dst = key
	}
	return def, nil
}

// decodeRipKey decodes one of the rip's base64 reference keys by field name, erroring on an
// undecodable value.
func decodeRipKey(d *SheetMetalRipData, field string) ([]byte, error) {
	encoded := map[string]string{"face": d.Face, "point": d.Point, "pointTwo": d.PointTwo}[field]
	if encoded == "" {
		return nil, nil
	}
	key, err := decodeKey(encoded)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal rip %s key: %w", field, err)
	}
	return key, nil
}
