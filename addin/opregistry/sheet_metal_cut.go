// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

const sheetMetalCutSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Index of the sketch whose profile is removed (see model.tree)."},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0, "description": "Which closed profile of the sketch to cut."},
    "direction": {"type": "string", "enum": ["positive", "negative", "symmetric"], "default": "positive", "description": "Which side of the sketch plane to cut toward."},
    "distance": {"type": "string", "description": "Cut depth, e.g. \"5 mm\" (omitted ⇒ cut through all the material)."},
    "acrossBend": {"type": "boolean", "default": false, "description": "Cut across bends (unfold/cut/refold) — reserved for the flat pattern (M13-F04), not yet supported."}
  },
  "required": ["sketchIndex"]
}`

// sheetMetalCutDescriptor is the self-describing "sheetMetalCut" operation: remove a sketch
// profile from a sheet-metal part.
func sheetMetalCutDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalCut,
		Summary: "Cut a closed sketch profile through a sheet-metal part (through all by default, or to a depth).",
		Schema:  json.RawMessage(sheetMetalCutSchema),
		Apply:   applySheetMetalCut,
	}
}

func applySheetMetalCut(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalCut")
	if err != nil {
		return nil, err
	}
	var in featureargs.SheetMetalCut
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalCut: invalid args: %w", err)
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalCutDefinition{
		Sketch: sk, ProfileIndex: in.ProfileIndex,
		Direction: parseExtentDirection(in.Direction), AcrossBend: in.AcrossBend,
	}
	if in.Distance != "" {
		dist, err := lengthClosure(part, in.Distance, "sheetMetalCut: distance")
		if err != nil {
			return nil, err
		}
		def.Distance = dist
	}
	return recomputeResult(part, feature.NewSheetMetalCutFeatures(part.Features()).Add(def))
}
