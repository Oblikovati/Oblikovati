// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/feature"
)

// The dress-up (subtractive/modifying) feature operations — fillet, chamfer, shell, draft.
// Each acts on an existing body's edges or faces, referenced by key (get_reference_keys), and
// follows the extrude descriptor shape: a JSON schema + an Apply that builds the feature and
// recomputes. They are how an MCP driver exercises the subtractive kernel end to end.

// edgeDressArgs is the shared shape of the edge-referencing operations (fillet, chamfer).
type edgeDressArgs struct {
	EdgeRefs []string `json:"edgeRefs"`
	Radius   string   `json:"radius,omitempty"`   // fillet
	Distance string   `json:"distance,omitempty"` // chamfer
}

const filletSchema = `{
  "type": "object",
  "properties": {
    "edgeRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the edges to round (from get_reference_keys)."},
    "radius": {"type": "string", "description": "Fillet radius with units, e.g. \"3 mm\"."}
  },
  "required": ["edgeRefs", "radius"]
}`

const chamferSchema = `{
  "type": "object",
  "properties": {
    "edgeRefs": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Reference keys of the edges to bevel (from get_reference_keys)."},
    "distance": {"type": "string", "description": "Chamfer setback with units, e.g. \"2 mm\"."}
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
	if len(in.EdgeRefs) == 0 {
		return nil, errors.New("fillet: edgeRefs is empty")
	}
	r, err := lengthValue(part, in.Radius, "fillet: radius")
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddFillet(refKeys(in.EdgeRefs), constFn(r))
	return recomputeResult(part, pf)
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
	d, err := lengthValue(part, in.Distance, "chamfer: distance")
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddChamfer(refKeys(in.EdgeRefs), constFn(d))
	return recomputeResult(part, pf)
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
	th, err := lengthValue(part, in.Thickness, "shell: thickness")
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddShell(refKeys(in.FaceRefs), constFn(th))
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
	a, err := angleValue(part, in.Angle, "draft: angle")
	if err != nil {
		return nil, err
	}
	pf := feature.NewDressUpFeatures(part.Features()).AddDraft(refKeys(in.FaceRefs), constFn(a))
	return recomputeResult(part, pf)
}
