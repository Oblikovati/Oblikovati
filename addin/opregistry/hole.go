// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The hole operation — a subtractive drilled hole on a picked face, referenced by key
// (get_reference_keys). With a depth it is a blind drilled hole; without one it drills through
// all. Counterbore/countersink/tapped variants follow the same shape (HoleFeatures.Add*).

const holeSchema = `{
  "type": "object",
  "properties": {
    "faceRef": {"type": "string", "description": "Reference key of the planar face to drill into (from get_reference_keys)."},
    "type": {"type": "string", "enum": ["drilled", "counterbore", "countersink", "tapped"], "default": "drilled", "description": "Hole style."},
    "diameter": {"type": "string", "description": "Hole diameter with units, e.g. \"5 mm\"."},
    "depth": {"type": "string", "description": "Blind hole depth, e.g. \"8 mm\". Omit (drilled only) for a through-all hole."},
    "counterDiameter": {"type": "string", "description": "Counterbore diameter (type=counterbore)."},
    "counterDepth": {"type": "string", "description": "Counterbore depth (type=counterbore)."},
    "sinkDiameter": {"type": "string", "description": "Countersink top diameter (type=countersink)."},
    "includedAngle": {"type": "string", "description": "Countersink included angle, e.g. \"90 deg\" (type=countersink)."},
    "designation": {"type": "string", "description": "Thread designation, e.g. \"M5x0.8\" (type=tapped)."},
    "center": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Explicit drill point [x,y,z] in model-space cm (projected onto the picked face). Omit to drill at the face centroid. Needed to place more than one hole on a face."},
    "centerExpr": {"type": "array", "items": {"type": "string"}, "minItems": 3, "maxItems": 3, "description": "Parameter-expression form of center: [x,y,z] each an expression with units (e.g. \"L/2\"). Overrides center when present."},
    "placementFaceGeom": {"type": "object", "description": "Select the placement face by GEOMETRY instead of faceRef, so the binding survives recompute (for an external author that cannot mint a stable key). Give either this or faceRef.", "properties": {"centroid": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Face centroid [x,y,z] cm."}, "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Outward unit normal [x,y,z]."}}, "required": ["centroid", "normal"]}
  },
  "required": ["diameter"]
}`

func holeDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindHole, Summary: "Drill a hole into a picked face: drilled (blind/through), counterbore, countersink, or tapped.", Schema: json.RawMessage(holeSchema), Apply: applyHole}
}

func applyHole(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Hole](s, raw)
	if err != nil {
		return nil, err
	}
	if in.FaceRef == "" && in.PlacementFaceGeom == nil {
		return nil, errors.New("hole: needs faceRef or placementFaceGeom")
	}
	dia, err := lengthClosure(part, in.Diameter, "hole: diameter")
	if err != nil {
		return nil, err
	}
	pf, err := buildHole(part, in, dia)
	if err != nil {
		return nil, err
	}
	if err := applyHoleCenter(part, pf, in); err != nil {
		return nil, err
	}
	if err := applyHolePlacementGeom(pf, in); err != nil {
		return nil, err
	}
	return recomputeResult(part, pf)
}

// applyHolePlacementGeom binds the hole's placement face by GEOMETRY (centroid + normal) when
// PlacementFaceGeom is given, so an external author's face selection survives recompute — the
// hole re-resolves it against the running body each time (see resolveHoleFace / GeomFace).
func applyHolePlacementGeom(pf *feature.PartFeature, in featureargs.Hole) error {
	if in.PlacementFaceGeom == nil {
		return nil
	}
	ref, err := geomFaceRef(*in.PlacementFaceGeom)
	if err != nil {
		return err
	}
	pf.Definition().(*feature.HoleFeature).Definition().GeomFace = &ref
	return nil
}

// applyHoleCenter sets the hole's explicit drill center from the args (centerExpr wins over the
// literal center); absent both, the definition keeps its face-centroid default (Center stays nil).
func applyHoleCenter(part *compdef.PartComponentDefinition, pf *feature.PartFeature, in featureargs.Hole) error {
	center, err := holeCenterPoint(part, in)
	if err != nil || center == nil {
		return err
	}
	pf.Definition().(*feature.HoleFeature).Definition().Center = center
	return nil
}

