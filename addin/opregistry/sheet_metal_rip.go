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

const sheetMetalRipSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Index of the sketch holding the rip line (pointToPoint form; see model.tree)."},
    "lineIndex": {"type": "integer", "minimum": 0, "default": 0, "description": "Which line of the sketch is the rip seam."},
    "gap": {"type": "string", "default": "0.1 mm", "description": "Width of the slit the rip opens, e.g. \"0.1 mm\"."},
    "type": {"type": "string", "enum": ["pointToPoint", "singlePoint", "faceExtents"], "default": "pointToPoint", "description": "How the rip line is defined on its face."},
    "faceKey": {"type": "string", "description": "Reference key of the face to rip (RipFace); required for singlePoint and faceExtents."},
    "point": {"type": "string", "description": "Reference key of a face vertex — the point for singlePoint, the first point for a two-vertex pointToPoint."},
    "pointTwo": {"type": "string", "description": "Reference key of the second face vertex for a two-vertex pointToPoint rip."},
    "gapSide": {"type": "string", "enum": ["positive", "negative", "symmetric"], "default": "symmetric", "description": "Which side of the rip line the gap sits."}
  }
}`

// sheetMetalRipDescriptor is the self-describing "sheetMetalRip" operation: cut a narrow slit so a
// closed or folded sheet can be developed flat — along a sketch line, or across a picked face.
func sheetMetalRipDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalRip,
		Summary: "Rip a sheet-metal part — a narrow through-thickness slit that opens a seam for unfolding, along a sketch line (pointToPoint) or a picked face (singlePoint / faceExtents).",
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
	def, err := ripDefinition(part, &in)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, feature.NewSheetMetalRipFeatures(part.Features()).Add(def))
}

// ripDefinition resolves the wire args into a rip recipe: a face-based rip when a face type or a
// faceKey is given, otherwise the sketch-line form. It errors on an unknown type or gap side, and
// on a face rip missing its face.
func ripDefinition(part *compdef.PartComponentDefinition, in *featureargs.SheetMetalRip) (*feature.SheetMetalRipDefinition, error) {
	ripType, ok := feature.ParseRipType(in.Type)
	if !ok {
		return nil, fmt.Errorf("sheetMetalRip: unknown type %q (want pointToPoint, singlePoint or faceExtents)", in.Type)
	}
	side, ok := feature.ParseRipGapSide(in.GapSide)
	if !ok {
		return nil, fmt.Errorf("sheetMetalRip: unknown gapSide %q (want positive, negative or symmetric)", in.GapSide)
	}
	gapExpr := in.Gap
	if gapExpr == "" {
		gapExpr = defaultRipGap
	}
	gap, err := lengthClosure(part, gapExpr, "sheetMetalRip: gap")
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalRipDefinition{Type: ripType, GapSide: side, Gap: gap}
	return fillRipInputs(part, in, def)
}

// fillRipInputs attaches either the face inputs or the sketch line to the recipe. A single-point
// or face-extents rip, or any rip naming a face, is face-based; everything else rips a sketch line.
func fillRipInputs(part *compdef.PartComponentDefinition, in *featureargs.SheetMetalRip,
	def *feature.SheetMetalRipDefinition) (*feature.SheetMetalRipDefinition, error) {
	faceBased := in.FaceKey != "" || def.Type == feature.SinglePointRip || def.Type == feature.FaceExtentsRip
	if faceBased {
		if in.FaceKey == "" {
			return nil, fmt.Errorf("sheetMetalRip: type %q needs a faceKey", in.Type)
		}
		def.FaceKey = []byte(in.FaceKey)
		def.PointKey = []byte(in.Point)
		def.PointTwoKey = []byte(in.PointTwo)
		return def, nil
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	def.Sketch = sk
	def.LineIndex = in.LineIndex
	return def, nil
}
