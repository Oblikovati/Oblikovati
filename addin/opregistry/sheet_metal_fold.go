// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

const sheetMetalFoldSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Index of the sketch holding the fold line (see model.tree)."},
    "lineIndex": {"type": "integer", "minimum": 0, "default": 0, "description": "Which line of the sketch is the fold axis. It must lie on the face."},
    "angle": {"type": "string", "description": "Bend angle, e.g. \"90 deg\" (default)."},
    "radius": {"type": "string", "description": "Inside bend radius (default: the rule's bend radius)."},
    "location": {"type": "string", "enum": ["centerline", "start", "end"], "default": "centerline", "description": "Where the fold line sits relative to the bend."},
    "flip": {"type": "boolean", "default": false, "description": "Fold toward the opposite side of the sketch plane."}
  },
  "required": ["sketchIndex"]
}`

// sheetMetalFoldDescriptor is the self-describing "sheetMetalFold" operation: fold a face
// along a sketch line at the active rule's bend radius, positioning the bend by location.
func sheetMetalFoldDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalFold,
		Summary: "Fold a sheet-metal face along a sketch line, placing the bend at the start, centerline, or end of the line.",
		Schema:  json.RawMessage(sheetMetalFoldSchema),
		Apply:   applySheetMetalFold,
	}
}

func applySheetMetalFold(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeSheetMetalArgs[featureargs.SheetMetalFold](s, raw, "sheetMetalFold")
	if err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	def, err := foldDef(part, sk, in)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, feature.NewSheetMetalFoldFeatures(part.Features()).Add(def))
}

// foldDef resolves the fold args into a definition: the sketch + line, the bend location, and
// the optional angle/radius closures.
func foldDef(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in featureargs.SheetMetalFold) (*feature.SheetMetalFoldDefinition, error) {
	loc, ok := feature.ParseBendLocation(in.Location)
	if !ok {
		return nil, fmt.Errorf("sheetMetalFold: unknown location %q (want centerline, start or end)", in.Location)
	}
	def := &feature.SheetMetalFoldDefinition{Sketch: sk, LineIndex: in.LineIndex, Location: loc, Flip: in.Flip}
	if in.Angle != "" {
		angle, err := angleClosure(part, in.Angle, "sheetMetalFold: angle")
		if err != nil {
			return nil, err
		}
		def.Angle = angle
	}
	if in.Radius != "" {
		radius, err := lengthClosure(part, in.Radius, "sheetMetalFold: radius")
		if err != nil {
			return nil, err
		}
		def.Radius = radius
	}
	return def, nil
}
