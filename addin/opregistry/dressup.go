// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The dress-up (subtractive/modifying) feature operations — fillet, chamfer, shell, draft.
// Each acts on an existing body's edges or faces, referenced by key (get_reference_keys), and
// follows the extrude descriptor shape: a JSON schema + an Apply that builds the feature and
// recomputes. They are how an MCP driver exercises the subtractive kernel end to end.

const filletSchema = `{
  "type": "object",
  "properties": {
    "edgeRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the edges to round (from get_reference_keys). Flat form: one constant radius over these edges."},
    "width": {"type": "string", "description": "Face-fillet form: size the blend by the CHORD it spans instead of by the rolling ball's radius, e.g. \"4 mm\" — Inventor's chordal alternative, and what is measured on the part. Resolved against the angle the two face sets meet at, so they must share an edge and be planar. Wins over radius."},
    "radius": {"type": "string", "description": "Fillet radius with units, e.g. \"3 mm\" (flat and face-fillet forms)."},
    "faceRefsA": {"type": "array", "items": {"type": "string"}, "description": "Face-fillet form (#694): first face set (reference keys). With faceRefsB and a radius (or width), rounds the edges the two sets share — pick by face instead of by edge. Sets that share NO edge are healed to the virtual edge their planes define and rounded there (#694)."},
    "faceRefsB": {"type": "array", "items": {"type": "string"}, "description": "Face-fillet form: second face set."},
    "edgeSets": {"type": "array", "minItems": 1, "description": "Edge-set form (takes precedence over edgeRefs): any mix of constant and variable radius sets.", "items": {"type": "object", "properties": {
      "edgeRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1},
      "radius": {"type": "string", "description": "Constant radius for this set, e.g. \"3 mm\"."},
      "startRadius": {"type": "string", "description": "Variable set: radius at the edge's start vertex (the set holds exactly one edge)."},
      "endRadius": {"type": "string", "description": "Variable set: radius at the edge's end vertex."},
      "radiusPoints": {"type": "array", "description": "Variable set: optional intermediate radius stops along the edge (#695), each strictly between the ends and strictly increasing in t.", "items": {"type": "object", "properties": {
        "t": {"type": "number", "description": "Fraction along the edge from start vertex (0) to end vertex (1); 0<t<1."},
        "radius": {"type": "string", "description": "Radius at this stop, e.g. \"4 mm\"."}
      }, "required": ["t", "radius"]}}
    }, "required": ["edgeRefs"]}},
    "cornerType": {"type": "string", "enum": ["miter", "setback", "round"], "default": "miter", "description": "How a vertex where two filleted edges meet (third edge sharp) is treated: miter (exact crease), round (fillets the third edge into a smooth sphere). setback tapers the third edge to a run-out, giving a smooth set-back sphere."},
    "crossSection": {"type": "string", "enum": ["arc", "g2", "conic"], "default": "arc", "description": "Blend cross-section shape (#1284): arc = circular rolling-ball (G1, default), g2 = curvature-continuous (no highlight break at the tangency lines), conic = rho-controlled. G2/conic apply to planar-walled edge fillets."},
    "rho": {"type": "number", "minimum": 0.1, "maximum": 0.9, "description": "Conic fullness when crossSection=conic: 0.5 = parabola, lower = flatter, higher = fuller."},
    "concaveStrategy": {"type": "string", "enum": ["outward", "inward"], "default": "outward", "description": "Concave (internal) edge handling: outward fills the inside corner with an exact rolling-ball cylinder (default). inward rounds a recess into the corner and is only valid where the faces extend into the material (e.g. a pocket). Convex edges ignore this."},
    "edgesGeom": {"type": "array", "description": "Select the rounded edges by GEOMETRY instead of edgeRefs, so the binding survives recompute (for an external author that cannot mint stable keys). Give either this + radius, edgeRefs, or edgeSets.", "items": {"type": "object", "properties": {"midpoint": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Edge midpoint [x,y,z] cm."}, "direction": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Unit tangent [x,y,z]."}}, "required": ["midpoint", "direction"]}}
  }
}`

