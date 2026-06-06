// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"

	"oblikovati/addin/modelaccess"
	"oblikovati/app"
	"oblikovati/model/compdef"
	"oblikovati/model/feature"
)

// The direct-edit / modify features: combine (boolean two bodies), thicken (a surface body),
// trim (by a plane), and the face direct edits (move/offset/delete/split). Body inputs are
// indices (model.tree body order); face inputs are reference keys (get_reference_keys).

// --- combine ---------------------------------------------------------------

type combineArgs struct {
	TargetIndex int    `json:"targetIndex"`
	ToolIndex   int    `json:"toolIndex"`
	Operation   string `json:"operation"`
}

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
	return &OperationDescriptor{Name: "combine", Summary: "Boolean two solid bodies (join/cut/intersect).", Schema: json.RawMessage(combineSchema), Apply: applyCombine}
}

func applyCombine(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in combineArgs
	if err := json.Unmarshal(raw, &in); err != nil {
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

type thickenArgs struct {
	Thickness string `json:"thickness"`
}

const thickenSchema = `{
  "type": "object",
  "properties": {"thickness": {"type": "string", "description": "Wall thickness to add to the surface body, e.g. \"2 mm\"."}},
  "required": ["thickness"]
}`

func thickenDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "thicken", Summary: "Thicken a surface body into a solid.", Schema: json.RawMessage(thickenSchema), Apply: applyThicken}
}

func applyThicken(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in thickenArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	th, err := lengthClosure(part, in.Thickness, "thicken: thickness")
	if err != nil {
		return nil, err
	}
	pf := feature.NewModifyFeatures(part.Features()).AddThickenFn(th)
	return recomputeResult(part, pf)
}

// --- trim ------------------------------------------------------------------

type trimArgs struct {
	Origin       []float64 `json:"origin"`
	Normal       []float64 `json:"normal"`
	KeepPositive bool      `json:"keepPositive,omitempty"`
}

const trimSchema = `{
  "type": "object",
  "properties": {
    "origin": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Cutting-plane origin [x,y,z] in cm."},
    "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Cutting-plane normal [x,y,z]."},
    "keepPositive": {"type": "boolean", "default": true, "description": "Keep the half on the +normal side."}
  },
  "required": ["origin", "normal"]
}`

func trimDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "trim", Summary: "Trim the body with a cutting plane, keeping one half.", Schema: json.RawMessage(trimSchema), Apply: applyTrim}
}

func applyTrim(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in trimArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	origin, err := point3(in.Origin, "trim: origin")
	if err != nil {
		return nil, err
	}
	normal, err := vec3(in.Normal, "trim: normal")
	if err != nil {
		return nil, err
	}
	pf := feature.NewTrimFeatures(part.Features()).AddByPlane(origin, normal, in.KeepPositive)
	return recomputeResult(part, pf)
}

// --- face direct edits (move / offset / delete / split) --------------------

type faceEditArgs struct {
	FaceRefs    []string  `json:"faceRefs"`
	Translation []float64 `json:"translation,omitempty"` // moveFace
	Distance    string    `json:"distance,omitempty"`    // faceOffset
}

const moveFaceSchema = `{
  "type": "object",
  "properties": {
    "faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to move (get_reference_keys)."},
    "translation": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Move vector [x,y,z] in cm."}
  },
  "required": ["faceRefs", "translation"]
}`

const faceOffsetSchema = `{
  "type": "object",
  "properties": {
    "faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to offset (get_reference_keys)."},
    "distance": {"type": "string", "description": "Offset distance with units, e.g. \"2 mm\"."}
  },
  "required": ["faceRefs", "distance"]
}`

const deleteFaceSchema = `{
  "type": "object",
  "properties": {"faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to delete (get_reference_keys)."}},
  "required": ["faceRefs"]
}`

const splitFaceSchema = `{
  "type": "object",
  "properties": {"faceRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the faces to split (get_reference_keys)."}},
  "required": ["faceRefs"]
}`

func moveFaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "moveFace", Summary: "Move picked faces by a vector (direct edit).", Schema: json.RawMessage(moveFaceSchema), Apply: applyMoveFace}
}

func faceOffsetDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "faceOffset", Summary: "Offset picked faces by a distance (direct edit).", Schema: json.RawMessage(faceOffsetSchema), Apply: applyFaceOffset}
}

func deleteFaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "deleteFace", Summary: "Delete picked faces, healing the body (direct edit).", Schema: json.RawMessage(deleteFaceSchema), Apply: applyDeleteFace}
}

func splitFaceDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "split", Summary: "Split picked faces along their intersections (direct edit).", Schema: json.RawMessage(splitFaceSchema), Apply: applySplitFace}
}

func applyMoveFace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFaceEdit(s, raw, "moveFace")
	if err != nil {
		return nil, err
	}
	t, err := vec3(in.Translation, "moveFace: translation")
	if err != nil {
		return nil, err
	}
	pf := feature.NewModifyFeatures(part.Features()).AddMoveFace(refKeys(in.FaceRefs), t)
	return recomputeResult(part, pf)
}

func applyFaceOffset(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFaceEdit(s, raw, "faceOffset")
	if err != nil {
		return nil, err
	}
	d, err := lengthClosure(part, in.Distance, "faceOffset: distance")
	if err != nil {
		return nil, err
	}
	pf := feature.NewModifyFeatures(part.Features()).AddFaceOffsetFn(refKeys(in.FaceRefs), d)
	return recomputeResult(part, pf)
}

func applyDeleteFace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFaceEdit(s, raw, "deleteFace")
	if err != nil {
		return nil, err
	}
	pf := feature.NewModifyFeatures(part.Features()).AddDeleteFace(refKeys(in.FaceRefs))
	return recomputeResult(part, pf)
}

func applySplitFace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFaceEdit(s, raw, "split")
	if err != nil {
		return nil, err
	}
	pf := feature.NewModifyFeatures(part.Features()).AddSplit(refKeys(in.FaceRefs))
	return recomputeResult(part, pf)
}

// decodeFaceEdit is the shared front of the face direct-edit operations: active part + decoded
// args with a non-empty faceRefs.
func decodeFaceEdit(s *app.Session, raw json.RawMessage, op string) (*compdef.PartComponentDefinition, faceEditArgs, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, faceEditArgs{}, err
	}
	var in faceEditArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, faceEditArgs{}, err
	}
	if len(in.FaceRefs) == 0 {
		return nil, faceEditArgs{}, errors.New(op + ": faceRefs is empty")
	}
	return part, in, nil
}
