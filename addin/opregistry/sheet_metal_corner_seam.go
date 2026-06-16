// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// sheetMetalCornerSeamArgs is the argument shape for the "sheetMetalCornerSeam" operation: the
// corner edges (the shared through-thickness edges where flanges meet), the gap, and the seam
// type.
type sheetMetalCornerSeamArgs struct {
	Edges []string `json:"edges"`
	Gap   string   `json:"gap"`
	Type  string   `json:"type,omitempty"` // gap (default)
}

const sheetMetalCornerSeamSchema = `{
  "type": "object",
  "properties": {
    "edges": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the corner edges where flanges meet (from get_reference_keys)."},
    "gap": {"type": "string", "description": "Seam gap, e.g. \"0.2 mm\" — the relief left between the two corner walls."},
    "type": {"type": "string", "enum": ["gap"], "default": "gap", "description": "The seam relief type."}
  },
  "required": ["edges", "gap"]
}`

// sheetMetalCornerSeamDescriptor is the self-describing "sheetMetalCornerSeam" operation:
// relieve the corner where two flanges meet with a gap.
func sheetMetalCornerSeamDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    "sheetMetalCornerSeam",
		Summary: "Relieve the corner where two sheet-metal flanges meet with a gap seam (a square notch of the given gap along the shared corner edges).",
		Schema:  json.RawMessage(sheetMetalCornerSeamSchema),
		Apply:   applySheetMetalCornerSeam,
	}
}

func applySheetMetalCornerSeam(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalCornerSeam")
	if err != nil {
		return nil, err
	}
	var in sheetMetalCornerSeamArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalCornerSeam: invalid args: %w", err)
	}
	if len(in.Edges) == 0 {
		return nil, fmt.Errorf("sheetMetalCornerSeam: edges is empty")
	}
	seam, ok := feature.ParseSeamType(in.Type)
	if !ok {
		return nil, fmt.Errorf("sheetMetalCornerSeam: unknown type %q (want gap)", in.Type)
	}
	gap, err := lengthClosure(part, in.Gap, "sheetMetalCornerSeam: gap")
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalCornerSeamDefinition{EdgeKeys: refKeys(in.Edges), Gap: gap, Type: seam}
	return recomputeResult(part, feature.NewSheetMetalCornerSeamFeatures(part.Features()).Add(def))
}
