// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SheetMetalCornerSeamData is the serialized form of a SheetMetalCornerSeamFeature: the corner
// edges (base64 reference keys), the gap, and the seam type.
type SheetMetalCornerSeamData struct {
	Edges []string `yaml:"edges"`
	Gap   float64  `yaml:"gap"`
	Type  int32    `yaml:"type,omitempty"`
}

// serializeSheetMetalCornerSeam projects a corner-seam recipe to its persisted form.
func serializeSheetMetalCornerSeam(def *SheetMetalCornerSeamDefinition) *SheetMetalCornerSeamData {
	return &SheetMetalCornerSeamData{Edges: encodeKeys(def.EdgeKeys), Gap: evalFloat(def.Gap), Type: int32(def.Type)}
}

// restoreSheetMetalCornerSeam rebuilds a corner-seam feature, erroring on a missing payload or
// an undecodable edge key.
func restoreSheetMetalCornerSeam(fs *PartFeatures, d *SheetMetalCornerSeamData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal corner seam feature is missing its payload")
	}
	keys, err := decodeKeys(d.Edges)
	if err != nil {
		return nil, fmt.Errorf("sheet-metal corner seam edge keys: %w", err)
	}
	return NewSheetMetalCornerSeamFeatures(fs).Add(&SheetMetalCornerSeamDefinition{
		EdgeKeys: keys, Gap: constFloat(d.Gap), Type: SeamType(d.Type),
	}), nil
}
