// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The direct-edit / modify features: combine (boolean two bodies), thicken (a surface body),
// trim (by a plane), and the face direct edits (move/offset/delete/split). Body inputs are
// indices (model.tree body order); face inputs are reference keys (get_reference_keys).

// --- combine ---------------------------------------------------------------

const combineSchema = `{
  "type": "object",
  "properties": {
    "targetIndex": {"type": "integer", "minimum": 0, "description": "Body kept (index in model.tree body order)."},
    "toolIndex": {"type": "integer", "minimum": 0, "description": "Body combined into the target."},
    "operation": {"type": "string", "enum": ["join", "cut", "intersect"], "default": "join"}
  },
  "required": ["targetIndex", "toolIndex", "operation"]
}`

func combineDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindCombine, Summary: "Boolean two solid bodies (join/cut/intersect).", Schema: json.RawMessage(combineSchema), Apply: applyCombine}
}

func applyCombine(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Combine](s, raw)
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	pf := feature.NewModifyFeatures(part.Features()).AddCombine(in.TargetIndex, in.ToolIndex, op)
	return recomputeResult(part, pf)
}

// --- thicken ---------------------------------------------------------------

const thickenSchema = `{
  "type": "object",
  "properties": {
    "thickness": {"type": "string", "description": "Wall thickness to add to the surface body, e.g. \"2 mm\" (0 allowed for operation surface = a copy)."},
    "direction": {"type": "string", "enum": ["positive", "negative", "symmetric"], "default": "positive", "description": "Offset side: +normal, -normal, or half each way."},
    "operation": {"type": "string", "enum": ["join", "cut", "intersect", "surface"], "default": "join", "description": "join solidifies; cut/intersect boolean the solid into the running solid; surface offsets as a surface body."},
    "faceRefs": {"type": "array", "items": {"type": "string"}, "description": "Reference keys of a face subset to thicken (get_reference_keys); empty = whole body."},
    "createVerticalSurfaces": {"type": "boolean", "default": true, "description": "Close subset-boundary edges with side walls."},
    "automaticFaceChain": {"type": "boolean", "default": false, "description": "Accepted for parity; selection is explicit, so not applied."},
    "automaticBlending": {"type": "boolean", "default": false, "description": "Accepted for parity; not geometrically applied."},
    "approximation": {"type": "string", "enum": ["none", "mean", "neverTooThick", "neverTooThin"], "description": "Accepted approximation (#331 parity); the kernel computes the exact offset, which satisfies every bound."}
  },
  "required": ["thickness"]
}`

func thickenDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindThicken, Summary: "Thicken a surface body into a solid.", Schema: json.RawMessage(thickenSchema), Apply: applyThicken}
}

func applyThicken(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Thicken](s, raw)
	if err != nil {
		return nil, err
	}
	th, err := lengthClosure(part, in.Thickness, "thicken: thickness")
	if err != nil {
		return nil, err
	}
	approx, err := approximationArg(in.Approximation, "thicken")
	if err != nil {
		return nil, err
	}
	dir, ok := ops.ParseThickenDirection(in.Direction)
	if !ok {
		return nil, fmt.Errorf("thicken: unknown direction %q (want positive/negative/symmetric)", in.Direction)
	}
	op, asSurface, err := parseThickenOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	pf := feature.NewModifyFeatures(part.Features()).AddThickenFn(th)
	tf := pf.Definition().(*feature.ThickenFeature)
	tf.SetApproximation(approx)
	tf.SetThickenOptions(dir, op, asSurface, refKeys(in.FaceRefs), boolOrTrue(in.CreateVerticalSurfaces), in.AutomaticFaceChain, in.AutomaticBlending)
	return recomputeResult(part, pf)
}

// parseThickenOperation maps the thicken output mode to (operation, asSurface). "" defaults to
// join; "surface" is the offset-surface path; cut/intersect boolean into the running solid.
func parseThickenOperation(name string) (ops.PartFeatureOperation, bool, error) {
	switch name {
	case "", "join":
		return ops.Join, false, nil
	case "cut":
		return ops.Cut, false, nil
	case "intersect":
		return ops.Intersect, false, nil
	case "surface":
		return ops.Join, true, nil
	default:
		return ops.Join, false, fmt.Errorf("thicken: unknown operation %q (want join/cut/intersect/surface)", name)
	}
}

