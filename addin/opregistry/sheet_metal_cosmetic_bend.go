// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

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
		Name:    featureargs.KindSheetMetalCosmeticBend,
		Summary: "Mark a cosmetic bend line on a sheet-metal part — it joins the bend table without deforming the model.",
		Schema:  json.RawMessage(sheetMetalCosmeticBendSchema),
		Apply:   applySheetMetalCosmeticBend,
	}
}

func applySheetMetalCosmeticBend(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeSheetMetalArgs[featureargs.SheetMetalCosmeticBend](s, raw, "sheetMetalCosmeticBend")
	if err != nil {
		return nil, err
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
