// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
    "type": {"type": "string", "enum": ["drilled", "counterbore", "countersink", "spotface", "tapped"], "default": "drilled", "description": "The hole's SEAT: the recess it opens into at the placement face. \"tapped\" is the older spelling of a drilled hole with tap=\"tapped\" — prefer setting \"tap\", which combines with any seat."},
    "tap": {"type": "string", "enum": ["none", "tapped", "taperTapped"], "default": "none", "description": "The hole's thread FUNCTION, independent of its seat — so a counterbored tapped hole is expressible. Needs a designation."},
    "threadClass": {"type": "string", "description": "Tap fit class, e.g. \"6H\" (metric) or \"2B\" (unified)."},
    "leftHanded": {"type": "boolean", "default": false, "description": "Reverse the tap's thread sense; the default is an ordinary right-hand thread."},
    "clearance": {"type": "object", "description": "Size the bore from a fastener table instead of diameter, keeping the fastener→hole link live. Inventor's HoleClearanceInfo.", "properties": {"standard": {"type": "string", "description": "Table the fastener is drawn from; \"ISO 273\" is carried."}, "fastener": {"type": "string", "description": "Thread designation the hole must pass, e.g. \"M6\"."}, "fit": {"type": "string", "enum": ["close", "medium", "free"], "default": "medium"}}, "required": ["fastener"]},
    "placement": {"type": "string", "enum": ["sketch", "linear", "concentric", "point"], "description": "The rule LOCATING the bores (Inventor's HolePlacementTypeEnum). Omit for the single bore at faceRef/center. \"sketch\" drills one hole per CENTRE POINT of placementSketchIndex."},
    "placementSketchIndex": {"type": "integer", "minimum": 0, "description": "Sketch whose centre points the \"sketch\" placement drills at."},
    "placementFlipped": {"type": "boolean", "default": false, "description": "Drill ALONG the sketch normal / work axis instead of into it."},
    "concentricRef": {"type": "string", "description": "Circular edge whose axis the \"concentric\" placement centres on."},
    "edge1Ref": {"type": "string", "description": "First reference edge of the \"linear\" placement."},
    "edge2Ref": {"type": "string", "description": "Second reference edge of the \"linear\" placement."},
    "offset1": {"type": "string", "description": "Distance from edge1Ref, measured INTO the placement face, e.g. \"10 mm\"."},
    "offset2": {"type": "string", "description": "Distance from edge2Ref, measured INTO the placement face."},
    "pointRef": {"type": "string", "description": "Work point the \"point\" placement drills at, e.g. \"point/0\"."},
    "axisRef": {"type": "string", "description": "Work axis the \"point\" placement drills along, e.g. \"origin/axis/z\"."},
    "termination": {"type": "string", "enum": ["distance", "through-all", "to-face", "from-to"], "default": "distance", "description": "Where the bore STOPS. A named terminator must be square to the drill axis — a bore bottoms at one depth."},
    "toFace": {"type": "string", "description": "Stop target for the to-face / from-to (end) terminations: a planar face key, \"plane/N\", or \"origin/plane/xy\"."},
    "toFaceGeom": {"type": "object", "description": "The toFace target named by GEOMETRY (centroid + normal) instead of toFace. Wins over toFace.", "properties": {"centroid": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}, "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}}, "required": ["centroid", "normal"]},
    "fromFace": {"type": "string", "description": "Start target for the from-to termination; the bore begins there rather than at the placement face."},
    "fromFaceGeom": {"type": "object", "description": "The fromFace target named by GEOMETRY instead of fromFace. Wins over fromFace.", "properties": {"centroid": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}, "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}}, "required": ["centroid", "normal"]},
    "diameter": {"type": "string", "description": "Hole diameter with units, e.g. \"5 mm\"."},
    "depth": {"type": "string", "description": "Blind hole depth, e.g. \"8 mm\". Omit (drilled only) for a through-all hole."},
    "counterDiameter": {"type": "string", "description": "Counterbore diameter (type=counterbore)."},
    "counterDepth": {"type": "string", "description": "Counterbore depth (type=counterbore)."},
    "sinkDiameter": {"type": "string", "description": "Countersink top diameter (type=countersink)."},
    "includedAngle": {"type": "string", "description": "Countersink included angle, e.g. \"90 deg\" (type=countersink)."},
    "designation": {"type": "string", "description": "Thread designation, e.g. \"M5x0.8\" (type=tapped)."},
    "center": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Explicit drill point [x,y,z] in model-space cm (projected onto the picked face). Omit to drill at the face centroid. Needed to place more than one hole on a face."},
    "centerExpr": {"type": "array", "items": {"type": "string"}, "minItems": 3, "maxItems": 3, "description": "Parameter-expression form of center: [x,y,z] each an expression with units (e.g. \"L/2\"). Overrides center when present."},
    "placementFaceGeom": {"type": "object", "description": "Select the placement face by GEOMETRY instead of faceRef, so the binding survives recompute (for an external author that cannot mint a stable key). Give either this or faceRef.", "properties": {"centroid": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Face centroid [x,y,z] cm."}, "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Outward unit normal [x,y,z]."}}, "required": ["centroid", "normal"]},
    "drillPoint": {"type": "string", "enum": ["flat", "angled"], "default": "flat", "description": "Blind-hole bottom: \"flat\" (disc) or \"angled\" (cone of tipAngle). Inventor's HoleDrillPointTypeEnum."},
    "tipAngle": {"type": "string", "description": "Included angle of an \"angled\" drill point, e.g. \"118 deg\" (the default when omitted)."}
  },
  "required": []
}`

func holeDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindHole, Summary: "Drill a hole into a picked face: drilled (blind/through), counterbore, countersink, or tapped.", Schema: json.RawMessage(holeSchema), Apply: applyHole}
}

func applyHole(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Hole](s, raw)
	if err != nil {
		return nil, err
	}
	if err := requireHoleSizeAndFace(in); err != nil {
		return nil, err
	}
	dia, err := optionalLengthClosure(part, in.Diameter, "hole: diameter")
	if err != nil {
		return nil, err
	}
	pf, err := buildHole(part, in, dia)
	if err != nil {
		return nil, err
	}
	if err := configureHole(part, pf, in); err != nil {
		return nil, err
	}
	return recomputeResult(part, pf)
}

// requireHoleSizeAndFace rejects the two requests that name no hole at all, so each is reported as
// the caller error it is rather than surfacing later as a zero-diameter recompute failure.
func requireHoleSizeAndFace(in featureargs.Hole) error {
	// Every placement but "point" starts its bore on a face; on-point drills at a work point along a
	// work axis, so demanding a face there would make the placement unusable (#1861).
	if in.FaceRef == "" && in.PlacementFaceGeom == nil && strings.TrimSpace(in.Placement) != "point" {
		return errors.New("hole: needs faceRef or placementFaceGeom (or the \"point\" placement)")
	}
	// A clearance hole is sized by its fastener, so it carries no diameter of its own (#1862).
	if strings.TrimSpace(in.Diameter) == "" && in.Clearance == nil {
		return errors.New("hole: needs \"diameter\", or a \"clearance\" fastener to size it from")
	}
	return nil
}

// configureHole applies everything that sits on top of the seat the builder chose: the drill point,
// the placement (both the geometric face selector and the placement rule), the tap and clearance
// axes, and the termination.
func configureHole(part *compdef.PartComponentDefinition, pf *feature.PartFeature,
	in featureargs.Hole) error {
	if err := applyHoleCenter(part, pf, in); err != nil {
		return err
	}
	if err := applyHolePlacementGeom(pf, in); err != nil {
		return err
	}
	if err := applyHoleDrillPoint(part, pf, in); err != nil {
		return err
	}
	if err := applyHoleOptions(part, pf, in); err != nil {
		return err
	}
	placement, err := buildHolePlacement(part, in)
	if err != nil {
		return err
	}
	pf.Definition().(*feature.HoleFeature).Definition().Placement = placement
	return nil
}

// applyHoleDrillPoint sets a blind hole's conical bottom from drillPoint/tipAngle, wiring the
// featureargs to the model's HoleDefinition.PointAngle: "flat" (or empty) keeps the flat bottom;
// "angled" sets the included cone angle, defaulting to the standard 118° twist-drill point. #1863.
func applyHoleDrillPoint(part *compdef.PartComponentDefinition, pf *feature.PartFeature, in featureargs.Hole) error {
	switch strings.ToLower(strings.TrimSpace(in.DrillPoint)) {
	case "", "flat":
		return nil
	case "angled":
		angle, err := holeTipAngle(part, in.TipAngle)
		if err != nil {
			return err
		}
		hf, ok := pf.Definition().(*feature.HoleFeature)
		if !ok {
			return errors.New("hole: drillPoint applies only to a drilled/tapped hole")
		}
		hf.Definition().PointAngle = angle
		return nil
	default:
		return fmt.Errorf("hole: unknown drillPoint %q (want flat|angled)", in.DrillPoint)
	}
}

// holeTipAngle parses the drill-point included angle, defaulting to 118° (the standard twist-drill
// point) when omitted. #1863.
func holeTipAngle(part *compdef.PartComponentDefinition, expr string) (func() float64, error) {
	if strings.TrimSpace(expr) == "" {
		expr = "118 deg"
	}
	return angleClosure(part, expr, "hole: tipAngle")
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
	case "counterbore", "spotface":
		return buildSeatedHole(part, holes, in, key, dia)
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
		// The older spelling of a DRILLED hole that is also tapped; applyHoleOptions adds the tap,
		// which is what makes a tapped counterbore expressible at all (#1862).
		depth, err := lengthClosure(part, in.Depth, "hole: depth")
		if err != nil {
			return nil, err
		}
		return holes.AddDrilled(key, dia, depth), nil
	default:
		return nil, fmt.Errorf("hole: unknown type %q (want drilled|counterbore|countersink|spotface|tapped)", in.Type)
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

// buildSeatedHole builds the two flat-bottomed seats. A spotface IS a counterbore's recess, so they
// share the builder; the seat type is then set apart, because the distinction is real to everything
// downstream — callouts, hole notes, CAM (#1862).
func buildSeatedHole(part *compdef.PartComponentDefinition, holes *feature.HoleFeatures,
	in featureargs.Hole, key []byte, dia func() float64) (*feature.PartFeature, error) {
	depth, cdia, cdepth, err := threeLengths(part, in.Depth, in.CounterDiameter, in.CounterDepth, "hole "+in.Type)
	if err != nil {
		return nil, err
	}
	pf := holes.AddCounterbore(key, dia, depth, cdia, cdepth)
	if in.Type == "spotface" {
		pf.Definition().(*feature.HoleFeature).Definition().Type = feature.SpotFaceHole
	}
	return pf, nil
}