// boolOrTrue defaults an omitted *bool to true (Inventor's CreateVerticalSurfaces default).
func boolOrTrue(b *bool) bool { return b == nil || *b }

// approximationArg parses the optional #331 approximation spelling (empty = none/exact).
func approximationArg(s, op string) (types.FeatureApproximationType, error) {
	if s == "" {
		return 0, nil
	}
	a, ok := types.ParseFeatureApproximationType(s)
	if !ok {
		return 0, fmt.Errorf("%s: unknown approximation %q (want none/mean/neverTooThick/neverTooThin)", op, s)
	}
	return a, nil
}

// --- trim ------------------------------------------------------------------

const trimSchema = `{
  "type": "object",
  "properties": {
    "origin": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Explicit cutting-plane origin [x,y,z] in cm."},
    "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Explicit cutting-plane normal [x,y,z]."},
    "toolRef": {"type": "string", "description": "Cutting tool: a work plane or planar face (\"plane/N\", \"origin/plane/xy\", or a face key)."},
    "toolBodyIndex": {"type": "integer", "minimum": 0, "description": "Cutting tool: a planar surface body by index (model.tree order)."},
    "toolSketchIndex": {"type": "integer", "minimum": 0, "description": "Cutting tool: the sketch holding a straight cutting line (swept along the sketch normal)."},
    "toolLineIndex": {"type": "integer", "minimum": 0, "default": 0, "description": "Which line of toolSketchIndex is the cutting line."},
    "keepPositive": {"type": "boolean", "default": true, "description": "Keep the side on the +normal side of the cut."}
  }
}`

func trimDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindTrim, Summary: "Trim a surface with a cutting tool (plane / work plane / surface body / sketch line), keeping one side.", Schema: json.RawMessage(trimSchema), Apply: applyTrim}
}

func applyTrim(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Trim](s, raw)
	if err != nil {
		return nil, err
	}
	origin, normal, err := trimCuttingPlane(part, in)
	if err != nil {
		return nil, err
	}
	pf := feature.NewTrimFeatures(part.Features()).AddByPlane(origin, normal, in.KeepPositive)
	return recomputeResult(part, pf)
}

// trimCuttingPlane resolves the trim's cutting tool to a plane (origin+normal): an explicit plane,
// a work plane / planar face, a planar surface body, or a straight sketch line swept along its
// sketch normal (#1880).
func trimCuttingPlane(part *compdef.PartComponentDefinition, in featureargs.Trim) (math.Point3, math.Vector3, error) {
	switch {
	case in.ToolRef != "":
		wp, err := part.WorkGeometry().PlaneTargetFromRef(in.ToolRef)
		if err != nil {
			return math.Point3{}, math.Vector3{}, fmt.Errorf("trim: tool %q: %w", in.ToolRef, err)
		}
		return wp.Plane().Origin(), wp.Plane().Normal().AsVector(), nil
	case in.ToolBodyIndex != nil:
		return trimBodyPlane(part, *in.ToolBodyIndex)
	case in.ToolSketchIndex != nil:
		return trimSketchLinePlane(part, *in.ToolSketchIndex, in.ToolLineIndex)
	default:
		origin, err := point3(in.Origin, "trim: origin")
		if err != nil {
			return math.Point3{}, math.Vector3{}, err
		}
		normal, err := vec3(in.Normal, "trim: normal")
		return origin, normal, err
	}
}

// trimBodyPlane extracts the plane of a planar surface body used as a cutting tool.
func trimBodyPlane(part *compdef.PartComponentDefinition, index int) (math.Point3, math.Vector3, error) {
	bodies := part.Features().Result()
	if index < 0 || index >= len(bodies) {
		return math.Point3{}, math.Vector3{}, fmt.Errorf("trim: tool body index %d out of range (have %d)", index, len(bodies))
	}
	pl, ok := ops.BodyPlane(bodies[index])
	if !ok {
		return math.Point3{}, math.Vector3{}, fmt.Errorf("trim: tool body %d is not a planar surface", index)
	}
	return pl.Origin, pl.Normal(), nil // geom.Plane.Normal() is already a Vector3
}

