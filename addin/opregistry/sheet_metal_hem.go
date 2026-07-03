// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

const sheetMetalHemSchema = `{
  "type": "object",
  "properties": {
    "edge": {"type": "string", "description": "Reference key of the straight sheet edge to hem (from get_reference_keys)."},
    "length": {"type": "string", "description": "How far the folded-back wall runs, e.g. \"6 mm\"."},
    "type": {"type": "string", "enum": ["closed", "open"], "default": "closed", "description": "closed folds tight (radius ~ t/2); open leaves a rounded loop of the gap."},
    "gap": {"type": "string", "description": "Open-hem loop gap (radius = gap/2); ignored for a closed hem."},
    "flip": {"type": "boolean", "default": false, "description": "Fold toward the opposite side of the sheet."}
  },
  "required": ["edge", "length"]
}`

// sheetMetalHemDescriptor is the self-describing "sheetMetalHem" operation: fold a sheet edge
// back on itself.
func sheetMetalHemDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalHem,
		Summary: "Fold a sheet-metal edge back on itself (a hem): closed (tight) or open (a rounded loop of the given gap).",
		Schema:  json.RawMessage(sheetMetalHemSchema),
		Apply:   applySheetMetalHem,
	}
}

func applySheetMetalHem(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalHem")
	if err != nil {
		return nil, err
	}
	var in featureargs.SheetMetalHem
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalHem: invalid args: %w", err)
	}
	if in.Edge == "" {
		return nil, fmt.Errorf("sheetMetalHem: edge is required")
	}
	def, err := hemDef(part, in)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, feature.NewSheetMetalHemFeatures(part.Features()).Add(def))
}

// hemDef resolves the hem args into a definition: the edge, the length closure, the type, and
// the optional open-hem gap.
func hemDef(part *compdef.PartComponentDefinition, in featureargs.SheetMetalHem) (*feature.SheetMetalHemDefinition, error) {
	hemType, ok := feature.ParseHemType(in.Type)
	if !ok {
		return nil, fmt.Errorf("sheetMetalHem: unknown type %q (want closed or open)", in.Type)
	}
	length, err := lengthClosure(part, in.Length, "sheetMetalHem: length")
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalHemDefinition{EdgeKey: []byte(in.Edge), Length: length, Type: hemType, Flip: in.Flip}
	if in.Gap != "" {
		gap, err := lengthClosure(part, in.Gap, "sheetMetalHem: gap")
		if err != nil {
			return nil, err
		}
		def.Gap = gap
	}
	return def, nil
}
