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
    "length": {"type": "string", "description": "How far the folded-back leg runs, e.g. \"6 mm\" — single and double hems only."},
    "type": {"type": "string", "enum": ["single", "double", "rolled", "teardrop", "closed", "open"], "default": "single", "description": "Hem shape (Inventor HemTypeEnum). single folds back once, double folds back twice so the free edge stacks on the first leg; both take length+gap. rolled curls the edge, teardrop curls past a half-turn and closes back onto the sheet; both take radius+angle. \"closed\" and \"open\" are the older spellings and both mean single."},
    "gap": {"type": "string", "description": "Clear gap between the folded-back leg and the parent (fold radius = gap/2); absent folds tight at half the thickness. single/double only."},
    "radius": {"type": "string", "description": "Curl inside radius, e.g. \"3 mm\" — rolled and teardrop only."},
    "angle": {"type": "string", "description": "Curl sweep, e.g. \"270 deg\" — rolled and teardrop only. A teardrop must sweep more than 180 deg and less than 360 deg so its tail closes."},
    "flip": {"type": "boolean", "default": false, "description": "Fold toward the opposite side of the sheet."}
  },
  "required": ["edge"]
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
	part, in, err := decodeSheetMetalArgs[featureargs.SheetMetalHem](s, raw, "sheetMetalHem")
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("sheetMetalHem: unknown type %q (want single, double, rolled or teardrop)", in.Type)
	}
	def := &feature.SheetMetalHemDefinition{EdgeKey: []byte(in.Edge), Type: hemType, Flip: in.Flip}
	if err := bindHemDims(part, def, in); err != nil {
		return nil, err
	}
	return def, nil
}

// bindHemDims attaches the dimensions the hem actually carries. A dimension the type does not use
// is left nil rather than defaulted, so asking for a rolled hem with a length says what is missing
// (the radius) instead of building a fold at some invented radius.
func bindHemDims(part *compdef.PartComponentDefinition, def *feature.SheetMetalHemDefinition,
	in featureargs.SheetMetalHem) error {
	for _, d := range []struct {
		expr, what string
		angle      bool
		dst        *func() float64
	}{
		{in.Length, "length", false, &def.Length},
		{in.Gap, "gap", false, &def.Gap},
		{in.Radius, "radius", false, &def.Radius},
		{in.Angle, "angle", true, &def.Angle},
	} {
		if d.expr == "" {
			continue
		}
		closure, err := hemDimClosure(part, d.expr, d.angle, "sheetMetalHem: "+d.what)
		if err != nil {
			return err
		}
		*d.dst = closure
	}
	return nil
}

// hemDimClosure resolves one hem dimension — an angle expression for the curl sweep, a length for
// everything else.
func hemDimClosure(part *compdef.PartComponentDefinition, expr string, angle bool, what string) (func() float64, error) {
	if angle {
		return angleClosure(part, expr, what)
	}
	return lengthClosure(part, expr, what)
}
