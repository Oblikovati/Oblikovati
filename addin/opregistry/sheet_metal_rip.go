// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

const sheetMetalRipSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Index of the sketch holding the rip line (see model.tree)."},
    "lineIndex": {"type": "integer", "minimum": 0, "default": 0, "description": "Which line of the sketch is the rip seam."},
    "gap": {"type": "string", "default": "0.1 mm", "description": "Width of the slit the rip opens, e.g. \"0.1 mm\"."}
  },
  "required": ["sketchIndex"]
}`

// sheetMetalRipDescriptor is the self-describing "sheetMetalRip" operation: cut a narrow slit
// along a sketch line so a closed or folded sheet can be developed flat.
func sheetMetalRipDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalRip,
		Summary: "Rip a sheet-metal part along a sketch line — a narrow through-thickness slit that opens a seam for unfolding.",
		Schema:  json.RawMessage(sheetMetalRipSchema),
		Apply:   applySheetMetalRip,
	}
}

// defaultRipGap is the slit width used when none is given — a thin manufacturing kerf (0.1 mm).
const defaultRipGap = "0.1 mm"

func applySheetMetalRip(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalRip")
	if err != nil {
		return nil, err
	}
	var in featureargs.SheetMetalRip
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalRip: invalid args: %w", err)
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	gapExpr := in.Gap
	if gapExpr == "" {
		gapExpr = defaultRipGap
	}
	gap, err := lengthClosure(part, gapExpr, "sheetMetalRip: gap")
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalRipDefinition{Sketch: sk, LineIndex: in.LineIndex, Gap: gap}
	return recomputeResult(part, feature.NewSheetMetalRipFeatures(part.Features()).Add(def))
}
