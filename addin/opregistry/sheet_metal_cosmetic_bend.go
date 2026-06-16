// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// sheetMetalCosmeticBendArgs is the argument shape for the "sheetMetalCosmeticBend" operation:
// the sketch bend line (sketch + line index) and the optional bend angle/radius. It marks a
// fold for manufacturing without deforming the model.
type sheetMetalCosmeticBendArgs struct {
	SketchIndex int    `json:"sketchIndex"`
	LineIndex   int    `json:"lineIndex"`
	Angle       string `json:"angle,omitempty"`
	Radius      string `json:"radius,omitempty"`
}

const sheetMetalCosmeticBendSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Index of the sketch holding the cosmetic bend line (see model.tree)."},
    "lineIndex": {"type": "integer", "minimum": 0, "default": 0, "description": "Which line of the sketch is the bend axis."},
    "angle": {"type": "string", "description": "Bend angle, e.g. \"90 deg\" (default), for the bend table."},
    "radius": {"type": "string", "description": "Inside bend radius (default: the rule's bend radius)."}
  },
  "required": ["sketchIndex"]
}`

// sheetMetalCosmeticBendDescriptor is the self-describing "sheetMetalCosmeticBend" operation:
// annotate a fold line for manufacturing without folding the geometry.
func sheetMetalCosmeticBendDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    "sheetMetalCosmeticBend",
		Summary: "Mark a cosmetic bend line on a sheet-metal part — it joins the bend table without deforming the model.",
		Schema:  json.RawMessage(sheetMetalCosmeticBendSchema),
		Apply:   applySheetMetalCosmeticBend,
	}
}

func applySheetMetalCosmeticBend(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalCosmeticBend")
	if err != nil {
		return nil, err
	}
	var in sheetMetalCosmeticBendArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalCosmeticBend: invalid args: %w", err)
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	angle, radius, err := optionalBendDims(part, in.Angle, in.Radius, "sheetMetalCosmeticBend")
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalCosmeticBendDefinition{Sketch: sk, LineIndex: in.LineIndex, Angle: angle, Radius: radius}
	return recomputeResult(part, feature.NewSheetMetalCosmeticBendFeatures(part.Features()).Add(def))
}