// trimSketchLinePlane derives the cutting plane of a straight sketch line: the plane containing the
// line and the sketch normal (the line swept perpendicular to its sketch).
func trimSketchLinePlane(part *compdef.PartComponentDefinition, sketchIndex, lineIndex int) (math.Point3, math.Vector3, error) {
	sk, err := sketchAt(part, sketchIndex)
	if err != nil {
		return math.Point3{}, math.Vector3{}, err
	}
	if lineIndex < 0 || lineIndex >= sk.Lines().Count() {
		return math.Point3{}, math.Vector3{}, fmt.Errorf("trim: line index %d out of range (sketch has %d lines)", lineIndex, sk.Lines().Count())
	}
	origin, dir := sk.Lines().Item(lineIndex).Axis3D(sk.Plane())
	normal := dir.Cross(sk.Plane().Normal().AsVector())
	if normal.LengthSquared() < 1e-18 {
		return math.Point3{}, math.Vector3{}, fmt.Errorf("trim: cutting line is degenerate")
	}
	return origin, normal, nil
}

// --- face direct edits (move / offset / delete / split) --------------------

const moveFaceSchema = `{
  "type": "object",
  "properties": {
    "faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to move (get_reference_keys)."},
    "translation": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Translate mode: move vector [x,y,z] in cm."},
    "axisPoint": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Rotate mode: a point on the rotation axis [x,y,z] in cm."},
    "axisDir": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Rotate mode: the rotation axis direction [x,y,z]."},
    "angle": {"type": "string", "description": "Rotate mode: rotation angle with units, e.g. \"10 deg\"."}
  },
  "required": ["faceRefs"]
}`

const faceOffsetSchema = `{
  "type": "object",
  "properties": {
    "faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to offset (get_reference_keys)."},
    "distance": {"type": "string", "description": "Offset distance with units, e.g. \"2 mm\"."},
    "approximation": {"type": "string", "enum": ["none", "mean", "neverTooThick", "neverTooThin"], "description": "Accepted approximation (#331 parity); the kernel computes the exact offset, which satisfies every bound."}
  },
  "required": ["faceRefs", "distance"]
}`

const deleteFaceSchema = `{
  "type": "object",
  "properties": {
    "faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to delete (get_reference_keys). Faces on an internal void shell delete that void and restore mass."},
    "heal": {"type": "boolean", "default": false, "description": "Extend the neighbouring faces to close the opening. Default false (Inventor parity) leaves the body open (a surface)."}
  },
  "required": ["faceRefs"]
}`

const splitFaceSchema = `{
  "type": "object",
  "properties": {"faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to split (get_reference_keys)."}},
  "required": ["faceRefs"]
}`

func moveFaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindMoveFace, Summary: "Move picked faces by a vector (direct edit).", Schema: json.RawMessage(moveFaceSchema), Apply: applyMoveFace}
}

func faceOffsetDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindFaceOffset, Summary: "Offset picked faces by a distance (direct edit).", Schema: json.RawMessage(faceOffsetSchema), Apply: applyFaceOffset}
}

func deleteFaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindDeleteFace, Summary: "Delete picked faces (heal to close, or leave open); a void selection removes the void (direct edit).", Schema: json.RawMessage(deleteFaceSchema), Apply: applyDeleteFace}
}

func splitFaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindSplit, Summary: "Split picked faces along their intersections (direct edit).", Schema: json.RawMessage(splitFaceSchema), Apply: applySplitFace}
}