const chamferSchema = `{
  "type": "object",
  "properties": {
    "edgeRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the edges to bevel (from get_reference_keys)."},
    "distance": {"type": "string", "description": "Chamfer setback with units, e.g. \"2 mm\" (the first face for the asymmetric modes)."},
    "chamferType": {"type": "string", "enum": ["distance", "distanceAndAngle", "twoDistances"], "default": "distance", "description": "Setback mode."},
    "distance2": {"type": "string", "description": "twoDistances: setback on the second face, e.g. \"4 mm\"."},
    "angle": {"type": "string", "description": "distanceAndAngle: chamfer-face angle, e.g. \"30 deg\"."},
    "referenceFace": {"type": "string", "description": "Face the \"distance\" is measured on for the asymmetric modes (reference key). Without it the assignment falls to the edge's face order, a topology artefact that can put the larger setback on the wrong face of mirrored geometry."},
    "partialStart": {"type": "string", "description": "Where the bevel starts along each edge, measured from its start vertex, e.g. \"5 mm\" (default 0)."},
    "partialLength": {"type": "string", "description": "How much of each edge the bevel covers, e.g. \"20 mm\". Omit for the whole edge — Inventor's partial chamfer."},
    "concaveStrategy": {"type": "string", "enum": ["outward", "inward"], "default": "outward", "description": "Concave (internal) edge handling: outward fills the inside corner with material (default), inward cuts a recessed relief groove. Convex edges ignore this."},
    "edgesGeom": {"type": "array", "description": "Select the bevelled edges by GEOMETRY instead of edgeRefs, so the binding survives recompute. Give either this or edgeRefs.", "items": {"type": "object", "properties": {"midpoint": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Edge midpoint [x,y,z] cm."}, "direction": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Unit tangent [x,y,z]."}}, "required": ["midpoint", "direction"]}}
  },
  "required": ["distance"]
}`

func filletDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindFillet, Summary: "Round picked edges of a body by a radius.", Schema: json.RawMessage(filletSchema), Apply: applyFillet}
}

func chamferDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindChamfer, Summary: "Bevel picked edges of a body by a setback distance.", Schema: json.RawMessage(chamferSchema), Apply: applyChamfer}
}

const ruleFilletSchema = `{
  "type": "object",
  "properties": {
    "rule": {"type": "string", "enum": ["allRounds", "allFillets", "allEdges"], "default": "allRounds", "description": "Which edges to round, by dihedral class: allRounds = every convex (outside) edge, allFillets = every concave (inside) edge, allEdges = both."},
    "radius": {"type": "string", "description": "Fillet radius with units, e.g. \"1 mm\"."}
  },
  "required": ["radius"]
}`

func ruleFilletDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindRuleFillet, Summary: "Round a whole class of a body's edges (all rounds / all fillets) in one feature.", Schema: json.RawMessage(ruleFilletSchema), Apply: applyRuleFillet}
}

func applyRuleFillet(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.RuleFillet](s, raw)
	if err != nil {
		return nil, err
	}
	rule := feature.RuleFilletAllRounds
	if in.Rule != "" {
		r, ok := feature.ParseRuleFilletRule(in.Rule)
		if !ok {
			return nil, fmt.Errorf("ruleFillet: unknown rule %q (want allRounds, allFillets, or allEdges)", in.Rule)
		}
		rule = r
	}
	radius, err := lengthClosure(part, in.Radius, "ruleFillet: radius")
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddRuleFillet(rule, radius)
	return recomputeResult(part, pf)
}

func applyFillet(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Fillet](s, raw)
	if err != nil {
		return nil, err
	}
	corner, cs, prof, err := filletControls(in)
	if err != nil {
		return nil, err
	}
	if len(in.FaceRefsA) > 0 || len(in.FaceRefsB) > 0 {
		return applyFaceFillet(part, in)
	}
	if len(in.EdgeSets) > 0 {
		return applyFilletSets(part, in.EdgeSets, corner, cs, prof)
	}
	return applyFilletFlat(part, in, corner, cs, prof)
}

// blendProfileArgs carries the parsed cross-section + rho into the fillet apply functions.
type blendProfileArgs struct {
	cross types.FilletCrossSection
	rho   float64
}

