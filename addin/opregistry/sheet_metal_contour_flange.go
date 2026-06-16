// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// sheetMetalContourFlangeArgs is the argument shape for the "sheetMetalContourFlange"
// operation: the edge to sweep along, the open-profile sketch index, and a flip. Thickness
// comes from the rule.
type sheetMetalContourFlangeArgs struct {
	Edge          string `json:"edge"`
	ProfileSketch int    `json:"profileSketch"`
	Flip          bool   `json:"flip,omitempty"`
}

const sheetMetalContourFlangeSchema = `{
  "type": "object",
  "properties": {
    "edge": {"type": "string", "description": "Reference key of the straight sheet edge to sweep the profile along (from get_reference_keys)."},
    "profileSketch": {"type": "integer", "minimum": 0, "description": "Index of the sketch holding the open profile (the flange cross-section, a chain of lines starting at the edge)."},
    "flip": {"type": "boolean", "default": false, "description": "Sweep toward the opposite side of the sheet."}
  },
  "required": ["edge", "profileSketch"]
}`

// sheetMetalContourFlangeDescriptor is the self-describing "sheetMetalContourFlange" operation:
// sweep an open profile along a sheet edge into a contour flange.
func sheetMetalContourFlangeDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    "sheetMetalContourFlange",
		Summary: "Sweep an open sketch profile (the wall cross-section) along a sheet-metal edge into a contour flange, at the rule's thickness.",
		Schema:  json.RawMessage(sheetMetalContourFlangeSchema),
		Apply:   applySheetMetalContourFlange,
	}
}

func applySheetMetalContourFlange(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalContourFlange")
	if err != nil {
		return nil, err
	}
	var in sheetMetalContourFlangeArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalContourFlange: invalid args: %w", err)
	}
	if in.Edge == "" {
		return nil, fmt.Errorf("sheetMetalContourFlange: edge is required")
	}
	profile, err := sketchAt(part, in.ProfileSketch)
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalContourFlangeDefinition{EdgeKey: []byte(in.Edge), Profile: profile, Flip: in.Flip}
	return recomputeResult(part, feature.NewSheetMetalContourFlangeFeatures(part.Features()).Add(def))
}
