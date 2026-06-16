// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SheetMetalCornerData is the serialized form of a SheetMetalCornerFeature: the corner edges
// (base64 reference keys), the treatment, and its size.
type SheetMetalCornerData struct {
	Edges     []string `yaml:"edges"`
	Treatment int32    `yaml:"treatment,omitempty"`
	Size      float64  `yaml:"size"`
}

// serializeSheetMetalCorner projects a corner recipe to its persisted form.
func serializeSheetMetalCorner(def *SheetMetalCornerDefinition) *SheetMetalCornerData {
	return &SheetMetalCornerData{
		Edges:     encodeKeys(def.EdgeKeys),
		Treatment: int32(def.Treatment),
		Size:      evalFloat(def.Size),
	}
}

// restoreSheetMetalCorner rebuilds a corner feature, erroring on a missing payload or an
// undecodable edge key.
func restoreSheetMetalCorner(fs *PartFeatures, d *SheetMetalCornerData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal corner feature is missing its payload")
	}
	keys, err := decodeKeys(d.Edges)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal corner edge keys: %w", err)
	}
	return NewSheetMetalCornerFeatures(fs).Add(&SheetMetalCornerDefinition{
		EdgeKeys:  keys,
		Treatment: CornerTreatment(d.Treatment),
		Size:      constFloat(d.Size),
	}), nil
}
