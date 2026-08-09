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
    "width": {"type": "object", "description": "Bound the wall to PART of the edge (Inventor's flange width extents): a bracket tab on a long edge, or a wall that stops short of the corners. Absent = the whole edge.", "properties": {"type": {"type": "string", "enum": ["edge", "centered", "offsets", "offsetWidth"], "default": "edge", "description": "centered takes width; offsets takes offset (from the edge start) and offset2 (from its end); offsetWidth takes offset and width."}, "width": {"type": "string", "description": "Wall length, e.g. \"20 mm\"."}, "offset": {"type": "string", "description": "Distance from the edge's start."}, "offset2": {"type": "string", "description": "Distance from the edge's end (offsets type)."}}},
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
	if err := bindFlangeBend(part, def, in); err != nil {
		return nil, err
	}
	if err := bindFlangePlacement(part, def, in); err != nil {
		return nil, err
	}
	width, err := flangeWidthExtent(part, in.Width)
	if err != nil {
		return nil, err
	}
	def.Width = width
	return def, nil
}

// flangeWidthExtent resolves a width extent's type and distances (#1958). Each distance is a
// driven expression, so a parameter change moves the tab with the rest of the part.
func flangeWidthExtent(part *compdef.PartComponentDefinition, in *featureargs.FlangeWidthExtent) (feature.FlangeWidth, error) {
	if in == nil {
		return feature.FlangeWidth{}, nil
	}
	kind, ok := feature.ParseWidthExtent(strings.TrimSpace(in.Type))
	if !ok {
		return feature.FlangeWidth{}, fmt.Errorf("sheetMetalFlange: unknown width extent %q (want "+
			"edge, centered, offsets or offsetWidth)", in.Type)
	}
	w := feature.FlangeWidth{Type: kind}
	for _, d := range []struct {
		expr, what string
		dst        *func() float64
	}{{in.Width, "width", &w.Width}, {in.Offset, "offset", &w.Offset}, {in.Offset2, "offset2", &w.Offset2}} {
		if d.expr == "" {
			continue
		}
		closure, err := lengthClosure(part, d.expr, "sheetMetalFlange: width "+d.what)
		if err != nil {
			return feature.FlangeWidth{}, err
		}
		*d.dst = closure
	}
	return w, nil
}

// bindFlangeBend attaches the optional bend overrides; omitted ⇒ nil, so the feature applies its
// own defaults (90° over the rule's bend radius).
func bindFlangeBend(part *compdef.PartComponentDefinition, def *feature.SheetMetalFlangeDefinition,
	in featureargs.SheetMetalFlange) error {
	if in.Angle != "" {
		angle, err := angleClosure(part, in.Angle, "sheetMetalFlange: angle")
		if err != nil {
			return err
		}
		def.Angle = angle
	}
	if in.Radius == "" {
		return nil
	}
	radius, err := lengthClosure(part, in.Radius, "sheetMetalFlange: radius")
	if err != nil {
		return err
	}
	def.Radius = radius
	return nil
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
