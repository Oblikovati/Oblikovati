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

const sheetMetalCornerSchema = `{
  "type": "object",
  "properties": {
    "edges": {"type": "array", "items": {"type": "string"}, "description": "Reference keys of the through-thickness corner edges to finish (from get_reference_keys). For a multi-radius round use edgeSets instead."},
    "treatment": {"type": "string", "enum": ["chamfer", "round"], "description": "chamfer cuts a flat across the corner; round rolls a fillet."},
    "size": {"type": "string", "description": "Chamfer first setback or round radius, e.g. \"3 mm\"."},
    "chamferType": {"type": "string", "enum": ["distance", "distanceAndAngle", "twoDistances"], "default": "distance", "description": "The chamfer's setback shape."},
    "distanceTwo": {"type": "string", "description": "The second setback for a twoDistances chamfer."},
    "angle": {"type": "string", "description": "The bevel angle for a distanceAndAngle chamfer, e.g. \"30 deg\"."},
    "faceKey": {"type": "string", "description": "The face the chamfer's first setback is measured on."},
    "edgeSets": {"type": "array", "items": {"type": "object", "properties": {"edges": {"type": "array", "items": {"type": "string"}}, "radius": {"type": "string"}}, "required": ["edges", "radius"]}, "description": "A multi-radius corner round: each set its own edges and radius, in one feature."}
  },
  "required": ["treatment"]
}`

// sheetMetalCornerDescriptor is the self-describing "sheetMetalCorner" operation: chamfer or
// round one or more sheet-metal corners.
func sheetMetalCornerDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSheetMetalCorner,
		Summary: "Chamfer or round one or more corners of a sheet-metal face: a single-size chamfer or round, a distance-and-angle / two-distance chamfer, or a multi-radius round.",
		Schema:  json.RawMessage(sheetMetalCornerSchema),
		Apply:   applySheetMetalCorner,
	}
}

func applySheetMetalCorner(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := activeSheetMetalPart(s, "sheetMetalCorner")
	if err != nil {
		return nil, err
	}
	var in featureargs.SheetMetalCorner
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("sheetMetalCorner: invalid args: %w", err)
	}
	treatment, ok := feature.ParseCornerTreatment(in.Treatment)
	if !ok {
		return nil, fmt.Errorf("sheetMetalCorner: unknown treatment %q (want chamfer or round)", in.Treatment)
	}
	def, err := cornerDefinition(part, &in, treatment)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, feature.NewSheetMetalCornerFeatures(part.Features()).Add(def))
}

// cornerDefinition resolves the wire args into a corner recipe: a multi-radius round when edgeSets
// is given, otherwise the single-size chamfer/round form with any chamfer variant.
func cornerDefinition(part *compdef.PartComponentDefinition, in *featureargs.SheetMetalCorner, treatment feature.CornerTreatment) (*feature.SheetMetalCornerDefinition, error) {
	def := &feature.SheetMetalCornerDefinition{Treatment: treatment, FaceKey: []byte(in.FaceKey)}
	if treatment == feature.CornerRound && len(in.EdgeSets) > 0 {
		sets, err := cornerRoundSets(part, in.EdgeSets)
		if err != nil {
			return nil, err
		}
		def.RoundSets = sets
		return def, nil
	}
	if len(in.Edges) == 0 {
		return nil, fmt.Errorf("sheetMetalCorner: edges is empty")
	}
	def.EdgeKeys = refKeys(in.Edges)
	size, err := lengthClosure(part, in.Size, "sheetMetalCorner: size")
	if err != nil {
		return nil, err
	}
	def.Size = size
	if treatment == feature.CornerChamfer {
		return def, fillCornerChamferVariant(part, in, def)
	}
	return def, nil
}

// fillCornerChamferVariant resolves the chamfer's setback shape onto the recipe — the second
// distance for two-distance, the angle for distance-and-angle — erroring on an unknown type.
func fillCornerChamferVariant(part *compdef.PartComponentDefinition, in *featureargs.SheetMetalCorner, def *feature.SheetMetalCornerDefinition) error {
	ct := types.ChamferDistance
	if in.ChamferType != "" {
		parsed, ok := types.ParseChamferType(in.ChamferType)
		if !ok {
			return fmt.Errorf("sheetMetalCorner: unknown chamferType %q (want distance, distanceAndAngle or twoDistances)", in.ChamferType)
		}
		ct = parsed
	}
	def.ChamferType = ct
	if in.DistanceTwo != "" {
		d2, err := lengthClosure(part, in.DistanceTwo, "sheetMetalCorner: distanceTwo")
		if err != nil {
			return err
		}
		def.Distance2 = d2
	}
	if in.Angle != "" {
		angle, err := angleClosure(part, in.Angle, "sheetMetalCorner: angle")
		if err != nil {
			return err
		}
		def.Angle = angle
	}
	return nil
}

// cornerRoundSets resolves the wire edge sets into round recipe sets, erroring on an empty set or
// an unparseable radius.
func cornerRoundSets(part *compdef.PartComponentDefinition, sets []featureargs.CornerRoundEdgeSet) ([]feature.CornerRoundSet, error) {
	out := make([]feature.CornerRoundSet, len(sets))
	for i, s := range sets {
		if len(s.Edges) == 0 {
			return nil, fmt.Errorf("sheetMetalCorner: edge set %d has no edges", i)
		}
		radius, err := lengthClosure(part, s.Radius, fmt.Sprintf("sheetMetalCorner: edge set %d radius", i))
		if err != nil {
			return nil, err
		}
		out[i] = feature.CornerRoundSet{EdgeKeys: refKeys(s.Edges), Radius: radius}
	}
	return out, nil
}