// filletControls parses the fillet op's shared controls: the corner treatment, concave strategy, and
// blend cross-section/rho (each defaulting when absent, erroring on an unknown spelling).
func filletControls(in featureargs.Fillet) (types.FilletCornerType, types.FilletConcaveStrategy, blendProfileArgs, error) {
	corner, err := filletCornerOf(in.CornerType)
	if err != nil {
		return 0, 0, blendProfileArgs{}, err
	}
	cs, err := filletConcaveStrategyOf(in.ConcaveStrategy)
	if err != nil {
		return 0, 0, blendProfileArgs{}, err
	}
	cross, err := filletCrossOf(in.CrossSection)
	if err != nil {
		return 0, 0, blendProfileArgs{}, err
	}
	return corner, cs, blendProfileArgs{cross: cross, rho: in.Rho}, nil
}

// filletCrossOf resolves the optional crossSection wire spelling (empty ⇒ arc).
func filletCrossOf(spelling string) (types.FilletCrossSection, error) {
	c, ok := types.ParseFilletCrossSection(spelling)
	if !ok {
		return "", fmt.Errorf("fillet: unknown crossSection %q (want arc, g2, or conic)", spelling)
	}
	return c, nil
}

const fullRoundSchema = `{
  "type": "object",
  "properties": {
    "side1Ref": {"type": "string", "description": "First side face (reference key)."},
    "centerRef": {"type": "string", "description": "Center face to replace with the round."},
    "side2Ref": {"type": "string", "description": "Second side face, parallel to the first."}
  },
  "required": ["side1Ref", "centerRef", "side2Ref"]
}`

func fullRoundDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindFullRoundFillet,
		Summary: "Replace a center face with a full round (half-cylinder) tangent to two parallel side faces.",
		Schema:  json.RawMessage(fullRoundSchema),
		Apply:   applyFullRound,
	}
}

func applyFullRound(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.FullRoundFillet](s, raw)
	if err != nil {
		return nil, err
	}
	if in.Side1Ref == "" || in.CenterRef == "" || in.Side2Ref == "" {
		return nil, errors.New("full round: side1Ref, centerRef and side2Ref are all required")
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddFullRoundFillet(
		refKeys([]string{in.Side1Ref}), refKeys([]string{in.CenterRef}), refKeys([]string{in.Side2Ref}))
	return recomputeResult(part, pf)
}

// applyFaceFillet rounds the edges shared between two face sets (#694, adjacent-faces case): both
// faceRefsA and faceRefsB plus a radius (or a chordal width) are required.
func applyFaceFillet(part *compdef.PartComponentDefinition, in featureargs.Fillet) (json.RawMessage, error) {
	if len(in.FaceRefsA) == 0 || len(in.FaceRefsB) == 0 {
		return nil, errors.New("face fillet: both faceRefsA and faceRefsB are required")
	}
	if strings.TrimSpace(in.Radius) == "" && strings.TrimSpace(in.Width) == "" {
		return nil, errors.New("face fillet: needs \"radius\", or \"width\" to size the blend by its chord")
	}
	r, err := optionalLengthClosure(part, in.Radius, fieldFilletRadius)
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddFaceFillet(refKeys(in.FaceRefsA), refKeys(in.FaceRefsB), r)
	// A width is the CHORD across the blend (#1887); the model resolves it to a radius against the
	// angle the two face sets meet at, so it wins over any radius also given.
	if strings.TrimSpace(in.Width) != "" {
		w, err := lengthClosure(part, in.Width, "fillet: width")
		if err != nil {
			return nil, err
		}
		pf.Definition().(*feature.FaceFilletFeature).Definition().Width = w
	}
	return recomputeResult(part, pf)
}

// applyFilletFlat builds the flat (edgeRefs + single radius) fillet with the chosen corner
// treatment and concave-edge strategy.
func applyFilletFlat(part *compdef.PartComponentDefinition, in featureargs.Fillet, corner types.FilletCornerType, cs types.FilletConcaveStrategy, prof blendProfileArgs) (json.RawMessage, error) {
	if len(in.EdgeRefs) == 0 && len(in.EdgesGeom) == 0 {
		return nil, errors.New("fillet: edgeRefs is empty (give edgeRefs+radius, edgesGeom, or edgeSets)")
	}
	r, err := lengthClosure(part, in.Radius, fieldFilletRadius)
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddFilletCorner(refKeys(in.EdgeRefs), r, corner)
	if err := setFilletGeomEdges(pf, in.EdgesGeom); err != nil {
		return nil, err
	}
	applyDressProfile(pf, cs, prof)
	return recomputeResult(part, pf)
}

