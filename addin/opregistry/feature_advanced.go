// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// The remaining feature operations that need a custom resolver: sweep (a profile along a path
// sketch's chain), move body (a transform), replace face, core/cavity tooling, split solid by
// a work plane, and a sketch-driven pattern.

// --- sweep -----------------------------------------------------------------

type sweepArgs struct {
	SketchIndex     int    `json:"sketchIndex"`
	ProfileIndex    int    `json:"profileIndex"`
	PathSketchIndex int    `json:"pathSketchIndex"`
	PathIndex       int    `json:"pathIndex"`
	Twist           string `json:"twist,omitempty"`
	Operation       string `json:"operation,omitempty"`
	// The definition union (M08 PBI-094, #314), discriminated by the frozen
	// api/types sweep spellings:
	DefinitionType  string              `json:"definitionType,omitempty"` // path (default) | pathAndGuideRail | pathAndGuideSurface | pathAndSectionTwists | solid
	Orientation     string              `json:"orientation,omitempty"`    // normalToPath (default) | parallelToOriginalProfile | alignToVector
	AlignVector     []float64           `json:"alignVector,omitempty"`
	Taper           string              `json:"taper,omitempty"`
	TwistStations   []sweepTwistStation `json:"twistStations,omitempty"`
	RailSketchIndex int                 `json:"railSketchIndex,omitempty"`
	RailIndex       int                 `json:"railIndex,omitempty"`
	Scaling         string              `json:"scaling,omitempty"` // xy (default) | x | none
	GuideFaceKey    string              `json:"guideFaceKey,omitempty"`
	ToolBodyIndex   *int                `json:"toolBodyIndex,omitempty"`
}

type sweepTwistStation struct {
	T     float64 `json:"t"`
	Angle string  `json:"angle"`
}

const sweepSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0, "description": "Sketch holding the profile to sweep (omit for definitionType \"solid\")."},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "pathSketchIndex": {"type": "integer", "minimum": 0, "description": "Sketch holding the open path (rail) to sweep along."},
    "pathIndex": {"type": "integer", "minimum": 0, "default": 0, "description": "Which open path of the path sketch (see list_sketch_profiles / the sketch's chains)."},
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
    "operation": {"type": "string", "enum": ["new", "join", "cut"], "default": "new"}
  },
  "required": ["pathSketchIndex"]
}`

func sweepDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "sweep", Summary: "Sweep a profile along an open path (rail) into a solid.", Schema: json.RawMessage(sweepSchema), Apply: applySweep}
}

func applySweep(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in sweepArgs
	if err := json.Unmarshal(raw, &in); err != nil {
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
func sweepDefinitionFromArgs(part *compdef.PartComponentDefinition, in sweepArgs) (*feature.SweepDefinition, error) {
	// Validate the path once up front, then hand the feature a live provider that
	// re-derives it from the path sketch each recompute, so a parameter driving the
	// rail reshapes the sweep (a snapshot would freeze it at apply time).
	if _, err := pathFromSketch(part, in.PathSketchIndex, in.PathIndex); err != nil {
		return nil, err
	}
	def := &feature.SweepDefinition{
		ProfileIndex: in.ProfileIndex,
		Path: func() *sketch.Path3D {
			p, _ := pathFromSketch(part, in.PathSketchIndex, in.PathIndex)
			return p
		},
	}
	if err := sweepScalars(part, in, def); err != nil {
		return nil, err
	}
	return def, sweepVariantFields(part, in, def)
}

// sweepScalars resolves the profile sketch, twist, taper, orientation and
// operation (the fields every profile variant shares).
func sweepScalars(part *compdef.PartComponentDefinition, in sweepArgs, def *feature.SweepDefinition) error {
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

func sweepOrientation(in sweepArgs, def *feature.SweepDefinition) error {
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
func sweepVariantFields(part *compdef.PartComponentDefinition, in sweepArgs, def *feature.SweepDefinition) error {
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

func sweepScaling(in sweepArgs, def *feature.SweepDefinition) error {
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

func sweepStations(part *compdef.PartComponentDefinition, in sweepArgs, def *feature.SweepDefinition) error {
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

type moveBodyArgs struct {
	BodyIndex   int       `json:"bodyIndex"`
	Translation []float64 `json:"translation"`
}

const moveBodySchema = `{
  "type": "object",
  "properties": {
    "bodyIndex": {"type": "integer", "minimum": 0, "description": "Body to move (model.tree body order)."},
    "translation": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Move vector [x,y,z] in cm."}
  },
  "required": ["bodyIndex", "translation"]
}`

func moveBodyDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "moveBody", Summary: "Move a solid body by a translation.", Schema: json.RawMessage(moveBodySchema), Apply: applyMoveBody}
}

func applyMoveBody(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in moveBodyArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	t, err := vec3(in.Translation, "moveBody: translation")
	if err != nil {
		return nil, err
	}
	pf := feature.NewModifyFeatures(part.Features()).AddMove(in.BodyIndex, math.Translation4(t))
	return recomputeResult(part, pf)
}

// --- replace face ----------------------------------------------------------

type replaceFaceArgs struct {
	FaceRefs  []string `json:"faceRefs"`
	TargetRef string   `json:"targetRef"`
}

const replaceFaceSchema = `{
  "type": "object",
  "properties": {
    "faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to replace (get_reference_keys)."},
    "targetRef": {"type": "string", "description": "Reference key of the face whose surface replaces them."}
  },
  "required": ["faceRefs", "targetRef"]
}`

func replaceFaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "replaceFace", Summary: "Replace picked faces with another face's surface (direct edit).", Schema: json.RawMessage(replaceFaceSchema), Apply: applyReplaceFace}
}

func applyReplaceFace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in replaceFaceArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	if len(in.FaceRefs) == 0 || in.TargetRef == "" {
		return nil, errors.New("replaceFace: faceRefs and targetRef are required")
	}
	pf := feature.NewModifyFeatures(part.Features()).AddReplaceFace(refKeys(in.FaceRefs), []byte(in.TargetRef))
	return recomputeResult(part, pf)
}

// --- core/cavity -----------------------------------------------------------

type coreCavityArgs struct {
	Axis      string  `json:"axis,omitempty"`
	Position  string  `json:"position"`
	Shrinkage float64 `json:"shrinkage,omitempty"`
}

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
	return &OperationDescriptor{Name: "coreCavity", Summary: "Split the body at a parting plane into core and cavity tooling.", Schema: json.RawMessage(coreCavitySchema), Apply: applyCoreCavity}
}

func applyCoreCavity(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in coreCavityArgs
	if err := json.Unmarshal(raw, &in); err != nil {
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

type splitSolidArgs struct {
	WorkPlaneIndex int    `json:"workPlaneIndex"`
	Keep           string `json:"keep,omitempty"`
}

const splitSolidSchema = `{
  "type": "object",
  "properties": {
    "workPlaneIndex": {"type": "integer", "minimum": 0, "description": "Index of the work plane to split along (see list_work_planes)."},
    "keep": {"type": "string", "enum": ["both", "positive", "negative"], "default": "both", "description": "Which side(s) of the plane to keep."}
  },
  "required": ["workPlaneIndex"]
}`

func splitSolidDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "splitSolid", Summary: "Split the solid along a work plane, keeping one or both halves.", Schema: json.RawMessage(splitSolidSchema), Apply: applySplitSolid}
}

func applySplitSolid(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in splitSolidArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	planes := part.WorkPlanes()
	if in.WorkPlaneIndex < 0 || in.WorkPlaneIndex >= planes.Count() {
		return nil, fmt.Errorf("splitSolid: work plane %d out of range (part has %d)", in.WorkPlaneIndex, planes.Count())
	}
	pf := feature.NewModifyFeatures(part.Features()).AddSplitSolid(planes.Item(in.WorkPlaneIndex), splitSide(in.Keep))
	return recomputeResult(part, pf)
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

type sketchDrivenArgs struct {
	SourceFeatures []string    `json:"sourceFeatures"`
	Points         [][]float64 `json:"points"`
}

const sketchDrivenSchema = `{
  "type": "object",
  "properties": {
    "sourceFeatures": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Names of the features to replicate (see model.tree)."},
    "points": {"type": "array", "minItems": 1, "items": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3}, "description": "Placement points [[x,y,z],...] in cm, one occurrence per point."}
  },
  "required": ["sourceFeatures", "points"]
}`

func sketchDrivenDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "patternSketchDriven", Summary: "Replicate features at a set of points (sketch-driven pattern).", Schema: json.RawMessage(sketchDrivenSchema), Apply: applySketchDriven}
}

func applySketchDriven(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in sketchDrivenArgs
	if err := json.Unmarshal(raw, &in); err != nil {
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
