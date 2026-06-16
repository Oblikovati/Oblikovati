// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// sheetMetalFaceArgs is the argument shape for the "sheetMetalFace" operation. Thickness is
// intentionally absent: a Face is thickened by the active rule, so it carries no per-feature
// thickness (edit the rule to change gauge).
type sheetMetalFaceArgs struct {
	SketchIndex  int    `json:"sketchIndex"`
	ProfileIndex int    `json:"profileIndex"`
	Operation    string `json:"operation"`           // new (base wall) | join (secondary wall); default new
	Direction    string `json:"direction,omitempty"` // positive|negative|symmetric material side; default positive
}

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
		Name:    "sheetMetalFace",
		Summary: "Thicken a closed sketch profile into a sheet-metal wall (the base/secondary Face) at the active rule's thickness.",
		Schema:  json.RawMessage(sheetMetalFaceSchema),
		Apply:   applySheetMetalFace,
	}
}

func applySheetMetalFace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	if !part.IsSheetMetal() {
		return nil, fmt.Errorf("sheetMetalFace: the active part is not a sheet-metal part")
	}
	var in sheetMetalFaceArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalFace: invalid args: %w", err)
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