// setFilletGeomEdges binds the fillet's edges by geometry when edgesGeom is given, so an external
// author's edge selection survives recompute (bindGeomEdges folds them into the edge list).
func setFilletGeomEdges(pf *feature.PartFeature, sels []featureargs.GeomEdgeSel) error {
	if len(sels) == 0 {
		return nil
	}
	refs, err := geomEdgeRefs(sels)
	if err != nil {
		return err
	}
	pf.Definition().(*feature.FilletFeature).Definition().GeomEdges = refs
	return nil
}

// applyFilletSets decodes the edge-set form and adds the fillet with the chosen corner treatment,
// concave-edge strategy, and blend cross-section.
func applyFilletSets(part *compdef.PartComponentDefinition, args []featureargs.FilletEdgeSet, corner types.FilletCornerType, cs types.FilletConcaveStrategy, prof blendProfileArgs) (json.RawMessage, error) {
	sets, err := filletSetsFromArgs(part, args)
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddFilletSetsCorner(sets, corner)
	applyDressProfile(pf, cs, prof)
	return recomputeResult(part, pf)
}

// applyDressProfile sets the concave strategy and blend cross-section/rho on a freshly-added fillet.
func applyDressProfile(pf *feature.PartFeature, cs types.FilletConcaveStrategy, prof blendProfileArgs) {
	def := pf.Definition().(*feature.FilletFeature).Definition()
	def.ConcaveStrategy = cs
	def.CrossSection = prof.cross
	def.Rho = prof.rho
}

// filletConcaveStrategyOf resolves the optional concaveStrategy wire spelling (empty ⇒ outward).
func filletConcaveStrategyOf(spelling string) (types.FilletConcaveStrategy, error) {
	if spelling == "" {
		return types.FilletConcaveOutward, nil
	}
	v, ok := types.ParseFilletConcaveStrategy(spelling)
	if !ok {
		return 0, fmt.Errorf("fillet: unknown concaveStrategy %q (want outward or inward)", spelling)
	}
	return v, nil
}

// filletCornerOf resolves the optional cornerType wire spelling (empty ⇒ miter), erroring on an
// unknown value with the accepted set.
func filletCornerOf(spelling string) (types.FilletCornerType, error) {
	if spelling == "" {
		return types.FilletCornerMiter, nil
	}
	c, ok := types.ParseFilletCornerType(spelling)
	if !ok {
		return 0, fmt.Errorf("fillet: unknown cornerType %q (want miter, setback, or round)", spelling)
	}
	return c, nil
}

// filletSetsFromArgs decodes the edge-set form: each set is constant (radius) or variable
// (startRadius+endRadius); giving both or neither is a precise error.
func filletSetsFromArgs(part *compdef.PartComponentDefinition, args []featureargs.FilletEdgeSet) ([]feature.FilletEdgeSet, error) {
	out := make([]feature.FilletEdgeSet, len(args))
	for i, a := range args {
		if len(a.EdgeRefs) == 0 {
			return nil, fmt.Errorf("fillet: edgeSets[%d].edgeRefs is empty", i)
		}
		radii, err := filletSetRadii(part, a, i)
		if err != nil {
			return nil, err
		}
		radii.EdgeKeys = refKeys(a.EdgeRefs)
		out[i] = radii
	}
	return out, nil
}

// filletSetRadii resolves one set's radius closures from its constant or variable spelling.
func filletSetRadii(part *compdef.PartComponentDefinition, a featureargs.FilletEdgeSet, i int) (feature.FilletEdgeSet, error) {
	hasConst, hasVar := a.Radius != "", a.StartRadius != "" || a.EndRadius != ""
	if hasConst == hasVar {
		return feature.FilletEdgeSet{}, fmt.Errorf("fillet: edgeSets[%d] needs radius OR startRadius+endRadius (got radius=%q start=%q end=%q)", i, a.Radius, a.StartRadius, a.EndRadius)
	}
	if hasConst {
		r, err := lengthClosure(part, a.Radius, fieldFilletRadius)
		return feature.FilletEdgeSet{Radius: r}, err
	}
	r0, err := lengthClosure(part, a.StartRadius, "fillet: startRadius")
	if err != nil {
		return feature.FilletEdgeSet{}, err
	}
	r1, err := lengthClosure(part, a.EndRadius, "fillet: endRadius")
	if err != nil {
		return feature.FilletEdgeSet{}, err
	}
	pts, err := filletRadiusPoints(part, a.RadiusPoints)
	return feature.FilletEdgeSet{StartRadius: r0, EndRadius: r1, RadiusPoints: pts}, err
}

