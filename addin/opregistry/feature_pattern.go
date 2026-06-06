// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"

	"oblikovati/addin/modelaccess"
	"oblikovati/app"
	"oblikovati/math"
	"oblikovati/model/compdef"
	"oblikovati/model/feature"
)

// The pattern features replicate one or more existing features (referenced by name, as shown
// in model.tree) — rectangular grid, circular array, and mirror. The source features are
// resolved to their ids; the layout is given numerically (counts, steps, axis, plane normal).

type patternArgs struct {
	SourceFeatures []string  `json:"sourceFeatures"`
	CountX         int       `json:"countX,omitempty"`
	CountY         int       `json:"countY,omitempty"`
	StepX          []float64 `json:"stepX,omitempty"`
	StepY          []float64 `json:"stepY,omitempty"`
	Count          int       `json:"count,omitempty"`
	Angle          string    `json:"angle,omitempty"`
	AxisPoint      []float64 `json:"axisPoint,omitempty"`
	AxisDir        []float64 `json:"axisDir,omitempty"`
	Normal         []float64 `json:"normal,omitempty"`
	Origin         []float64 `json:"origin,omitempty"`
}

const rectPatternSchema = `{
  "type": "object",
  "properties": {
    "sourceFeatures": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Names of the features to replicate (see model.tree), e.g. [\"Extrusion1\"]."},
    "countX": {"type": "integer", "minimum": 1, "default": 2},
    "countY": {"type": "integer", "minimum": 1, "default": 1},
    "stepX": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Spacing vector for the first direction [x,y,z] in cm."},
    "stepY": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Spacing vector for the second direction [x,y,z] in cm."}
  },
  "required": ["sourceFeatures", "stepX"]
}`

const circPatternSchema = `{
  "type": "object",
  "properties": {
    "sourceFeatures": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Names of the features to replicate (see model.tree)."},
    "count": {"type": "integer", "minimum": 1, "default": 4},
    "angle": {"type": "string", "description": "Total sweep angle, e.g. \"360 deg\"."},
    "axisPoint": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "A point on the rotation axis [x,y,z] (default origin)."},
    "axisDir": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Rotation axis direction [x,y,z] (default +Z)."}
  },
  "required": ["sourceFeatures", "count", "angle"]
}`

const mirrorSchema = `{
  "type": "object",
  "properties": {
    "sourceFeatures": {"type": "array", "items": {"type": "string"}, "minItems": 1, "description": "Names of the features to mirror (see model.tree)."},
    "origin": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "A point on the mirror plane [x,y,z] (default origin)."},
    "normal": {"type": "array", "items": {"type": "number"}, "minItems": 3, "maxItems": 3, "description": "Mirror-plane normal [x,y,z], e.g. [1,0,0] for the YZ plane."}
  },
  "required": ["sourceFeatures", "normal"]
}`

func rectPatternDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "patternRectangular", Summary: "Replicate features on a rectangular grid.", Schema: json.RawMessage(rectPatternSchema), Apply: applyRectPattern}
}

func circPatternDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "patternCircular", Summary: "Replicate features in a circular array about an axis.", Schema: json.RawMessage(circPatternSchema), Apply: applyCircPattern}
}

func mirrorDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "mirror", Summary: "Mirror features across a plane.", Schema: json.RawMessage(mirrorSchema), Apply: applyMirror}
}

// decodePattern resolves the active part, decoded args, and the source-feature ids common to
// every pattern operation.
func decodePattern(s *app.Session, raw json.RawMessage) (*compdef.PartComponentDefinition, patternArgs, []feature.ID, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, patternArgs{}, nil, err
	}
	var in patternArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, patternArgs{}, nil, err
	}
	ids, err := featureIDsByName(part, in.SourceFeatures)
	if err != nil {
		return nil, patternArgs{}, nil, err
	}
	return part, in, ids, nil
}

func applyRectPattern(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, ids, err := decodePattern(s, raw)
	if err != nil {
		return nil, err
	}
	stepX, err := vec3(in.StepX, "patternRectangular: stepX")
	if err != nil {
		return nil, err
	}
	stepY := math.Vector3{}
	if len(in.StepY) == 3 {
		if stepY, err = vec3(in.StepY, "patternRectangular: stepY"); err != nil {
			return nil, err
		}
	}
	cx, cy := defaultInt(in.CountX, 2), defaultInt(in.CountY, 1)
	feature.NewPatternFeatures(part.Features()).AddRectangular(ids, constIntFn(cx), constIntFn(cy), stepX, stepY)
	return lastFeatureResult(part)
}

func applyCircPattern(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, ids, err := decodePattern(s, raw)
	if err != nil {
		return nil, err
	}
	angle, err := angleClosure(part, in.Angle, "patternCircular: angle")
	if err != nil {
		return nil, err
	}
	feature.NewPatternFeatures(part.Features()).AddCircular(ids, constIntFn(defaultInt(in.Count, 4)), angle, originOr(in.AxisPoint), zAxisOr(in.AxisDir))
	return lastFeatureResult(part)
}

func applyMirror(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, ids, err := decodePattern(s, raw)
	if err != nil {
		return nil, err
	}
	if len(in.Normal) != 3 {
		return nil, errors.New("mirror: normal needs 3 components [x,y,z]")
	}
	normal, _ := vec3(in.Normal, "mirror: normal")
	feature.NewPatternFeatures(part.Features()).AddMirror(ids, nil, originOr(in.Origin), normal)
	return lastFeatureResult(part)
}

func defaultInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func originOr(a []float64) math.Point3 {
	if len(a) == 3 {
		p, _ := point3(a, "point")
		return p
	}
	return math.P3(0, 0, 0)
}

func zAxisOr(a []float64) math.Vector3 {
	if len(a) == 3 {
		v, _ := vec3(a, "axis")
		return v
	}
	return math.V3(0, 0, 1)
}
