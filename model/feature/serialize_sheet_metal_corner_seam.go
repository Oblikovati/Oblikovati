// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
)

// SheetMetalCornerSeamData is the serialized form of a SheetMetalCornerSeamFeature: the corner
// edges (base64 reference keys), the gap, the seam type, and — for the lap styles (#1964) — the
// overlap percentage, the seam-root relief, and how the gap is measured. The lap fields ride the
// record even though only the gap style is modelled today (#2085), so a saved overlap seam keeps
// its intent when the geometry lands.
type SheetMetalCornerSeamData struct {
	Edges          []string `yaml:"edges"`
	Gap            float64  `yaml:"gap"`
	Type           int32    `yaml:"type,omitempty"`
	Overlap        float64  `yaml:"overlap,omitempty"`
	ReliefShape    int32    `yaml:"reliefShape,omitempty"`
	ReliefSize     float64  `yaml:"reliefSize,omitempty"`
	DefinitionType int32    `yaml:"definitionType,omitempty"`
}

// serializeSheetMetalCornerSeam projects a corner-seam recipe to its persisted form.
func serializeSheetMetalCornerSeam(def *SheetMetalCornerSeamDefinition) *SheetMetalCornerSeamData {
	return &SheetMetalCornerSeamData{
		Edges: encodeKeys(def.EdgeKeys), Gap: evalFloat(def.Gap), Type: int32(def.Type),
		Overlap: def.Overlap, ReliefShape: int32(def.ReliefShape), ReliefSize: evalFloat(def.ReliefSize),
		DefinitionType: int32(def.DefinitionType),
	}
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
	def := &SheetMetalCornerSeamDefinition{
		EdgeKeys: keys, Gap: constFloat(d.Gap), Type: SeamType(d.Type), Overlap: d.Overlap,
		ReliefShape: types.CornerReliefShape(d.ReliefShape), DefinitionType: types.CornerSeamDefinitionType(d.DefinitionType),
	}
	if d.ReliefSize != 0 {
		def.ReliefSize = constFloat(d.ReliefSize)
	}
	return NewSheetMetalCornerSeamFeatures(fs).Add(def), nil
}