func applyMoveFace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.MoveFace](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.FaceRefs) == 0 {
		return nil, errors.New("moveFace: faceRefs is empty")
	}
	if in.Angle != "" || len(in.AxisDir) > 0 {
		return applyMoveFaceRotate(part, in)
	}
	t, err := vec3(in.Translation, "moveFace: translation")
	if err != nil {
		return nil, err
	}
	pf := feature.NewModifyFeatures(part.Features()).AddMoveFace(refKeys(in.FaceRefs), t)
	return recomputeResult(part, pf)
}

// applyMoveFaceRotate is the rotate arm (#331): axisPoint + axisDir + angle.
func applyMoveFaceRotate(part *compdef.PartComponentDefinition, in featureargs.MoveFace) (json.RawMessage, error) {
	p, err := point3(in.AxisPoint, "moveFace: axisPoint")
	if err != nil {
		return nil, err
	}
	dir, err := vec3(in.AxisDir, "moveFace: axisDir")
	if err != nil {
		return nil, err
	}
	angle, err := angleClosure(part, in.Angle, "moveFace: angle")
	if err != nil {
		return nil, err
	}
	pf := feature.NewModifyFeatures(part.Features()).AddMoveFaceRotate(refKeys(in.FaceRefs), p, dir, angle)
	return recomputeResult(part, pf)
}

func applyFaceOffset(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.FaceOffset](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.FaceRefs) == 0 {
		return nil, errors.New("faceOffset: faceRefs is empty")
	}
	d, err := lengthClosure(part, in.Distance, "faceOffset: distance")
	if err != nil {
		return nil, err
	}
	approx, err := approximationArg(in.Approximation, "faceOffset")
	if err != nil {
		return nil, err
	}
	pf := feature.NewModifyFeatures(part.Features()).AddFaceOffsetApprox(refKeys(in.FaceRefs), d, approx)
	return recomputeResult(part, pf)
}

func applyDeleteFace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.DeleteFace](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.FaceRefs) == 0 {
		return nil, errors.New("deleteFace: faceRefs is empty")
	}
	pf := feature.NewModifyFeatures(part.Features()).AddDeleteFace(refKeys(in.FaceRefs), in.Heal)
	return recomputeResult(part, pf)
}

func applySplitFace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Split](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.FaceRefs) == 0 {
		return nil, errors.New("split: faceRefs is empty")
	}
	pf := feature.NewModifyFeatures(part.Features()).AddSplit(refKeys(in.FaceRefs))
	return recomputeResult(part, pf)
}

// --- simplify & unwrap (M20-F13) -------------------------------------------

const simplifySchema = `{
  "type": "object",
  "properties": {
    "faceRefs": {"type": "array", "items": {"type": "string"}, "description": "Reference keys of faces to remove and heal (from get_reference_keys)."},
    "fillVoids": {"type": "boolean", "default": false, "description": "Also fill internal voids (cavities)."}
  }
}`

func simplifyDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindSimplify, Summary: "Reduce a model: remove and heal selected faces and/or fill internal voids.", Schema: json.RawMessage(simplifySchema), Apply: applySimplify}
}

func applySimplify(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Simplify](s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.FaceRefs) == 0 && !in.FillVoids {
		return nil, errors.New("simplify: nothing to do (faceRefs empty and fillVoids off)")
	}
	pf := feature.NewModifyFeatures(part.Features()).AddSimplify(refKeys(in.FaceRefs), in.FillVoids)
	return recomputeResult(part, pf)
}

const unwrapSchema = `{
  "type": "object",
  "properties": {
    "faceRef": {"type": "string", "description": "Reference key of the cylindrical face to flatten (from get_reference_keys)."}
  },
  "required": ["faceRef"]
}`

func unwrapDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindUnwrap, Summary: "Flatten a cylindrical face into a flat sheet patch (circumference × height).", Schema: json.RawMessage(unwrapSchema), Apply: applyUnwrap}
}

func applyUnwrap(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Unwrap](s, raw)
	if err != nil {
		return nil, err
	}
	if in.FaceRef == "" {
		return nil, errors.New("unwrap: faceRef is required")
	}
	pf := feature.NewModifyFeatures(part.Features()).AddUnwrap([]byte(in.FaceRef))
	return recomputeResult(part, pf)
}
