// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"

	"strings"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

const sheetMetalContourFlangeSchema = `{
  "type": "object",
  "properties": {
    "edge": {"type": "string", "description": "Reference key of the straight sheet edge to sweep the profile along (from get_reference_keys)."},
    "profileSketch": {"type": "integer", "minimum": 0, "description": "Index of the sketch holding the open profile (the flange cross-section, a chain of lines starting at the edge)."},
    "operation": {"type": "string", "enum": ["join", "new"], "default": "join", "description": "join unions the wall onto the running sheet; new starts a body of its own."},
    "radius": {"type": "string", "description": "Bend radius the profile's corners are rounded to, e.g. \"3 mm\"; absent uses the rule's BendRadius. A contour flange's corners are bends."},
    "width": {"type": "object", "description": "Bound the swept wall to PART of the edge (#1958); absent = the whole edge.", "properties": {"type": {"type": "string", "enum": ["edge", "centered", "offsets", "offsetWidth"]}, "width": {"type": "string"}, "offset": {"type": "string"}, "offset2": {"type": "string"}}},
    "flip": {"type": "boolean", "default": false, "description": "Sweep toward the opposite side of the sheet."}
  },
  "required": ["edge", "profileSketch"]
}`

// contourFlangeDef assembles the swept wall's recipe: how it joins the model, and the radius its
// corners bend at (#1961).
func contourFlangeDef(part *compdef.PartComponentDefinition, in featureargs.SheetMetalContourFlange,
	profile *sketch.Sketch, width feature.FlangeWidth) (*feature.SheetMetalContourFlangeDefinition, error) {
	op, err := contourFlangeOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	def := &feature.SheetMetalContourFlangeDefinition{EdgeKey: []byte(in.Edge), Profile: profile,
		Flip: in.Flip, Width: width, Operation: op}
	if in.Radius == "" {
		return def, nil
	}
	if def.Radius, err = lengthClosure(part, in.Radius, "sheetMetalContourFlange: radius"); err != nil {
		return nil, err
	}
	return def, nil
}

// contourFlangeOperation maps the wall's operation; only join and new are meaningful, since a
// swept wall has nothing to cut or intersect with.
func contourFlangeOperation(spelling string) (ops.PartFeatureOperation, error) {
	switch strings.ToLower(strings.TrimSpace(spelling)) {
	case "", "join":
		return ops.Join, nil
	case "new", "newbody":
		return ops.NewBody, nil
	default:
		return ops.Join, fmt.Errorf("sheetMetalContourFlange: unknown operation %q (want join or new)", spelling)
	}
}

// sheetMetalContourFlangeDescriptor is the self-describing "sheetMetalContourFlange" operation:
// sweep an open profile along a sheet edge into a contour flange.
func sheetMetalContourFlangeDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalContourFlange,
		Summary: "Sweep an open sketch profile (the wall cross-section) along a sheet-metal edge into a contour flange, at the rule's thickness.",
		Schema:  json.RawMessage(sheetMetalContourFlangeSchema),
		Apply:   applySheetMetalContourFlange,
	}
}

func applySheetMetalContourFlange(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeSheetMetalArgs[featureargs.SheetMetalContourFlange](s, raw, "sheetMetalContourFlange")
	if err != nil {
		return nil, err
	}
	if in.Edge == "" {
		return nil, fmt.Errorf("sheetMetalContourFlange: edge is required")
	}
	profile, err := sketchAt(part, in.ProfileSketch)
	if err != nil {
		return nil, err
	}
	width, err := flangeWidthExtent(part, in.Width)
	if err != nil {
		return nil, err
	}
	def, err := contourFlangeDef(part, in, profile, width)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, feature.NewSheetMetalContourFlangeFeatures(part.Features()).Add(def))
}
