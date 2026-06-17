// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The dress-up (subtractive/modifying) feature operations — fillet, chamfer, shell, draft.
// Each acts on an existing body's edges or faces, referenced by key (get_reference_keys), and
// follows the extrude descriptor shape: a JSON schema + an Apply that builds the feature and
// recomputes. They are how an MCP driver exercises the subtractive kernel end to end.

// edgeDressArgs is the shared shape of the edge-referencing operations (fillet, chamfer).
// EdgeSets is fillet-only: the edge-set form (#323), taking precedence over the flat
// edgeRefs+radius pair.
type edgeDressArgs struct {
	EdgeRefs        []string        `json:"edgeRefs"`
	Radius          string          `json:"radius,omitempty"`          // fillet
	Distance        string          `json:"distance,omitempty"`        // chamfer
	EdgeSets        []filletSetArgs `json:"edgeSets,omitempty"`        // fillet
	CornerType      string          `json:"cornerType,omitempty"`      // fillet shared-corner treatment (default miter)
	ChamferType     string          `json:"chamferType,omitempty"`     // chamfer mode (default distance)
	Distance2       string          `json:"distance2,omitempty"`       // chamfer twoDistances
	Angle           string          `json:"angle,omitempty"`           // chamfer distanceAndAngle
	ConcaveStrategy string          `json:"concaveStrategy,omitempty"` // chamfer concave-edge handling (default outward)
}

// filletSetArgs is one fillet edge set over the wire: constant (radius) or variable
// (startRadius+endRadius over exactly one edge).
type filletSetArgs struct {
	EdgeRefs    []string `json:"edgeRefs"`
	Radius      string   `json:"radius,omitempty"`
	StartRadius string   `json:"startRadius,omitempty"`
	EndRadius   string   `json:"endRadius,omitempty"`
}

const filletSchema = `{
  "type": "object",
  "properties": {
    "edgeRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the edges to round (from get_reference_keys). Flat form: one constant radius over these edges."},
    "radius": {"type": "string", "description": "Fillet radius with units, e.g. \"3 mm\" (flat form)."},
    "edgeSets": {"type": "array", "minItems": 1, "description": "Edge-set form (takes precedence over edgeRefs): any mix of constant and variable radius sets.", "items": {"type": "object", "properties": {
      "edgeRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1},
      "radius": {"type": "string", "description": "Constant radius for this set, e.g. \"3 mm\"."},
      "startRadius": {"type": "string", "description": "Variable set: radius at the edge's start vertex (the set holds exactly one edge)."},
      "endRadius": {"type": "string", "description": "Variable set: radius at the edge's end vertex."}
    }, "required": ["edgeRefs"]}},
    "cornerType": {"type": "string", "enum": ["miter", "setback", "round"], "default": "miter", "description": "How a vertex where two filleted edges meet (third edge sharp) is treated: miter (exact crease), round (fillets the third edge into a smooth sphere). setback is reserved."}
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
    "concaveStrategy": {"type": "string", "enum": ["outward", "inward"], "default": "outward", "description": "Concave (internal) edge handling: outward fills the inside corner with material (default), inward cuts a recessed relief groove. Convex edges ignore this."}
  },
  "required": ["edgeRefs", "distance"]
}`

func filletDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "fillet", Summary: "Round picked edges of a body by a radius.", Schema: json.RawMessage(filletSchema), Apply: applyFillet}
}

func chamferDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "chamfer", Summary: "Bevel picked edges of a body by a setback distance.", Schema: json.RawMessage(chamferSchema), Apply: applyChamfer}
}

func applyFillet(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in edgeDressArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	corner, err := filletCornerOf(in.CornerType)
	if err != nil {
		return nil, err
	}
	if len(in.EdgeSets) > 0 {
		return applyFilletSets(part, in.EdgeSets, corner)
	}
	if len(in.EdgeRefs) == 0 {
		return nil, errors.New("fillet: edgeRefs is empty (give edgeRefs+radius or edgeSets)")
	}
	r, err := lengthClosure(part, in.Radius, "fillet: radius")
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddFilletCorner(refKeys(in.EdgeRefs), r, corner)
	return recomputeResult(part, pf)
}