// filletRadiusPoints resolves a variable set's intermediate radius stops' closures (#695).
func filletRadiusPoints(part *compdef.PartComponentDefinition, pts []featureargs.FilletRadiusPoint) ([]feature.FilletRadiusPoint, error) {
	if len(pts) == 0 {
		return nil, nil
	}
	out := make([]feature.FilletRadiusPoint, len(pts))
	for i, p := range pts {
		r, err := lengthClosure(part, p.Radius, "fillet: radiusPoints radius")
		if err != nil {
			return nil, err
		}
		out[i] = feature.FilletRadiusPoint{T: p.T, Radius: r}
	}
	return out, nil
}

func applyChamfer(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Chamfer](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.EdgeRefs) == 0 && len(in.EdgesGeom) == 0 {
		return nil, errors.New("chamfer: edgeRefs is empty (give edgeRefs or edgesGeom)")
	}
	pf, err := buildChamfer(part, in)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, pf)
}

// buildChamfer resolves the chamfer mode, setbacks, and concave-edge strategy from the args and
// adds the feature (not yet recomputed).
func buildChamfer(part *compdef.PartComponentDefinition, in featureargs.Chamfer) (*feature.PartFeature, error) {
	d, err := lengthClosure(part, in.Distance, "chamfer: distance")
	if err != nil {
		return nil, err
	}
	ct, err := chamferTypeOf(in.ChamferType)
	if err != nil {
		return nil, err
	}
	cs, err := chamferConcaveStrategyOf(in.ConcaveStrategy)
	if err != nil {
		return nil, err
	}
	def := &feature.ChamferDefinition{
		EdgeKeys: refKeys(in.EdgeRefs), Distance: d, Type: ct, FlatCorners: true, ConcaveStrategy: cs,
	}
	if err := setChamferModeInput(part, def, in); err != nil {
		return nil, err
	}
	if err := setChamferRun(part, def, in); err != nil {
		return nil, err
	}
	if len(in.EdgesGeom) > 0 {
		// Bind the chamfered edges by geometry when authored geometrically (survives recompute).
		refs, err := geomEdgeRefs(in.EdgesGeom)
		if err != nil {
			return nil, err
		}
		def.GeomEdges = refs
	}
	return feature.NewDressUpFeatures(part.Features()).AddChamferDef(def), nil
}

// chamferTypeOf resolves the chamfer mode spelling, defaulting to equal-distance.
func chamferTypeOf(spelling string) (types.ChamferType, error) {
	if spelling == "" {
		return types.ChamferDistance, nil
	}
	v, ok := types.ParseChamferType(spelling)
	if !ok {
		return 0, fmt.Errorf("chamfer: unknown chamferType %q", spelling)
	}
	return v, nil
}

// chamferConcaveStrategyOf resolves the concave-edge strategy spelling, defaulting to outward.
func chamferConcaveStrategyOf(spelling string) (types.ChamferConcaveStrategy, error) {
	if spelling == "" {
		return types.ChamferConcaveOutward, nil
	}
	v, ok := types.ParseChamferConcaveStrategy(spelling)
	if !ok {
		return 0, fmt.Errorf("chamfer: unknown concaveStrategy %q (want outward or inward)", spelling)
	}
	return v, nil
}

// setChamferModeInput resolves the second input the asymmetric modes take (distance2 or angle)
// onto the definition. The equal-distance mode takes none.
func setChamferModeInput(part *compdef.PartComponentDefinition, def *feature.ChamferDefinition, in featureargs.Chamfer) error {
	switch def.Type {
	case types.ChamferTwoDistances:
		d2, err := lengthClosure(part, in.Distance2, "chamfer: distance2")
		if err != nil {
			return err
		}
		def.Distance2 = d2
	case types.ChamferDistanceAndAngle:
		a, err := angleClosure(part, in.Angle, "chamfer: angle")
		if err != nil {
			return err
		}
		def.Angle = a
	}
	return nil
}

