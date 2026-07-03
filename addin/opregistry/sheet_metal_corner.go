// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

const sheetMetalCornerSchema = `{
  "type": "object",
  "properties": {
    "edges": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the through-thickness corner edges to finish (from get_reference_keys)."},
    "treatment": {"type": "string", "enum": ["chamfer", "round"], "description": "chamfer cuts a flat across the corner; round rolls a fillet."},
    "size": {"type": "string", "description": "Chamfer setback or round radius, e.g. \"3 mm\"."}
  },
  "required": ["edges", "treatment", "size"]
}`

// sheetMetalCornerDescriptor is the self-describing "sheetMetalCorner" operation: chamfer or
// round one or more sheet-metal corners.
func sheetMetalCornerDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalCorner,
		Summary: "Chamfer or round one or more corners of a sheet-metal face (the through-thickness corner edges).",
		Schema:  json.RawMessage(sheetMetalCornerSchema),
		Apply:   applySheetMetalCorner,
	}
}

func applySheetMetalCorner(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalCorner")
	if err != nil {
		return nil, err
	}
	var in featureargs.SheetMetalCorner
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalCorner: invalid args: %w", err)
	}
	if len(in.Edges) == 0 {
		return nil, fmt.Errorf("sheetMetalCorner: edges is empty")
	}
	treatment, ok := feature.ParseCornerTreatment(in.Treatment)
	if !ok {
		return nil, fmt.Errorf("sheetMetalCorner: unknown treatment %q (want chamfer or round)", in.Treatment)
	}
	size, err := lengthClosure(part, in.Size, "sheetMetalCorner: size")
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalCornerDefinition{EdgeKeys: refKeys(in.Edges), Treatment: treatment, Size: size}
	return recomputeResult(part, feature.NewSheetMetalCornerFeatures(part.Features()).Add(def))
}
