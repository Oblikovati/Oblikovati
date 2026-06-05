// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/feature"
)

// The surfacing features: build surface bodies (boundary patch, ruled surface), modify them
// (surface offset, extend), and convert between surfaces and solids (thicken — see
// feature_modify.go, midSurface, stitch, sculpt). Profile-based surfaces take a sketch
// profile; the rest act on the part's existing surface/solid bodies.

// --- boundary patch --------------------------------------------------------

type patchArgs struct {
	SketchIndex  int    `json:"sketchIndex"`
	ProfileIndex int    `json:"profileIndex"`
	Condition    string `json:"condition,omitempty"`
}

const boundaryPatchSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "condition": {"type": "string", "enum": ["free", "tangent"], "default": "free", "description": "Edge continuity to neighbouring faces."}
  },
  "required": ["sketchIndex"]
}`

func boundaryPatchDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "boundaryPatch", Summary: "Fill a closed sketch loop with a surface patch.", Schema: json.RawMessage(boundaryPatchSchema), Apply: applyBoundaryPatch}
}

func applyBoundaryPatch(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in patchArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	cond := feature.PatchFree
	if strings.EqualFold(in.Condition, "tangent") {
		cond = feature.PatchTangent
	}
	pf := feature.NewBoundaryPatchFeatures(part.Features()).Add(sk, in.ProfileIndex, cond)
	return recomputeResult(part, pf)
}

// --- ruled surface ---------------------------------------------------------

type ruledArgs struct {
	SketchIndex  int    `json:"sketchIndex"`
	ProfileIndex int    `json:"profileIndex"`
	Type         string `json:"type,omitempty"`
	Distance     string `json:"distance"`
}

const ruledSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "type": {"type": "string", "enum": ["normal", "tangent", "perpendicular"], "default": "normal", "description": "Direction of the straight rulings."},
    "distance": {"type": "string", "description": "Ruling length, e.g. \"10 mm\"."}
  },
  "required": ["sketchIndex", "distance"]
}`

func ruledSurfaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "ruledSurface", Summary: "Sweep straight rulings off a profile into a surface.", Schema: json.RawMessage(ruledSchema), Apply: applyRuledSurface}
}

func applyRuledSurface(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in ruledArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	dist, err := lengthValue(part, in.Distance, "ruledSurface: distance")
	if err != nil {
		return nil, err
	}
	pf := feature.NewRuledSurfaceFeatures(part.Features()).AddByDistance(sk, in.ProfileIndex, ruledType(in.Type), constFn(dist))
	return recomputeResult(part, pf)
}

func ruledType(name string) feature.RuledSurfaceType {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "tangent":
		return feature.RuledTangent
	case "perpendicular":
		return feature.RuledPerpendicular
	default:
		return feature.RuledNormal
	}
}

// --- surface offset / extend / midSurface / stitch / sculpt ----------------

type surfaceDistArgs struct {
	Distance          string `json:"distance,omitempty"`
	EdgeRef           string `json:"edgeRef,omitempty"`
	MaxThickness      string `json:"maxThickness,omitempty"`
	Tolerance         string `json:"tolerance,omitempty"`
	MaintainAsSurface bool   `json:"maintainAsSurface,omitempty"`
	Operation         string `json:"operation,omitempty"`
}

const surfaceOffsetSchema = `{
  "type": "object",
  "properties": {"distance": {"type": "string", "description": "Offset distance for the surface bodies, e.g. \"2 mm\"."}},
  "required": ["distance"]
}`

const extendSchema = `{
  "type": "object",
  "properties": {
    "edgeRef": {"type": "string", "description": "Reference key of the surface edge to extend (get_reference_keys)."},
    "distance": {"type": "string", "description": "Extend distance, e.g. \"5 mm\"."}
  },
  "required": ["edgeRef", "distance"]
}`

const midSurfaceSchema = `{
  "type": "object",
  "properties": {"maxThickness": {"type": "string", "description": "Max wall thickness to pair into a mid-surface, e.g. \"3 mm\"."}},
  "required": ["maxThickness"]
}`