const shellSchema = `{
  "type": "object",
  "properties": {
    "faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to remove, hollowing the body (from get_reference_keys)."},
    "thickness": {"type": "string", "description": "Remaining wall thickness with units, e.g. \"1 mm\"."},
    "facesGeom": {"type": "array", "description": "Select the removed faces by GEOMETRY instead of faceRefs, so the binding survives recompute. Give either this or faceRefs.", "items": {"type": "object", "properties": {"centroid": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Face centroid [x,y,z] cm."}, "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Outward unit normal [x,y,z]."}}, "required": ["centroid", "normal"]}},
    "direction": {"type": "string", "enum": ["inside", "outside", "both"], "default": "inside", "description": "Which side the wall grows onto: \"inside\" (outer dimensions kept), \"outside\" (outer dimensions grow by thickness), or \"both\" (wall centred on the faces). Inventor's ShellDirectionEnum."}
  },
  "required": ["thickness"]
}`

const draftSchema = `{
  "type": "object",
  "properties": {
    "faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to draft (from get_reference_keys)."},
    "angle": {"type": "string", "description": "Draft angle with units, e.g. \"3 deg\"."},
    "pullDirection": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Explicit pull/parting direction [dx,dy,dz] (only its orientation matters). Omit to let the host infer it from the neutral faces."},
    "facesGeom": {"type": "array", "description": "Select the drafted faces by GEOMETRY instead of faceRefs, so the binding survives recompute. Give either this or faceRefs.", "items": {"type": "object", "properties": {"centroid": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Face centroid [x,y,z] cm."}, "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Outward unit normal [x,y,z]."}}, "required": ["centroid", "normal"]}},
    "neutralPlane": {"type": "string", "description": "Fixed-plane draft: a planar face key, work plane (\"plane/N\"), or origin plane (\"origin/plane/xy\"). Faces pivot on their intersection with it (dimensions in the plane preserved); pull defaults to its normal. Inventor's kFixedPlaneFaceDraftDefinitionType."}
  },
  "required": ["angle"]
}`

func shellDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindShell, Summary: "Hollow a body to a wall thickness, removing the picked faces.", Schema: json.RawMessage(shellSchema), Apply: applyShell}
}

func draftDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindDraft, Summary: "Taper picked faces by a draft angle.", Schema: json.RawMessage(draftSchema), Apply: applyDraft}
}

func applyShell(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Shell](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.FaceRefs) == 0 && len(in.FacesGeom) == 0 {
		return nil, errors.New("shell: faceRefs is empty (give faceRefs or facesGeom)")
	}
	th, err := lengthClosure(part, in.Thickness, "shell: thickness")
	if err != nil {
		return nil, err
	}
	dir, ok := feature.ParseShellDirection(strings.ToLower(strings.TrimSpace(in.Direction)))
	if !ok {
		return nil, fmt.Errorf("shell: unknown direction %q (want inside|outside|both)", in.Direction)
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddShell(refKeys(in.FaceRefs), th)
	def := pf.Definition().(*feature.ShellFeature).Definition()
	def.Direction = dir
	if len(in.FacesGeom) > 0 {
		// Bind the removed faces by geometry when authored geometrically (survives recompute).
		refs, err := geomFaceRefs(in.FacesGeom)
		if err != nil {
			return nil, err
		}
		def.GeomFaces = refs
	}
	return recomputeResult(part, pf)
}

func applyDraft(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Draft](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.FaceRefs) == 0 && len(in.FacesGeom) == 0 {
		return nil, errors.New("draft: faceRefs is empty (give faceRefs or facesGeom)")
	}
	a, err := angleClosure(part, in.Angle, "draft: angle")
	if err != nil {
		return nil, err
	}
	pf, err := buildDraft(part, in, a)
	if err != nil {
		return nil, err
	}
	if len(in.FacesGeom) > 0 {
		// Bind the drafted faces by geometry when authored geometrically (survives recompute).
		refs, err := geomFaceRefs(in.FacesGeom)
		if err != nil {
			return nil, err
		}
		pf.Definition().(*feature.FaceDraftFeature).Definition().GeomFaces = refs
	}
	return recomputeResult(part, pf)
}

