// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

const sheetMetalContourRollSchema = `{
  "type": "object",
  "properties": {
    "profileSketch": {"type": "integer", "minimum": 0, "description": "Index of the sketch holding the open profile and the axis centerline."},
    "axisLine": {"type": "integer", "minimum": 0, "description": "Index of the line in that sketch to roll around (a centerline)."},
    "angle": {"type": "string", "description": "Sweep angle, e.g. \"360 deg\" (default — a full tube)."},
    "operation": {"type": "string", "enum": ["new", "join"], "default": "new", "description": "new for a standalone roll, join to merge it onto the running part."}
  },
  "required": ["profileSketch", "axisLine"]
}`

// sheetMetalContourRollDescriptor is the self-describing "sheetMetalContourRoll" operation:
// revolve an open profile around an axis into a rolled shell.
func sheetMetalContourRollDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalContourRoll,
		Summary: "Revolve an open sketch profile around an axis line into a rolled sheet-metal shell (a tube/cone), at the rule's thickness.",
		Schema:  json.RawMessage(sheetMetalContourRollSchema),
		Apply:   applySheetMetalContourRoll,
	}
}

func applySheetMetalContourRoll(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeSheetMetalArgs[featureargs.SheetMetalContourRoll](s, raw, "sheetMetalContourRoll")
	if err != nil {
		return nil, err
	}
	profile, err := sketchAt(part, in.ProfileSketch)
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	angle, err := angleOverride(part, in.Angle, "sheetMetalContourRoll: angle")
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalContourRollDefinition{Profile: profile, AxisLine: in.AxisLine, Angle: angle, Operation: op}
	return recomputeResult(part, feature.NewSheetMetalContourRollFeatures(part.Features()).Add(def))
}
