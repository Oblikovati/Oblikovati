// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SheetMetalHemData is the serialized form of a SheetMetalHemFeature: the picked edge (base64
// reference key), the hem length, the hem type, the open-hem gap, and the flip. Thickness is
// read live from the rule, so it is not stored.
type SheetMetalHemData struct {
	Edge   string  `yaml:"edge"`
	Length float64 `yaml:"length"`
	Type   int32   `yaml:"type,omitempty"`
	Gap    float64 `yaml:"gap,omitempty"`
	Flip   bool    `yaml:"flip,omitempty"`
}

// serializeSheetMetalHem projects a hem recipe to its persisted form.
func serializeSheetMetalHem(def *SheetMetalHemDefinition) *SheetMetalHemData {
	return &SheetMetalHemData{
		Edge:   encodeKey(def.EdgeKey),
		Length: evalFloat(def.Length),
		Type:   int32(def.Type),
		Gap:    evalFloat(def.Gap),
		Flip:   def.Flip,
	}
}

// restoreSheetMetalHem rebuilds a hem feature, erroring on a missing payload or an undecodable
// edge key.
func restoreSheetMetalHem(fs *PartFeatures, d *SheetMetalHemData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal hem feature is missing its payload")
	}
	key, err := decodeKey(d.Edge)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal hem edge key: %w", err)
	}
	def := &SheetMetalHemDefinition{
		EdgeKey: key,
		Length:  constFloat(d.Length),
		Type:    HemType(d.Type),
		Flip:    d.Flip,
	}
	if d.Gap != 0 {
		def.Gap = constFloat(d.Gap)
	}
	return NewSheetMetalHemFeatures(fs).Add(def), nil
}