// buildDraft adds the draft feature: a fixed-plane (neutral) draft when neutralPlane is given,
// else an explicit-pull draft (AddDraftPull) when pullDirection is given, else the host's inferred
// pull (AddDraft, default +Z).
func buildDraft(part *compdef.PartComponentDefinition, in featureargs.Draft, angle func() float64) (*feature.PartFeature, error) {
	du := feature.NewDressUpFeatures(part.Features())
	keys := refKeys(in.FaceRefs)
	if strings.TrimSpace(in.NeutralPlane) != "" {
		return buildNeutralDraft(part, du, in, keys, angle)
	}
	if len(in.PullDirection) == 0 {
		return du.AddDraft(keys, angle), nil
	}
	pull, err := vec3(in.PullDirection, "draft: pullDirection")
	if err != nil {
		return nil, err
	}
	return du.AddDraftPull(keys, pull, angle), nil
}

// buildNeutralDraft resolves the fixed neutral plane (a planar face key, "plane/N", or origin
// plane) and drafts the faces pivoting on their intersection with it — Inventor's fixed-plane
// draft. The pull defaults to the plane normal unless pullDirection overrides it. #1866.
func buildNeutralDraft(part *compdef.PartComponentDefinition, du *feature.DressUpFeatures, in featureargs.Draft, keys [][]byte, angle func() float64) (*feature.PartFeature, error) {
	wp, err := part.WorkGeometry().PlaneTargetFromRef(in.NeutralPlane)
	if err != nil {
		return nil, fmt.Errorf("draft: neutralPlane %q: %w", in.NeutralPlane, err)
	}
	pl := wp.Plane()
	neutral, err := geom.NewPlane(pl.Origin(), pl.Normal().AsVector())
	if err != nil {
		return nil, fmt.Errorf("draft: neutralPlane %q: %w", in.NeutralPlane, err)
	}
	pull := pl.Normal().AsVector()
	if len(in.PullDirection) > 0 {
		if pull, err = vec3(in.PullDirection, "draft: pullDirection"); err != nil {
			return nil, err
		}
	}
	return du.AddDraftPullNeutral(keys, pull, &neutral, angle), nil
}

// --- lip / groove (M20-F10) ------------------------------------------------

const lipSchema = `{
  "type": "object",
  "properties": {
    "edgeRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the edges to run the bead along (from get_reference_keys)."},
    "width": {"type": "string", "description": "Bead width along the first adjacent face, e.g. \"2 mm\"."},
    "height": {"type": "string", "description": "Bead height along the second adjacent face, e.g. \"2 mm\"."},
    "groove": {"type": "boolean", "default": false, "description": "Cut a recessed groove instead of raising a lip."}
  },
  "required": ["edgeRefs", "width", "height"]
}`

func lipDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindLip, Summary: "Run a raised lip (or recessed groove) bead along picked edges.", Schema: json.RawMessage(lipSchema), Apply: applyLip}
}

func applyLip(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Lip](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.EdgeRefs) == 0 {
		return nil, errors.New("lip: edgeRefs is empty")
	}
	w, err := lengthClosure(part, in.Width, "lip: width")
	if err != nil {
		return nil, err
	}
	h, err := lengthClosure(part, in.Height, "lip: height")
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddLip(refKeys(in.EdgeRefs), w, h, in.Groove)
	return recomputeResult(part, pf)
}

// setChamferRun records which face the first setback is measured on and how much of each edge the
// bevel covers (#1888). Both are optional; absent, the chamfer runs the whole edge with the
// setbacks in the edge's own face order, exactly as before.
func setChamferRun(part *compdef.PartComponentDefinition, def *feature.ChamferDefinition,
	in featureargs.Chamfer) error {
	def.ReferenceFace = []byte(in.ReferenceFace)
	if strings.TrimSpace(in.PartialLength) == "" {
		return nil
	}
	length, err := lengthClosure(part, in.PartialLength, "chamfer: partialLength")
	if err != nil {
		return err
	}
	start, err := optionalLengthClosure(part, in.PartialStart, "chamfer: partialStart")
	if err != nil {
		return err
	}
	def.PartialStart, def.PartialLength = start, length
	return nil
}
