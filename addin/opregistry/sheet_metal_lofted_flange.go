// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// sheetMetalLoftedFlangeArgs is the argument shape for the "sheetMetalLoftedFlange" operation:
// the two open-profile sketch indices and the boolean operation. Thickness comes from the rule.
type sheetMetalLoftedFlangeArgs struct {
	ProfileA  int    `json:"profileA"`
	ProfileB  int    `json:"profileB"`
	Operation string `json:"operation"` // new (default) | join
}

const sheetMetalLoftedFlangeSchema = `{
  "type": "object",
  "properties": {
    "profileA": {"type": "integer", "minimum": 0, "description": "Index of the sketch holding the first open profile."},
    "profileB": {"type": "integer", "minimum": 0, "description": "Index of the sketch holding the second open profile (same vertex count as A)."},
    "operation": {"type": "string", "enum": ["new", "join"], "default": "new", "description": "new for a standalone transition wall, join to merge it onto the running part."}
  },
  "required": ["profileA", "profileB"]
}`

// sheetMetalLoftedFlangeDescriptor is the self-describing "sheetMetalLoftedFlange" operation:
// loft a constant-thickness wall between two open profiles.
func sheetMetalLoftedFlangeDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    "sheetMetalLoftedFlange",
		Summary: "Loft a constant-thickness sheet-metal wall between two open profiles (a transition piece), at the rule's thickness.",
		Schema:  json.RawMessage(sheetMetalLoftedFlangeSchema),
		Apply:   applySheetMetalLoftedFlange,
	}
}

func applySheetMetalLoftedFlange(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalLoftedFlange")
	if err != nil {
		return nil, err
	}
	var in sheetMetalLoftedFlangeArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalLoftedFlange: invalid args: %w", err)
	}
	profileA, err := sketchAt(part, in.ProfileA)
	if err != nil {
		return nil, err
	}
	profileB, err := sketchAt(part, in.ProfileB)
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalLoftedFlangeDefinition{ProfileA: profileA, ProfileB: profileB, Operation: op}
	return recomputeResult(part, feature.NewSheetMetalLoftedFlangeFeatures(part.Features()).Add(def))
}
