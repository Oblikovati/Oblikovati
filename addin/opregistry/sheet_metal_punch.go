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

const sheetMetalPunchSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Index of the sketch whose closed profiles are punched (see model.tree)."},
    "depth": {"type": "string", "description": "Punch depth (default: through all the material)."},
    "angle": {"type": "string", "description": "Rotate the punched profiles about their centroid, e.g. \"30 deg\"."},
    "acrossBends": {"type": "boolean", "default": false, "description": "Let the punch span a bent region."},
    "unfoldInFlat": {"type": "boolean", "default": false, "description": "Develop the punch into the flat pattern."},
    "representationType": {"type": "string", "enum": ["default", "formedFeature", "sketch2D", "centermark", "sketch2DAndCentermark"], "default": "default", "description": "The punch's flat/drawing appearance."},
    "toolId": {"type": "string", "description": "Identifier of the die tool (metadata for the flat/drawing)."}
  },
  "required": ["sketchIndex"]
}`

// sheetMetalPunchDescriptor is the self-describing "sheetMetalPunch" operation: stamp every
// closed profile of a sketch through the sheet in one die-pattern punch.
func sheetMetalPunchDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalPunch,
		Summary: "Punch every closed profile of a sketch through a sheet-metal part — a die pattern (vents, louvers, perforations) in one operation, with a rotation angle and die metadata.",
		Schema:  json.RawMessage(sheetMetalPunchSchema),
		Apply:   applySheetMetalPunch,
	}
}

func applySheetMetalPunch(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeSheetMetalArgs[featureargs.SheetMetalPunch](s, raw, "sheetMetalPunch")
	if err != nil {
		return nil, err
	}
	def, err := punchDefinition(part, &in)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, feature.NewSheetMetalPunchFeatures(part.Features()).Add(def))
}

// punchDefinition resolves the wire args into a punch recipe: the sketch, the optional depth and
// rotation, and the die metadata — erroring on an unknown representation type or a bad length/angle.
func punchDefinition(part *compdef.PartComponentDefinition, in *featureargs.SheetMetalPunch) (*feature.SheetMetalPunchDefinition, error) {
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	rep, ok := types.ParsePunchRepresentationType(in.RepresentationType)
	if !ok {
		return nil, fmt.Errorf("sheetMetalPunch: unknown representationType %q", in.RepresentationType)
	}
	def := &feature.SheetMetalPunchDefinition{
		Sketch: sk, AcrossBends: in.AcrossBends, UnfoldInFlat: in.UnfoldInFlat, ToolID: in.ToolID, Representation: rep,
	}
	if in.Depth != "" {
		if def.Depth, err = lengthClosure(part, in.Depth, "sheetMetalPunch: depth"); err != nil {
			return nil, err
		}
	}
	if in.Angle != "" {
		if def.Angle, err = angleClosure(part, in.Angle, "sheetMetalPunch: angle"); err != nil {
			return nil, err
		}
	}
	return def, nil
}
