// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

const sheetMetalPunchSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Index of the sketch whose closed profiles are punched (see model.tree)."},
    "depth": {"type": "string", "description": "Punch depth (default: through all the material)."}
  },
  "required": ["sketchIndex"]
}`

// sheetMetalPunchDescriptor is the self-describing "sheetMetalPunch" operation: stamp every
// closed profile of a sketch through the sheet in one die-pattern punch.
func sheetMetalPunchDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalPunch,
		Summary: "Punch every closed profile of a sketch through a sheet-metal part — a die pattern (vents, louvers, perforations) in one operation.",
		Schema:  json.RawMessage(sheetMetalPunchSchema),
		Apply:   applySheetMetalPunch,
	}
}

func applySheetMetalPunch(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalPunch")
	if err != nil {
		return nil, err
	}
	var in featureargs.SheetMetalPunch
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalPunch: invalid args: %w", err)
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalPunchDefinition{Sketch: sk}
	if in.Depth != "" {
		depth, err := lengthClosure(part, in.Depth, "sheetMetalPunch: depth")
		if err != nil {
			return nil, err
		}
		def.Depth = depth
	}
	return recomputeResult(part, feature.NewSheetMetalPunchFeatures(part.Features()).Add(def))
}
