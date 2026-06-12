// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The consolidated direct-edit operation (M09-F04 PBI-108, #332): one descriptor
// discriminated by the frozen DirectEditOperationType spellings.

type directEditArgs struct {
	Operation   string    `json:"operation"`
	FaceRefs    []string  `json:"faceRefs,omitempty"`
	Translation []float64 `json:"translation,omitempty"`
	Direction   []float64 `json:"direction,omitempty"`
	Distance    string    `json:"distance,omitempty"`
	AxisPoint   []float64 `json:"axisPoint,omitempty"`
	AxisDir     []float64 `json:"axisDir,omitempty"`
	Angle       string    `json:"angle,omitempty"`
	Scale       float64   `json:"scale,omitempty"`
	Base        []float64 `json:"base,omitempty"`
}

const directEditSchema = `{
  "type": "object",
  "properties": {
    "operation": {"type": "string", "enum": ["move", "size", "rotate", "delete", "scale"], "description": "Direct-edit operation (frozen DirectEditOperationType spellings)."},
    "faceRefs": {"type": "array", "items": {"type": "string"}, "description": "Picked faces (move/size/rotate/delete; get_reference_keys)."},
    "translation": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "move: displacement [x,y,z] in cm."},
    "direction": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "size: push/pull direction [x,y,z]."},
    "distance": {"type": "string", "description": "size: push distance with units, e.g. \"3 mm\"."},
    "axisPoint": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "rotate: a point on the axis [x,y,z] in cm."},
    "axisDir": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "rotate: the axis direction."},
    "angle": {"type": "string", "description": "rotate: angle with units, e.g. \"10 deg\"."},
    "scale": {"type": "number", "exclusiveMinimum": 0, "description": "scale: uniform factor applied to the running body."},
    "base": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "scale: fixed base point [x,y,z] in cm (default origin)."}
  },
  "required": ["operation"]
}`

func directEditDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "directEdit", Summary: "Direct edit: move/size/rotate/delete picked faces, or scale the body uniformly.", Schema: json.RawMessage(directEditSchema), Apply: applyDirectEdit}
}

func applyDirectEdit(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in directEditArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	op, ok := types.ParseDirectEditOperationType(in.Operation)
	if !ok {
		return nil, fmt.Errorf("directEdit: unknown operation %q (want move/size/rotate/delete/scale)", in.Operation)
	}
	def, err := directEditDefinition(part, op, in)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, feature.NewModifyFeatures(part.Features()).AddDirectEdit(def))
}

// directEditDefinition builds the definition for the operation, validating the
// fields it needs.
func directEditDefinition(part *compdef.PartComponentDefinition, op types.DirectEditOperationType, in directEditArgs) (*feature.DirectEditDefinition, error) {
	def := &feature.DirectEditDefinition{Operation: op, FaceKeys: refKeys(in.FaceRefs)}
	if op != types.DirectEditScaleOperation && len(in.FaceRefs) == 0 {
		return nil, errors.New("directEdit: faceRefs is empty")
	}
	switch op {
	case types.DirectEditMoveOperation:
		t, err := vec3(in.Translation, "directEdit: translation")
		if err != nil {
			return nil, err
		}
		def.Translation = t
	case types.DirectEditSizeOperation:
		return directEditSize(part, def, in)
	case types.DirectEditRotateOperation:
		return directEditRotate(part, def, in)
	case types.DirectEditScaleOperation:
		if in.Scale <= 0 {
			return nil, fmt.Errorf("directEdit: scale %g must be > 0", in.Scale)
		}
		def.ScaleFactor = constScale(in.Scale)
		def.BasePoint = optionalPoint(in.Base)
	}
	return def, nil
}

func directEditSize(part *compdef.PartComponentDefinition, def *feature.DirectEditDefinition, in directEditArgs) (*feature.DirectEditDefinition, error) {
	dir, err := vec3(in.Direction, "directEdit: direction")
	if err != nil {
		return nil, err
	}
	dist, err := lengthClosure(part, in.Distance, "directEdit: distance")
	if err != nil {
		return nil, err
	}
	def.Direction, def.Distance = dir, dist
	return def, nil
}

func directEditRotate(part *compdef.PartComponentDefinition, def *feature.DirectEditDefinition, in directEditArgs) (*feature.DirectEditDefinition, error) {
	p, err := point3(in.AxisPoint, "directEdit: axisPoint")
	if err != nil {
		return nil, err
	}
	dir, err := vec3(in.AxisDir, "directEdit: axisDir")
	if err != nil {
		return nil, err
	}
	angle, err := angleClosure(part, in.Angle, "directEdit: angle")
	if err != nil {
		return nil, err
	}
	def.AxisPoint, def.AxisDir, def.Angle = p, dir, angle
	return def, nil
}

func constScale(v float64) func() float64 { return func() float64 { return v } }

// optionalPoint reads a 3-coord point, defaulting to the origin when absent.
func optionalPoint(s []float64) math.Point3 {
	if len(s) != 3 {
		return math.P3(0, 0, 0)
	}
	return math.P3(s[0], s[1], s[2])
}
