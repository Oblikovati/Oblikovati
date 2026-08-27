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

const sheetMetalLipSchema = `{
  "type": "object",
  "properties": {
    "edge": {"type": "string", "description": "Reference key of the straight sheet edge to build the lip on."},
    "height": {"type": "string", "description": "Flange wall height before the return, e.g. \"10 mm\"."},
    "returnLength": {"type": "string", "default": "3 mm", "description": "Return wall length after the 180° curl."},
    "angle": {"type": "string", "description": "Flange bend angle, e.g. \"90 deg\" (default)."},
    "radius": {"type": "string", "description": "Inside bend radius (default: the rule's bend radius)."},
    "flip": {"type": "boolean", "default": false, "description": "Fold toward the opposite side of the sheet."}
  },
  "required": ["edge", "height"]
}`

// sheetMetalLipDescriptor is the self-describing "sheetMetalLip" operation: fold a stiffening
// lip (a short flange curled 180° back on itself) onto a sheet edge.
func sheetMetalLipDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalLip,
		Summary: "Fold a stiffening lip (a short flange curled 180° back on itself) onto a straight sheet-metal edge.",
		Schema:  json.RawMessage(sheetMetalLipSchema),
		Apply:   applySheetMetalLip,
	}
}

// defaultLipReturn is the return-wall length used when none is given.
const defaultLipReturn = "3 mm"

func applySheetMetalLip(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeSheetMetalArgs[featureargs.SheetMetalLip](s, raw, "sheetMetalLip")
	if err != nil {
		return nil, err
	}
	def, err := lipDef(part, in)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, feature.NewSheetMetalLipFeatures(part.Features()).Add(def))
}

// lipDef resolves the lip args into a definition: the edge + parameter-backed height/return and
// the optional angle/radius closures (omitted ⇒ nil, so the feature uses its defaults).
func lipDef(part *compdef.PartComponentDefinition, in featureargs.SheetMetalLip) (*feature.SheetMetalLipDefinition, error) {
	if in.Edge == "" {
		return nil, fmt.Errorf("sheetMetalLip: edge is required")
	}
	height, err := lengthClosure(part, in.Height, "sheetMetalLip: height")
	if err != nil {
		return nil, err
	}
	returnExpr := in.ReturnLength
	if returnExpr == "" {
		returnExpr = defaultLipReturn
	}
	returnLen, err := lengthClosure(part, returnExpr, "sheetMetalLip: returnLength")
	if err != nil {
		return nil, err
	}
	angle, radius, err := optionalBendDims(part, in.Angle, in.Radius, "sheetMetalLip")
	if err != nil {
		return nil, err
	}
	return &feature.SheetMetalLipDefinition{
		EdgeKey: []byte(in.Edge), Height: height, ReturnLength: returnLen,
		Angle: angle, Radius: radius, Flip: in.Flip,
	}, nil
}
