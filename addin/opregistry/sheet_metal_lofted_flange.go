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

const sheetMetalLoftedFlangeSchema = `{
  "type": "object",
  "properties": {
    "profileA": {"type": "integer", "minimum": 0, "description": "Index of the sketch holding the first open profile."},
    "profileB": {"type": "integer", "minimum": 0, "description": "Index of the sketch holding the second open profile (same vertex count as A)."},
    "operation": {"type": "string", "enum": ["new", "join"], "default": "new", "description": "new for a standalone transition wall, join to merge it onto the running part."},
    "outputType": {"type": "string", "enum": ["dieFormed", "pressBrakeChordTolerance", "pressBrakeFacetAngle", "pressBrakeFacetDistance"], "default": "dieFormed", "description": "How the transition is calculated: die-formed (smooth) or a press-brake faceted mode."},
    "facetTolerance": {"type": "string", "description": "Facet bound for a press-brake output — a length (\"0.5 mm\") for chord/distance modes, an angle (\"5 deg\") for facet-angle."},
    "converge": {"type": "boolean", "default": false, "description": "Converge the transition's corners to a point (recorded; geometry pending #2086)."},
    "radius": {"type": "string", "description": "End-bend radius (recorded; geometry pending #2086)."}
  },
  "required": ["profileA", "profileB"]
}`

// sheetMetalLoftedFlangeDescriptor is the self-describing "sheetMetalLoftedFlange" operation:
// loft a constant-thickness wall between two open profiles, die-formed or press-brake faceted.
func sheetMetalLoftedFlangeDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalLoftedFlange,
		Summary: "Loft a constant-thickness sheet-metal wall between two open profiles (a transition piece) at the rule's thickness — die-formed (smooth) or press-brake faceted.",
		Schema:  json.RawMessage(sheetMetalLoftedFlangeSchema),
		Apply:   applySheetMetalLoftedFlange,
	}
}

func applySheetMetalLoftedFlange(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalLoftedFlange")
	if err != nil {
		return nil, err
	}
	var in featureargs.SheetMetalLoftedFlange
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalLoftedFlange: invalid args: %w", err)
	}
	def, err := loftedFlangeDefinition(part, &in)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, feature.NewSheetMetalLoftedFlangeFeatures(part.Features()).Add(def))
}

// loftedFlangeDefinition resolves the wire args into a lofted-flange recipe, erroring on an unknown
// output type or an unparseable facet tolerance / radius.
func loftedFlangeDefinition(part *compdef.PartComponentDefinition, in *featureargs.SheetMetalLoftedFlange) (*feature.SheetMetalLoftedFlangeDefinition, error) {
	profileA, err := sketchAt(part, in.ProfileA)
	if err != nil {
		return nil, err
	}
	profileB, err := sketchAt(part, in.ProfileB)
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	output, ok := types.ParseLoftedFlangeOutputType(in.OutputType)
	if !ok {
		return nil, fmt.Errorf("sheetMetalLoftedFlange: unknown outputType %q (want dieFormed or a pressBrake… mode)", in.OutputType)
	}
	def := &feature.SheetMetalLoftedFlangeDefinition{ProfileA: profileA, ProfileB: profileB, Operation: op, Output: output, Converge: in.Converge}
	return fillLoftedFlangeTolerance(part, in, output, def)
}

// fillLoftedFlangeTolerance parses the facet tolerance (a length for chord/distance modes, an angle
// for facet-angle) and the end-bend radius onto the recipe.
func fillLoftedFlangeTolerance(part *compdef.PartComponentDefinition, in *featureargs.SheetMetalLoftedFlange,
	output types.LoftedFlangeOutputType, def *feature.SheetMetalLoftedFlangeDefinition) (*feature.SheetMetalLoftedFlangeDefinition, error) {
	if in.FacetTolerance != "" && output.IsPressBrake() {
		tol, err := loftedFlangeToleranceValue(part, in.FacetTolerance, output)
		if err != nil {
			return nil, err
		}
		def.FacetTolerance = tol
	}
	if in.Radius != "" {
		radius, err := lengthClosure(part, in.Radius, "sheetMetalLoftedFlange: radius")
		if err != nil {
			return nil, err
		}
		def.Radius = radius
	}
	return def, nil
}

// loftedFlangeToleranceValue reads the facet tolerance in the unit its mode measures — an angle in
// radians for the facet-angle mode, a length otherwise.
func loftedFlangeToleranceValue(part *compdef.PartComponentDefinition, expr string, output types.LoftedFlangeOutputType) (float64, error) {
	if output == types.PressBrakeFacetAngleLoftedFlange {
		return angleValue(part, expr, "sheetMetalLoftedFlange: facetTolerance")
	}
	return lengthValue(part, expr, "sheetMetalLoftedFlange: facetTolerance")
}
