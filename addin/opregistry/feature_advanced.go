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
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// The remaining feature operations that need a custom resolver: sweep (a profile along a path
// sketch's chain), move body (a transform), replace face, core/cavity tooling, split solid by
// a work plane, and a sketch-driven pattern.

// --- sweep -----------------------------------------------------------------

const sweepSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Sketch holding the profile to sweep (omit for definitionType \"solid\")."},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "pathSketchIndex": {"type": "integer", "minimum": 0, "description": "Sketch holding the open path (rail) to sweep along (used when pathPoints is absent)."},
    "pathIndex": {"type": "integer", "minimum": 0, "default": 0, "description": "Which open path of the path sketch (see list_sketch_profiles / the sketch's chains)."},
    "pathPoints": {"type": "array", "items": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}, "minItems": 2, "description": "Explicit 3D polyline path in cm (each entry [x,y,z]), mirroring how a loft rail's points override its sketch path. When given, the sketch path (pathSketchIndex/pathIndex) is ignored."},
    "definitionType": {"type": "string", "enum": ["path", "pathAndGuideRail", "pathAndGuideSurface", "pathAndSectionTwists", "solid"], "default": "path", "description": "The sweep definition union discriminator."},
    "orientation": {"type": "string", "enum": ["normalToPath", "parallelToOriginalProfile", "alignToVector"], "default": "normalToPath"},
    "alignVector": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Fixed profile normal for orientation alignToVector."},
    "twist": {"type": "string", "description": "Optional total twist along the path, e.g. \"90 deg\"."},
    "taper": {"type": "string", "description": "Optional draft angle along the path, e.g. \"3 deg\" (constant wall draft)."},
    "twistStations": {"type": "array", "items": {"type": "object", "properties": {"t": {"type": "number"}, "angle": {"type": "string"}}, "required": ["t", "angle"]}, "description": "pathAndSectionTwists rows: twist angle at normalized arclength t."},
    "railSketchIndex": {"type": "integer", "minimum": 0, "description": "Sketch holding the guide rail (pathAndGuideRail)."},
    "railIndex": {"type": "integer", "minimum": 0, "default": 0},
    "scaling": {"type": "string", "enum": ["xy", "x", "none"], "default": "xy", "description": "How the guide rail scales the profile."},
    "guideFaceKey": {"type": "string", "description": "Reference key of the guide face (pathAndGuideSurface)."},
    "toolBodyIndex": {"type": "integer", "minimum": 0, "description": "Running-body index of the tool to drag (definitionType \"solid\")."},
    "operation": {"type": "string", "enum": ["new", "join", "cut", "intersect", "surface"], "default": "new", "description": "Boolean against existing bodies, or \"surface\" to sweep the profile into an open swept-surface (sheet) body — Inventor's kSurfaceOperation."}
  }
}`

func sweepDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindSweep, Summary: "Sweep a profile along an open path (rail) into a solid.", Schema: json.RawMessage(sweepSchema), Apply: applySweep}
}

func applySweep(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Sweep](s, raw)
	if err != nil {
		return nil, err
	}
	def, err := sweepDefinitionFromArgs(part, in)
	if err != nil {
		return nil, err
	}
	pf := feature.NewSweepFeatures(part.Features()).AddDefinition(def)
	return recomputeResult(part, pf)
}

// sweepDefinitionFromArgs builds the union definition from the wire args,
// validating the discriminator's required fields.
func sweepDefinitionFromArgs(part *compdef.PartComponentDefinition, in featureargs.Sweep) (*feature.SweepDefinition, error) {
	path, pathSk, err := sweepPath(part, in)
	if err != nil {
		return nil, err
	}
	def := &feature.SweepDefinition{
		ProfileIndex: in.ProfileIndex,
		Path:         path,
		// Attribute the live path to its sketch so a parameter driving the rail re-sweeps the
		// body (#1414 tail invalidation; Oblikovati#1693). An explicit pathPoints polyline has no
		// sketch, so PathSketch is nil.
		PathSketch: pathSk,
	}
	if err := sweepScalars(part, in, def); err != nil {
		return nil, err
	}
	return def, sweepVariantFields(part, in, def)
}

// sweepPath resolves the sweep's path: an explicit pathPoints polyline (a static path, no sketch)
// when given, otherwise a live provider re-derived from the path sketch each recompute (so a
// parameter driving the rail reshapes the sweep; a snapshot would freeze it at apply time).
func sweepPath(part *compdef.PartComponentDefinition, in featureargs.Sweep) (func() *sketch.Path3D, *sketch.Sketch, error) {
	if len(in.PathPoints) > 0 {
		p, err := path3DFromPoints(in.PathPoints)
		if err != nil {
			return nil, nil, err
		}
		return func() *sketch.Path3D { return p }, nil, nil
	}
	if _, err := pathFromSketch(part, in.PathSketchIndex, in.PathIndex); err != nil {
		return nil, nil, err
	}
	pathSk, err := sketchAt(part, in.PathSketchIndex)
	if err != nil {
		return nil, nil, err
	}
	return func() *sketch.Path3D {
		p, _ := pathFromSketch(part, in.PathSketchIndex, in.PathIndex)
		return p
	}, pathSk, nil
}

// path3DFromPoints builds an open 3D path from an [x,y,z] polyline (cm), mirroring how a loft
// rail consumes its explicit Points. Needs at least two points.
func path3DFromPoints(pts [][]float64) (*sketch.Path3D, error) {
	if len(pts) < 2 {
		return nil, fmt.Errorf("sweep: pathPoints needs at least 2 points, got %d", len(pts))
	}
	out := make([]*sketch.Point3D, len(pts))
	for i, p := range pts {
		if len(p) != 3 {
			return nil, fmt.Errorf("sweep: pathPoints[%d] needs [x,y,z], got %v", i, p)
		}
		out[i] = sketch.NewPoint3D(math.P3(p[0], p[1], p[2]))
	}
	return sketch.NewPath3D(out, false), nil
}

// sweepScalars resolves the profile sketch, twist, taper, orientation and
// operation (the fields every profile variant shares).
func sweepScalars(part *compdef.PartComponentDefinition, in featureargs.Sweep, def *feature.SweepDefinition) error {
	if in.DefinitionType != "solid" {
		sk, err := sketchAt(part, in.SketchIndex)
		if err != nil {
			return err
		}
		def.Sketch = sk
	}
	twist, err := optionalAngleClosure(part, in.Twist, "sweep: twist")
	if err != nil {
		return err
	}
	def.Twist = twist
	taper, err := optionalAngleClosure(part, in.Taper, "sweep: taper")
	if err != nil {
		return err
	}
	def.Taper = taper
	op, err := parseOperation(in.Operation)
	if err != nil {
		return err
	}
	def.Operation = op
	return sweepOrientation(in, def)
}

func sweepOrientation(in featureargs.Sweep, def *feature.SweepDefinition) error {
	if in.Orientation == "" {
		return nil
	}
	o, ok := types.ParseSweepProfileOrientation(in.Orientation)
	if !ok {
		return fmt.Errorf("sweep: unknown orientation %q (want normalToPath, parallelToOriginalProfile or alignToVector)", in.Orientation)
	}
	def.Orientation = o
	if o == types.AlignToVector {
		if len(in.AlignVector) != 3 {
			return fmt.Errorf("sweep: alignToVector needs alignVector as [x, y, z], got %v", in.AlignVector)
		}
		def.AlignVector = math.V3(math.Scalar(in.AlignVector[0]), math.Scalar(in.AlignVector[1]), math.Scalar(in.AlignVector[2]))
	}
	return nil
}

// sweepVariantFields applies the discriminated union variant.
func sweepVariantFields(part *compdef.PartComponentDefinition, in featureargs.Sweep, def *feature.SweepDefinition) error {
	switch in.DefinitionType {
	case "", "path":
		return nil
	case "pathAndGuideRail":
		if _, err := pathFromSketch(part, in.RailSketchIndex, in.RailIndex); err != nil {
			return fmt.Errorf("sweep guide rail: %w", err)
		}
		def.GuideRail = func() *sketch.Path3D {
			p, _ := pathFromSketch(part, in.RailSketchIndex, in.RailIndex)
			return p
		}
		if railSk, err := sketchAt(part, in.RailSketchIndex); err == nil {
			def.GuideRailSketch = railSk // rail-driving parameters re-sweep too (#1693)
		}
		return sweepScaling(in, def)
	case "pathAndGuideSurface":
		if in.GuideFaceKey == "" {
			return fmt.Errorf("sweep: pathAndGuideSurface needs guideFaceKey")
		}
		def.GuideFaceKey = []byte(in.GuideFaceKey)
		return nil
	case "pathAndSectionTwists":
		return sweepStations(part, in, def)
	case "solid":
		if in.ToolBodyIndex == nil {
			return fmt.Errorf("sweep: definitionType solid needs toolBodyIndex")
		}
		def.SolidToolIndex = in.ToolBodyIndex
		return nil
	default:
		return fmt.Errorf("sweep: unknown definitionType %q", in.DefinitionType)
	}
}

func sweepScaling(in featureargs.Sweep, def *feature.SweepDefinition) error {
	if in.Scaling == "" {
		return nil
	}
	sc, ok := types.ParseSweepProfileScaling(in.Scaling)
	if !ok {
		return fmt.Errorf("sweep: unknown scaling %q (want xy, x or none)", in.Scaling)
	}
	def.Scaling = sc
	return nil
}

func sweepStations(part *compdef.PartComponentDefinition, in featureargs.Sweep, def *feature.SweepDefinition) error {
	if len(in.TwistStations) < 2 {
		return fmt.Errorf("sweep: pathAndSectionTwists needs 2+ twistStations, got %d", len(in.TwistStations))
	}
	for i, st := range in.TwistStations {
		angle, err := angleClosure(part, st.Angle, "sweep: twist station angle")
		if err != nil {
			return err
		}
		if i > 0 && st.T <= in.TwistStations[i-1].T {
			return fmt.Errorf("sweep: twist stations must have ascending t (station %d: %g after %g)", i, st.T, in.TwistStations[i-1].T)
		}
		def.TwistStations = append(def.TwistStations, feature.SweepTwistStation{T: st.T, Angle: angle()})
	}
	return nil
}

// pathFromSketch lifts a 2D sketch's open path (chain) to a 3D sweep rail via the sketch's
// plane — the same construction the interactive sweep tool uses.
func pathFromSketch(part *compdef.PartComponentDefinition, sketchIndex, pathIndex int) (*sketch.Path3D, error) {
	sk, err := sketchAt(part, sketchIndex)
	if err != nil {
		return nil, err
	}
	paths := sk.Paths()
	if pathIndex < 0 || pathIndex >= len(paths) {
		return nil, fmt.Errorf("sweep: path %d not found (path sketch has %d paths)", pathIndex, len(paths))
	}
	p := paths[pathIndex]
	plane := sk.Plane()
	pts := p.Points()
	chain := make([]*sketch.Point3D, len(pts))
	for i, q := range pts {
		chain[i] = sketch.NewPoint3D(plane.ToModel(q))
	}
	return sketch.NewPath3D(chain, p.IsClosed()), nil
}

// --- move body -------------------------------------------------------------

const moveBodySchema = `{
  "type": "object",
  "properties": {
    "bodyIndex": {"type": "integer", "minimum": 0, "description": "Body to move (model.tree body order)."},
    "translation": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Simple form: move vector [x,y,z] in cm. Ignored when operations is given."},
    "operations": {"type": "array", "description": "Ordered move operations composed in list order (e.g. rotate then slide). Each is a separately parametric step.", "items": {"type": "object", "properties": {
      "type": {"type": "string", "enum": ["freeDrag", "alongRay", "rotateAboutLine"], "description": "freeDrag: x/y/z offsets. alongRay: dir + dist. rotateAboutLine: point + dir + angle."},
      "x": {"type": "string", "description": "freeDrag X offset, e.g. \"5 mm\"."},
      "y": {"type": "string", "description": "freeDrag Y offset."},
      "z": {"type": "string", "description": "freeDrag Z offset."},
      "dir": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Direction [x,y,z]: the ray for alongRay, the axis for rotateAboutLine."},
      "dist": {"type": "string", "description": "alongRay distance, e.g. \"5 mm\"."},
      "point": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "rotateAboutLine axis point [x,y,z] in cm."},
      "angle": {"type": "string", "description": "rotateAboutLine angle, e.g. \"30 deg\"."}
    }, "required": ["type"]}}
  },
  "required": ["bodyIndex"]
}`

func moveBodyDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindMoveBody, Summary: "Move a solid body by a translation or an ordered list of parametric operations (free-drag / along-ray / rotate-about-line).", Schema: json.RawMessage(moveBodySchema), Apply: applyMoveBody}
}

func applyMoveBody(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.MoveBody](s, raw)
	if err != nil {
		return nil, err
	}
	mods := feature.NewModifyFeatures(part.Features())
	if len(in.Operations) > 0 {
		ops, err := buildMoveOps(part, in.Operations)
		if err != nil {
			return nil, err
		}
		return recomputeResult(part, mods.AddMoveOps(in.BodyIndex, ops))
	}
	t, err := vec3(in.Translation, "moveBody: translation")
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, mods.AddMove(in.BodyIndex, math.Translation4(t)))
}

// buildMoveOps resolves each operation arg into a parametric model move operation.
func buildMoveOps(part *compdef.PartComponentDefinition, args []featureargs.MoveBodyOp) ([]feature.MoveOperation, error) {
	ops := make([]feature.MoveOperation, len(args))
	for i, a := range args {
		op, err := buildMoveOp(part, a)
		if err != nil {
			return nil, fmt.Errorf("moveBody operation %d: %w", i, err)
		}
		ops[i] = op
	}
	return ops, nil
}

// buildMoveOp dispatches on the operation type.
func buildMoveOp(part *compdef.PartComponentDefinition, a featureargs.MoveBodyOp) (feature.MoveOperation, error) {
	switch types.MoveOperationType(a.Type) {
	case types.MoveFreeDrag:
		return buildFreeDragOp(part, a)
	case types.MoveAlongRay:
		return buildAlongRayOp(part, a)
	case types.MoveRotateAboutLine:
		return buildRotateAboutLineOp(part, a)
	default:
		return feature.MoveOperation{}, fmt.Errorf("unknown type %q (want freeDrag/alongRay/rotateAboutLine)", a.Type)
	}
}

func buildFreeDragOp(part *compdef.PartComponentDefinition, a featureargs.MoveBodyOp) (feature.MoveOperation, error) {
	x, err := optionalLengthClosure(part, a.X, "moveBody freeDrag x")
	if err != nil {
		return feature.MoveOperation{}, err
	}
	y, err := optionalLengthClosure(part, a.Y, "moveBody freeDrag y")
	if err != nil {
		return feature.MoveOperation{}, err
	}
	z, err := optionalLengthClosure(part, a.Z, "moveBody freeDrag z")
	if err != nil {
		return feature.MoveOperation{}, err
	}
	return feature.FreeDragOp(x, y, z), nil
}

func buildAlongRayOp(part *compdef.PartComponentDefinition, a featureargs.MoveBodyOp) (feature.MoveOperation, error) {
	dir, err := vec3(a.Dir, "moveBody alongRay dir")
	if err != nil {
		return feature.MoveOperation{}, err
	}
	dist, err := lengthClosure(part, a.Dist, "moveBody alongRay dist")
	if err != nil {
		return feature.MoveOperation{}, err
	}
	return feature.AlongRayOp(dir, dist), nil
}

func buildRotateAboutLineOp(part *compdef.PartComponentDefinition, a featureargs.MoveBodyOp) (feature.MoveOperation, error) {
	dir, err := vec3(a.Dir, "moveBody rotateAboutLine dir")
	if err != nil {
		return feature.MoveOperation{}, err
	}
	point, err := point3(a.Point, "moveBody rotateAboutLine point")
	if err != nil {
		return feature.MoveOperation{}, err
	}
	angle, err := angleClosure(part, a.Angle, "moveBody rotateAboutLine angle")
	if err != nil {
		return feature.MoveOperation{}, err
	}
	return feature.RotateAboutLineOp(point, dir, angle), nil
}

// --- bend part -------------------------------------------------------------

const bendPartSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Sketch holding the bend line."},
    "lineIndex": {"type": "integer", "minimum": 0, "default": 0, "description": "Index of the bend line within the sketch's lines."},
    "bendType": {"type": "string", "enum": ["radiusAndAngle", "radiusAndArcLength", "arcLengthAndAngle"], "default": "radiusAndAngle", "description": "Which two inputs drive the bend (the third is derived)."},
    "radius": {"type": "string", "description": "Bend radius, e.g. \"5 mm\" (radiusAndAngle / radiusAndArcLength)."},
    "angle": {"type": "string", "description": "Bend angle, e.g. \"90 deg\" (radiusAndAngle / arcLengthAndAngle)."},
    "arcLength": {"type": "string", "description": "Bend arc length (radiusAndArcLength / arcLengthAndAngle)."},
    "flip": {"type": "boolean", "default": false, "description": "Fold toward the opposite side of the sketch plane."}
  },
  "required": ["sketchIndex"]
}`

func bendPartDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindBendPart, Summary: "Bend a solid body around a sketch bend line (radius/angle/arc-length controlled).", Schema: json.RawMessage(bendPartSchema), Apply: applyBendPart}
}

func applyBendPart(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.BendPart](s, raw)
	if err != nil {
		return nil, err
	}
	def, err := bendDefinition(part, in)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, feature.NewBendPartFeatures(part.Features()).Add(def))
}

// bendDefinition resolves the bend args (sketch, bend type, parametric scalars) into a model
// bend definition.
func bendDefinition(part *compdef.PartComponentDefinition, in featureargs.BendPart) (*feature.BendPartDefinition, error) {
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	bendType := types.RadiusAndAngleBend
	if in.BendType != "" {
		v, ok := types.ParseBendPartType(in.BendType)
		if !ok {
			return nil, fmt.Errorf("bendPart: unknown bendType %q", in.BendType)
		}
		bendType = v
	}
	radius, err := optionalLengthClosure(part, in.Radius, "bendPart: radius")
	if err != nil {
		return nil, err
	}
	angle, err := optionalAngleClosure(part, in.Angle, "bendPart: angle")
	if err != nil {
		return nil, err
	}
	arc, err := optionalLengthClosure(part, in.ArcLength, "bendPart: arcLength")
	if err != nil {
		return nil, err
	}
	return &feature.BendPartDefinition{
		Sketch: sk, LineIndex: in.LineIndex, BendType: bendType,
		Radius: radius, Angle: angle, ArcLength: arc, Flip: in.Flip,
	}, nil
}

