// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"
	"strings"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

const sheetMetalFlangeSchema = `{
  "type": "object",
  "properties": {
    "edge": {"type": "string", "description": "Reference key of the straight sheet edge to flange from (from get_reference_keys)."},
    "height": {"type": "string", "description": "Flange wall length with units, e.g. \"15 mm\"."},
    "angle": {"type": "string", "description": "Bend angle, e.g. \"90 deg\" (default). The wall folds this far from the parent face."},
    "radius": {"type": "string", "description": "Inside bend radius (default: the rule's bend radius)."},
    "flip": {"type": "boolean", "default": false, "description": "Fold toward the opposite side of the sheet."},
    "bendPosition": {"type": "string", "enum": ["adjacentFace", "outsideBaseFace", "insideBendFace", "outerEdgeOffset", "innerEdgeOffset"], "default": "adjacentFace", "description": "How far back from the picked edge the bend sits (Inventor BendPositionEnum). adjacentFace starts the bend AT the edge; outsideBaseFace and insideBendFace set it back until the wall's outer or inner face reaches the edge, so the wall does not overhang; the two ...EdgeOffset positions add positionOffset to those."},
    "positionOffset": {"type": "string", "description": "Explicit distance for the outerEdgeOffset / innerEdgeOffset positions, e.g. \"2 mm\"."},
    "heightDatum": {"type": "string", "enum": ["tangent", "outer", "inner", "outerOrtho", "innerOrtho"], "default": "tangent", "description": "What height is measured FROM (Inventor HeightDatumTypeEnum). tangent measures the wall from where the bend ends; outer/inner measure from the sharp corner the outer/inner faces would make, the way a drawing dimensions it; the ortho pair measures those corners perpendicular to the base face."}
  },
  "required": ["edge", "height"]
}`

// sheetMetalFlangeDescriptor is the self-describing "sheetMetalFlange" operation: fold a wall
// onto a sheet edge over a bend at the active rule's gauge.
func sheetMetalFlangeDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalFlange,
		Summary: "Fold a wall (flange) onto a straight sheet-metal edge over a cylindrical bend, at the active rule's thickness and bend radius.",
		Schema:  json.RawMessage(sheetMetalFlangeSchema),
		Apply:   applySheetMetalFlange,
	}
}

func applySheetMetalFlange(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalFlange")
	if err != nil {
		return nil, err
	}
	var in featureargs.SheetMetalFlange
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalFlange: invalid args: %w", err)
	}
	if in.Edge == "" {
		return nil, fmt.Errorf("sheetMetalFlange: edge is required")
	}
	def, err := flangeDef(part, in)
	if err != nil {
		return nil, err
	}
	pf := feature.NewSheetMetalFlangeFeatures(part.Features()).Add(def)
	return recomputeResult(part, pf)
}

// flangeDef resolves the flange args into a definition: the edge key, the height closure, and
// the optional angle/radius closures (omitted ⇒ nil, so the feature uses its defaults).
func flangeDef(part *compdef.PartComponentDefinition, in featureargs.SheetMetalFlange) (*feature.SheetMetalFlangeDefinition, error) {
	height, err := lengthClosure(part, in.Height, "sheetMetalFlange: height")
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalFlangeDefinition{EdgeKey: []byte(in.Edge), Height: height, Flip: in.Flip}
	if in.Angle != "" {
		angle, err := angleClosure(part, in.Angle, "sheetMetalFlange: angle")
		if err != nil {
			return nil, err
		}
		def.Angle = angle
	}
	if in.Radius != "" {
		radius, err := lengthClosure(part, in.Radius, "sheetMetalFlange: radius")
		if err != nil {
			return nil, err
		}
		def.Radius = radius
	}
	if err := bindFlangePlacement(part, def, in); err != nil {
		return nil, err
	}
	return def, nil
}

// bindFlangePlacement resolves where the wall lands (#1957): the bend position and its offset, and
// the datum the height is measured from.
func bindFlangePlacement(part *compdef.PartComponentDefinition, def *feature.SheetMetalFlangeDefinition,
	in featureargs.SheetMetalFlange) error {
	position, ok := feature.ParseBendPosition(strings.TrimSpace(in.BendPosition))
	if !ok {
		return fmt.Errorf("sheetMetalFlange: unknown bendPosition %q (want adjacentFace, "+
			"outsideBaseFace, insideBendFace, outerEdgeOffset or innerEdgeOffset)", in.BendPosition)
	}
	datum, ok := feature.ParseHeightDatum(strings.TrimSpace(in.HeightDatum))
	if !ok {
		return fmt.Errorf("sheetMetalFlange: unknown heightDatum %q (want tangent, outer, inner, "+
			"outerOrtho or innerOrtho)", in.HeightDatum)
	}
	def.Position, def.HeightDatum = position, datum
	if in.PositionOffset == "" {
		return nil
	}
	offset, err := lengthClosure(part, in.PositionOffset, "sheetMetalFlange: positionOffset")
	if err != nil {
		return err
	}
	def.PositionOffset = offset
	return nil
}
