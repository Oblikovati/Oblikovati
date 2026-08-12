// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

const sheetMetalCornerSeamSchema = `{
  "type": "object",
  "properties": {
    "edges": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the corner edges where flanges meet (from get_reference_keys)."},
    "gap": {"type": "string", "description": "Seam gap, e.g. \"0.2 mm\" — the relief left between the two corner walls."},
    "type": {"type": "string", "enum": ["gap", "overlap", "reverseOverlap", "noOverlap"], "default": "gap", "description": "The seam finish. Only 'gap' cuts a solid today; the lap/butt styles are recorded and modelled in a follow-up."},
    "overlap": {"type": "number", "description": "How far one wall laps over the other, as a percentage 0-100 (overlap / reverseOverlap only)."},
    "reliefShape": {"type": "string", "enum": ["trimToBend", "round", "square", "tear", "fullRound", "roundWithRadius", "intersection"], "description": "Relief shape cut at the seam root; absent leaves no seam-root relief."},
    "reliefSize": {"type": "string", "description": "Size of the seam-root relief, e.g. \"1 mm\"."},
    "definitionType": {"type": "string", "enum": ["maxDistance", "faceEdgeDistance"], "default": "maxDistance", "description": "How the gap is measured — the two agree on a square miter."}
  },
  "required": ["edges", "gap"]
}`

// sheetMetalCornerSeamDescriptor is the self-describing "sheetMetalCornerSeam" operation:
// finish the corner where two flanges meet (gap seam; lap/butt styles recorded, #2085).
func sheetMetalCornerSeamDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalCornerSeam,
		Summary: "Finish the corner where two sheet-metal flanges meet: a gap seam (a square notch of the given gap along the shared corner edges), or a recorded overlap/reverse-overlap/no-overlap seam.",
		Schema:  json.RawMessage(sheetMetalCornerSeamSchema),
		Apply:   applySheetMetalCornerSeam,
	}
}

func applySheetMetalCornerSeam(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalCornerSeam")
	if err != nil {
		return nil, err
	}
	var in featureargs.SheetMetalCornerSeam
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalCornerSeam: invalid args: %w", err)
	}
	if len(in.Edges) == 0 {
		return nil, fmt.Errorf("sheetMetalCornerSeam: edges is empty")
	}
	def, err := cornerSeamDefinition(part, &in)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, feature.NewSheetMetalCornerSeamFeatures(part.Features()).Add(def))
}

// cornerSeamDefinition resolves the wire args into a corner-seam recipe, erroring on any unknown
// seam type, relief shape or definition type with the offending value and the allowed set.
func cornerSeamDefinition(part *compdef.PartComponentDefinition, in *featureargs.SheetMetalCornerSeam) (*feature.SheetMetalCornerSeamDefinition, error) {
	seam, ok := feature.ParseSeamType(in.Type)
	if !ok {
		return nil, fmt.Errorf("sheetMetalCornerSeam: unknown type %q (want gap, overlap, reverseOverlap or noOverlap)", in.Type)
	}
	defType, ok := types.ParseCornerSeamDefinitionType(in.DefinitionType)
	if !ok {
		return nil, fmt.Errorf("sheetMetalCornerSeam: unknown definitionType %q (want maxDistance or faceEdgeDistance)", in.DefinitionType)
	}
	gap, err := lengthClosure(part, in.Gap, "sheetMetalCornerSeam: gap")
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalCornerSeamDefinition{
		EdgeKeys: refKeys(in.Edges), Gap: gap, Type: seam, Overlap: in.Overlap, DefinitionType: defType,
	}
	return def, applyCornerSeamRelief(part, in, def)
}

// applyCornerSeamRelief resolves the optional seam-root relief (shape + size) onto the recipe,
// erroring on an unknown shape or an unparseable size. Absent relief leaves the recipe untouched.
func applyCornerSeamRelief(part *compdef.PartComponentDefinition, in *featureargs.SheetMetalCornerSeam,
	def *feature.SheetMetalCornerSeamDefinition) error {
	if in.ReliefShape != "" {
		shape, ok := types.ParseCornerReliefShape(in.ReliefShape)
		if !ok {
			return fmt.Errorf("sheetMetalCornerSeam: unknown reliefShape %q", in.ReliefShape)
		}
		def.ReliefShape = shape
	}
	if in.ReliefSize != "" {
		size, err := lengthClosure(part, in.ReliefSize, "sheetMetalCornerSeam: reliefSize")
		if err != nil {
			return err
		}
		def.ReliefSize = size
	}
	return nil
}