// --- replace face ----------------------------------------------------------

const replaceFaceSchema = `{
  "type": "object",
  "properties": {
    "faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to replace (get_reference_keys)."},
    "newFaceRefs": {"type": "array", "items": {"type": "string"}, "description": "Replacement geometry: planar face keys and/or work planes (\"plane/N\", \"origin/plane/xy\"); faces may be from other bodies. Each picked face retrims onto its nearest new face."},
    "targetRef": {"type": "string", "description": "Legacy single same-body target face (associative). newFaceRefs takes precedence."}
  },
  "required": ["faceRefs"]
}`

func replaceFaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindReplaceFace, Summary: "Replace picked faces with new faces / a work plane (direct edit).", Schema: json.RawMessage(replaceFaceSchema), Apply: applyReplaceFace}
}

func applyReplaceFace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.ReplaceFace](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.FaceRefs) == 0 {
		return nil, errors.New("replaceFace: faceRefs is required")
	}
	if len(in.NewFaceRefs) > 0 {
		planes, err := replaceFacePlanes(part, in.NewFaceRefs)
		if err != nil {
			return nil, err
		}
		pf := feature.NewModifyFeatures(part.Features()).AddReplaceFacePlanes(refKeys(in.FaceRefs), planes)
		return recomputeResult(part, pf)
	}
	if in.TargetRef == "" {
		return nil, errors.New("replaceFace: newFaceRefs or targetRef is required")
	}
	pf := feature.NewModifyFeatures(part.Features()).AddReplaceFace(refKeys(in.FaceRefs), []byte(in.TargetRef))
	return recomputeResult(part, pf)
}