const stitchSchema = `{
  "type": "object",
  "properties": {
    "tolerance": {"type": "string", "description": "Stitch gap tolerance, e.g. \"0.1 mm\".", "default": "0.1 mm"},
    "maintainAsSurface": {"type": "boolean", "default": false, "description": "Keep a surface body instead of solidifying a closed quilt."}
  }
}`

const sculptSchema = `{
  "type": "object",
  "properties": {
    "operation": {"type": "string", "enum": ["new", "join", "cut"], "default": "new"},
    "tolerance": {"type": "string", "description": "Sculpt tolerance, e.g. \"0.1 mm\".", "default": "0.1 mm"}
  }
}`

func surfaceOffsetDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "surfaceOffset", Summary: "Offset the part's surface bodies by a distance.", Schema: json.RawMessage(surfaceOffsetSchema), Apply: applySurfaceOffset}
}

func extendDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "extend", Summary: "Extend a surface body past a picked edge.", Schema: json.RawMessage(extendSchema), Apply: applyExtend}
}

func midSurfaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "midSurface", Summary: "Build the mid-surface between thin-wall face pairs.", Schema: json.RawMessage(midSurfaceSchema), Apply: applyMidSurface}
}

func stitchDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "stitch", Summary: "Stitch surface bodies into a quilt (or solid).", Schema: json.RawMessage(stitchSchema), Apply: applyStitch}
}

func sculptDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "sculpt", Summary: "Combine surfaces and solids into a sculpted body.", Schema: json.RawMessage(sculptSchema), Apply: applySculpt}
}

func applySurfaceOffset(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeSurface(s, raw)
	if err != nil {
		return nil, err
	}
	d, err := lengthValue(part, in.Distance, "surfaceOffset: distance")
	if err != nil {
		return nil, err
	}
	pf := feature.NewSurfaceOffsetFeatures(part.Features()).AddByDistance(constFn(d))
	return recomputeResult(part, pf)
}

func applyExtend(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeSurface(s, raw)
	if err != nil {
		return nil, err
	}
	if in.EdgeRef == "" {
		return nil, errors.New("extend: edgeRef is empty")
	}
	d, err := lengthValue(part, in.Distance, "extend: distance")
	if err != nil {
		return nil, err
	}
	pf := feature.NewExtendFeatures(part.Features()).Add([]byte(in.EdgeRef), constFn(d))
	return recomputeResult(part, pf)
}

func applyMidSurface(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeSurface(s, raw)
	if err != nil {
		return nil, err
	}
	th, err := lengthValue(part, in.MaxThickness, "midSurface: maxThickness")
	if err != nil {
		return nil, err
	}
	pf := feature.NewMidSurfaceFeatures(part.Features()).AddByThickness(th)
	return recomputeResult(part, pf)
}

func applyStitch(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeSurface(s, raw)
	if err != nil {
		return nil, err
	}
	tol, err := toleranceValue(part, in.Tolerance)
	if err != nil {
		return nil, err
	}
	pf := feature.NewStitchFeatures(part.Features()).Add(tol, in.MaintainAsSurface)
	return recomputeResult(part, pf)
}

func applySculpt(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeSurface(s, raw)
	if err != nil {
		return nil, err
	}
	tol, err := toleranceValue(part, in.Tolerance)
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	pf := feature.NewSculptFeatures(part.Features()).Add(op, tol)
	return recomputeResult(part, pf)
}

// decodeSurface is the shared front of the surface operations (active part + decoded args).
func decodeSurface(s *app.Session, raw json.RawMessage) (part *compdef.PartComponentDefinition, in surfaceDistArgs, err error) {
	p, perr := modelaccess.ActivePart(s)
	if perr != nil {
		return nil, surfaceDistArgs{}, perr
	}
	if jerr := json.Unmarshal(raw, &in); jerr != nil {
		return nil, surfaceDistArgs{}, jerr
	}
	return p, in, nil
}

// toleranceValue parses a stitch/sculpt tolerance ("" ⇒ 0.1 mm = 0.01 cm).
func toleranceValue(part *compdef.PartComponentDefinition, expr string) (float64, error) {
	if strings.TrimSpace(expr) == "" {
		return 0.01, nil
	}
	v, err := lengthValue(part, expr, "tolerance")
	if err != nil {
		return 0, fmt.Errorf("invalid tolerance: %w", err)
	}
	return v, nil
}