// applyFilletSets decodes the edge-set form and adds the fillet with the chosen corner treatment.
func applyFilletSets(part *compdef.PartComponentDefinition, args []filletSetArgs, corner types.FilletCornerType) (json.RawMessage, error) {
	sets, err := filletSetsFromArgs(part, args)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, feature.NewDressUpFeatures(part.Features()).AddFilletSetsCorner(sets, corner))
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
func filletSetsFromArgs(part *compdef.PartComponentDefinition, args []filletSetArgs) ([]feature.FilletEdgeSet, error) {
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
func filletSetRadii(part *compdef.PartComponentDefinition, a filletSetArgs, i int) (feature.FilletEdgeSet, error) {
	hasConst, hasVar := a.Radius != "", a.StartRadius != "" || a.EndRadius != ""
	if hasConst == hasVar {
		return feature.FilletEdgeSet{}, fmt.Errorf("fillet: edgeSets[%d] needs radius OR startRadius+endRadius (got radius=%q start=%q end=%q)", i, a.Radius, a.StartRadius, a.EndRadius)
	}
	if hasConst {
		r, err := lengthClosure(part, a.Radius, "fillet: radius")
		return feature.FilletEdgeSet{Radius: r}, err
	}
	r0, err := lengthClosure(part, a.StartRadius, "fillet: startRadius")
	if err != nil {
		return feature.FilletEdgeSet{}, err
	}
	r1, err := lengthClosure(part, a.EndRadius, "fillet: endRadius")
	return feature.FilletEdgeSet{StartRadius: r0, EndRadius: r1}, err
}

func applyChamfer(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in edgeDressArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	if len(in.EdgeRefs) == 0 {
		return nil, errors.New("chamfer: edgeRefs is empty")
	}
	pf, err := buildChamfer(part, in)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, pf)
}

// buildChamfer resolves the chamfer mode, setbacks, and concave-edge strategy from the args and
// adds the feature (not yet recomputed).
func buildChamfer(part *compdef.PartComponentDefinition, in edgeDressArgs) (*feature.PartFeature, error) {
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
	pf, err := applyChamferMode(part, refKeys(in.EdgeRefs), d, ct, in)
	if err != nil {
		return nil, err
	}
	pf.Definition().(*feature.ChamferFeature).Definition().ConcaveStrategy = cs
	return pf, nil
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

// applyChamferMode places the chamfer feature for the resolved mode, resolving the second
// input (distance2 or angle) for the asymmetric modes.
func applyChamferMode(part *compdef.PartComponentDefinition, keys [][]byte, d func() float64, ct types.ChamferType, in edgeDressArgs) (*feature.PartFeature, error) {
	du := feature.NewDressUpFeatures(part.Features())
	switch ct {
	case types.ChamferTwoDistances:
		d2, err := lengthClosure(part, in.Distance2, "chamfer: distance2")
		if err != nil {
			return nil, err
		}
		return du.AddChamferTwoDistances(keys, d, d2), nil
	case types.ChamferDistanceAndAngle:
		a, err := angleClosure(part, in.Angle, "chamfer: angle")
		if err != nil {
			return nil, err
		}
		return du.AddChamferDistanceAngle(keys, d, a), nil
	default:
		return du.AddChamfer(keys, d), nil
	}
}

// faceDressArgs is the shared shape of the face-referencing operations (shell, draft).
type faceDressArgs struct {
	FaceRefs  []string `json:"faceRefs"`
	Thickness string   `json:"thickness,omitempty"` // shell
	Angle     string   `json:"angle,omitempty"`     // draft
}

const shellSchema = `{
  "type": "object",
  "properties": {
    "faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to remove, hollowing the body (from get_reference_keys)."},
    "thickness": {"type": "string", "description": "Remaining wall thickness with units, e.g. \"1 mm\"."}
  },
  "required": ["faceRefs", "thickness"]
}`

const draftSchema = `{
  "type": "object",
  "properties": {
    "faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to draft (from get_reference_keys)."},
    "angle": {"type": "string", "description": "Draft angle with units, e.g. \"3 deg\"."}
  },
  "required": ["faceRefs", "angle"]
}`

func shellDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "shell", Summary: "Hollow a body to a wall thickness, removing the picked faces.", Schema: json.RawMessage(shellSchema), Apply: applyShell}
}

func draftDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "draft", Summary: "Taper picked faces by a draft angle.", Schema: json.RawMessage(draftSchema), Apply: applyDraft}
}

func applyShell(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in faceDressArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	if len(in.FaceRefs) == 0 {
		return nil, errors.New("shell: faceRefs is empty")
	}
	th, err := lengthClosure(part, in.Thickness, "shell: thickness")
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddShell(refKeys(in.FaceRefs), th)
	return recomputeResult(part, pf)
}

func applyDraft(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in faceDressArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	if len(in.FaceRefs) == 0 {
		return nil, errors.New("draft: faceRefs is empty")
	}
	a, err := angleClosure(part, in.Angle, "draft: angle")
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddDraft(refKeys(in.FaceRefs), a)
	return recomputeResult(part, pf)
}

// --- lip / groove (M20-F10) ------------------------------------------------

type lipArgs struct {
	EdgeRefs []string `json:"edgeRefs"`
	Width    string   `json:"width"`
	Height   string   `json:"height"`
	Groove   bool     `json:"groove,omitempty"`
}

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
	return &OperationDescriptor{Name: "lip", Summary: "Run a raised lip (or recessed groove) bead along picked edges.", Schema: json.RawMessage(lipSchema), Apply: applyLip}
}

func applyLip(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in lipArgs
	if err := json.Unmarshal(raw, &in); err != nil {
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