// replaceFacePlanes resolves each new-face reference (a planar face key or a work plane) to a
// frozen plane the replace-face feature retrims onto (#1886).
func replaceFacePlanes(part *compdef.PartComponentDefinition, refs []string) ([]geom.Plane, error) {
	planes := make([]geom.Plane, 0, len(refs))
	for _, ref := range refs {
		wp, err := part.WorkGeometry().PlaneTargetFromRef(ref)
		if err != nil {
			return nil, fmt.Errorf("replaceFace: new face %q: %w", ref, err)
		}
		pl, err := geom.NewPlane(wp.Plane().Origin(), wp.Plane().Normal().AsVector())
		if err != nil {
			return nil, fmt.Errorf("replaceFace: new face %q: %w", ref, err)
		}
		planes = append(planes, pl)
	}
	return planes, nil
}

// --- core/cavity -----------------------------------------------------------

const coreCavitySchema = `{
  "type": "object",
  "properties": {
    "axis": {"type": "string", "enum": ["z", "x", "y"], "default": "z", "description": "Draw direction the parting plane is perpendicular to."},
    "position": {"type": "string", "description": "Parting-plane position along the axis, e.g. \"10 mm\"."},
    "shrinkage": {"type": "number", "default": 0, "description": "Shrinkage compensation fraction, e.g. 0.02."}
  },
  "required": ["position"]
}`

func coreCavityDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindCoreCavity, Summary: "Split the body at a parting plane into core and cavity tooling.", Schema: json.RawMessage(coreCavitySchema), Apply: applyCoreCavity}
}

func applyCoreCavity(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.CoreCavity](s, raw)
	if err != nil {
		return nil, err
	}
	pos, err := lengthClosure(part, in.Position, "coreCavity: position")
	if err != nil {
		return nil, err
	}
	pf := feature.NewCoreCavityFeatures(part.Features()).AddByPartingPlaneFn(partingAxis(in.Axis), pos, in.Shrinkage)
	return recomputeResult(part, pf)
}

func partingAxis(name string) feature.PartingAxis {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "x":
		return feature.PartingX
	case "y":
		return feature.PartingY
	default:
		return feature.PartingZ
	}
}

// --- split solid -----------------------------------------------------------

const splitSolidSchema = `{
  "type": "object",
  "properties": {
    "workPlaneIndex": {"type": "integer", "minimum": 0, "description": "Index of the work plane to split along (see list_work_planes)."},
    "keep": {"type": "string", "enum": ["both", "positive", "negative"], "default": "both", "description": "Which side(s) of the plane to keep (trim modes)."},
    "type": {"type": "string", "enum": ["trimSolid", "splitFaces", "splitBody"], "description": "Split kind: trimSolid keeps one side (per keep, default positive); splitFaces imprints the plane onto the faces without removing material; splitBody keeps both pieces as separate solids. Absent: derived from keep."}
  },
  "required": ["workPlaneIndex"]
}`

func splitSolidDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindSplitSolid, Summary: "Split the solid along a work plane: trim one side, split into bodies, or split faces only.", Schema: json.RawMessage(splitSolidSchema), Apply: applySplitSolid}
}

func applySplitSolid(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.SplitSolid](s, raw)
	if err != nil {
		return nil, err
	}
	planes := part.WorkPlanes()
	if in.WorkPlaneIndex < 0 || in.WorkPlaneIndex >= planes.Count() {
		return nil, fmt.Errorf("splitSolid: work plane %d out of range (part has %d)", in.WorkPlaneIndex, planes.Count())
	}
	pf, err := addSplitOfType(part, planes.Item(in.WorkPlaneIndex), in)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, pf)
}

// addSplitOfType dispatches on the frozen split-type spelling; absent keeps the original
// keep-driven behavior (both sides by default).
func addSplitOfType(part *compdef.PartComponentDefinition, wp *feature.WorkPlane, in featureargs.SplitSolid) (*feature.PartFeature, error) {
	mods := feature.NewModifyFeatures(part.Features())
	if in.Type == "" {
		return mods.AddSplitSolid(wp, splitSide(in.Keep)), nil
	}
	st, ok := types.ParseSplitType(in.Type)
	if !ok {
		return nil, fmt.Errorf("splitSolid: unknown type %q (want trimSolid/splitFaces/splitBody)", in.Type)
	}
	switch st {
	case types.SplitFacesSplit:
		return mods.AddSplitFaces(wp), nil
	case types.SplitBodySplit:
		return mods.AddSplitSolid(wp, feature.SplitBoth), nil
	default: // trimSolid keeps one side; "both" makes no sense here, default positive
		side := splitSide(in.Keep)
		if side == feature.SplitBoth {
			side = feature.SplitPositive
		}
		return mods.AddSplitSolid(wp, side), nil
	}
}

func splitSide(name string) feature.SplitSide {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "positive":
		return feature.SplitPositive
	case "negative":
		return feature.SplitNegative
	default:
		return feature.SplitBoth
	}
}

// --- sketch-driven pattern -------------------------------------------------

const sketchDrivenSchema = `{
  "type": "object",
  "properties": {
    "sourceFeatures": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Names of the features to replicate (see model.tree)."},
    "points": {"type": "array", "minItems": 1, "items": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}, "description": "Placement points [[x,y,z],...] in cm, one occurrence per point."}
  },
  "required": ["sourceFeatures", "points"]
}`

func sketchDrivenDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindPatternSketchDriven, Summary: "Replicate features at a set of points (sketch-driven pattern).", Schema: json.RawMessage(sketchDrivenSchema), Apply: applySketchDriven}
}

func applySketchDriven(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.PatternSketchDriven](s, raw)
	if err != nil {
		return nil, err
	}
	ids, err := featureIDsByName(part, in.SourceFeatures)
	if err != nil {
		return nil, err
	}
	pts := make([]math.Point3, len(in.Points))
	for i, p := range in.Points {
		if pts[i], err = point3(p, "patternSketchDriven: points"); err != nil {
			return nil, err
		}
	}
	feature.NewPatternFeatures(part.Features()).AddSketchDriven(ids, func() []math.Point3 { return pts })
	return lastFeatureResult(part)
}
