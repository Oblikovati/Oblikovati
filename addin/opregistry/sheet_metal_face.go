// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

const sheetMetalFaceSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Index of the sketch whose profile becomes the wall (see model.tree)."},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0, "description": "Which closed profile of the sketch to thicken."},
    "operation": {"type": "string", "enum": ["new", "join"], "default": "new", "description": "new for the base wall, join to add a secondary wall to the running sheet."},
    "direction": {"type": "string", "enum": ["positive", "negative", "symmetric"], "default": "positive", "description": "Which side of the sketch plane the material grows toward."}
  },
  "required": ["sketchIndex"]
}`

// sheetMetalFaceDescriptor is the self-describing "sheetMetalFace" operation: thicken a
// closed sketch profile into a sheet-metal wall at the active rule's gauge.
func sheetMetalFaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalFace,
		Summary: "Thicken a closed sketch profile into a sheet-metal wall (the base/secondary Face) at the active rule's thickness.",
		Schema:  json.RawMessage(sheetMetalFaceSchema),
		Apply:   applySheetMetalFace,
	}
}

func applySheetMetalFace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeSheetMetalArgs[featureargs.SheetMetalFace](s, raw, "sheetMetalFace")
	if err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalFaceDefinition{
		Sketch:       sk,
		ProfileIndex: in.ProfileIndex,
		Direction:    parseExtentDirection(in.Direction),
		Operation:    op,
	}
	pf := feature.NewSheetMetalFaceFeatures(part.Features()).Add(def)
	return recomputeResult(part, pf)
}