// holeCenterPoint resolves the explicit drill center: centerExpr (three unit-bearing expressions,
// resolved to model cm) when present, else the literal center [x,y,z]; nil when neither is given.
func holeCenterPoint(part *compdef.PartComponentDefinition, in featureargs.Hole) (*math.Point3, error) {
	if len(in.CenterExpr) > 0 {
		return centerFromExprs(part, in.CenterExpr)
	}
	if len(in.Center) == 0 {
		return nil, nil
	}
	p, err := point3(in.Center, "hole: center")
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// centerFromExprs resolves the three coordinate expressions of centerExpr into a model-space point.
func centerFromExprs(part *compdef.PartComponentDefinition, exprs []string) (*math.Point3, error) {
	if len(exprs) != 3 {
		return nil, fmt.Errorf("hole: centerExpr needs 3 expressions [x,y,z], got %d", len(exprs))
	}
	var c [3]float64
	for i, e := range exprs {
		v, err := lengthClosure(part, e, "hole: centerExpr")
		if err != nil {
			return nil, err
		}
		c[i] = v()
	}
	p := math.P3(c[0], c[1], c[2])
	return &p, nil
}

// buildHole dispatches on the hole type, resolving the extra dimensions each variant needs.
//
//nolint:funlen // one-case-per-hole-type dispatch switch (drilled/counterbore/countersink/tapped); length is the dispatch, like the serialize codecs.
func buildHole(part *compdef.PartComponentDefinition, in featureargs.Hole, dia func() float64) (*feature.PartFeature, error) {
	holes := feature.NewHoleFeatures(part.Features())
	key := []byte(in.FaceRef)
	switch in.Type {
	case "", "drilled":
		if in.Depth == "" {
			return holes.AddDrilledThrough(key, dia), nil
		}
		depth, err := lengthClosure(part, in.Depth, "hole: depth")
		if err != nil {
			return nil, err
		}
		return holes.AddDrilled(key, dia, depth), nil
	case "counterbore":
		depth, cdia, cdepth, err := threeLengths(part, in.Depth, in.CounterDiameter, in.CounterDepth, "hole counterbore")
		if err != nil {
			return nil, err
		}
		return holes.AddCounterbore(key, dia, depth, cdia, cdepth), nil
	case "countersink":
		depth, sdia, err := twoLengths(part, in.Depth, in.SinkDiameter, "hole countersink")
		if err != nil {
			return nil, err
		}
		angle, err := angleClosure(part, in.IncludedAngle, "hole: includedAngle")
		if err != nil {
			return nil, err
		}
		return holes.AddCountersink(key, dia, depth, sdia, angle), nil
	case "tapped":
		depth, err := lengthClosure(part, in.Depth, "hole: depth")
		if err != nil {
			return nil, err
		}
		if in.Designation == "" {
			return nil, errors.New("hole: tapped needs a designation, e.g. \"M5x0.8\"")
		}
		return holes.AddTapped(key, dia, depth, in.Designation), nil
	default:
		return nil, fmt.Errorf("hole: unknown type %q (want drilled|counterbore|countersink|tapped)", in.Type)
	}
}

func twoLengths(part *compdef.PartComponentDefinition, a, b, ctx string) (func() float64, func() float64, error) {
	av, err := lengthClosure(part, a, ctx+": depth")
	if err != nil {
		return nil, nil, err
	}
	bv, err := lengthClosure(part, b, ctx+": diameter")
	if err != nil {
		return nil, nil, err
	}
	return av, bv, nil
}

func threeLengths(part *compdef.PartComponentDefinition, a, b, c, ctx string) (func() float64, func() float64, func() float64, error) {
	av, bv, err := twoLengths(part, a, b, ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	cv, err := lengthClosure(part, c, ctx+": counterDepth")
	if err != nil {
		return nil, nil, nil, err
	}
	return av, bv, cv, nil
}
