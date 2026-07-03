// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// The freeform (T-spline) primitive features: a box, a plane, and a quadball, each a
// subdivision cage that becomes editable freeform geometry. They take only dimensions and a
// subdivision level — no sketch or body — so they are the simplest add_feature kinds to drive.

const freeformBoxSchema = `{
  "type": "object",
  "properties": {
    "sizeX": {"type": "string", "description": "Box X size, e.g. \"40 mm\"."},
    "sizeY": {"type": "string", "description": "Box Y size, e.g. \"30 mm\"."},
    "sizeZ": {"type": "string", "description": "Box Z size, e.g. \"20 mm\"."},
    "level": {"type": "integer", "minimum": 0, "default": 1, "description": "Subdivision level of the cage."}
  },
  "required": ["sizeX", "sizeY", "sizeZ"]
}`

const freeformPlaneSchema = `{
  "type": "object",
  "properties": {
    "sizeX": {"type": "string", "description": "Plane X size, e.g. \"40 mm\"."},
    "sizeY": {"type": "string", "description": "Plane Y size, e.g. \"30 mm\"."},
    "level": {"type": "integer", "minimum": 0, "default": 1}
  },
  "required": ["sizeX", "sizeY"]
}`

const freeformQuadBallSchema = `{
  "type": "object",
  "properties": {
    "radius": {"type": "string", "description": "Quadball radius, e.g. \"20 mm\"."},
    "level": {"type": "integer", "minimum": 0, "default": 1}
  },
  "required": ["radius"]
}`

func freeformBoxDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindFreeformBox, Summary: "Create a freeform (T-spline) box primitive.", Schema: json.RawMessage(freeformBoxSchema), Apply: applyFreeformBox}
}

func freeformPlaneDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindFreeformPlane, Summary: "Create a freeform (T-spline) plane primitive.", Schema: json.RawMessage(freeformPlaneSchema), Apply: applyFreeformPlane}
}

func freeformQuadBallDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindFreeformQuadBall, Summary: "Create a freeform (T-spline) quadball (sphere) primitive.", Schema: json.RawMessage(freeformQuadBallSchema), Apply: applyFreeformQuadBall}
}

func applyFreeformBox(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.FreeformBox](s, raw)
	if err != nil {
		return nil, err
	}
	sx, err := lengthValue(part, in.SizeX, "freeformBox: sizeX")
	if err != nil {
		return nil, err
	}
	sy, err := lengthValue(part, in.SizeY, "freeformBox: sizeY")
	if err != nil {
		return nil, err
	}
	sz, err := lengthValue(part, in.SizeZ, "freeformBox: sizeZ")
	if err != nil {
		return nil, err
	}
	pf := feature.NewFreeformFeatures(part.Features()).AddBox(sx, sy, sz, freeformLevel(in.Level))
	return recomputeResult(part, pf)
}

func applyFreeformPlane(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.FreeformPlane](s, raw)
	if err != nil {
		return nil, err
	}
	sx, err := lengthValue(part, in.SizeX, "freeformPlane: sizeX")
	if err != nil {
		return nil, err
	}
	sy, err := lengthValue(part, in.SizeY, "freeformPlane: sizeY")
	if err != nil {
		return nil, err
	}
	pf := feature.NewFreeformFeatures(part.Features()).AddPlane(sx, sy, freeformLevel(in.Level))
	return recomputeResult(part, pf)
}

func applyFreeformQuadBall(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.FreeformQuadBall](s, raw)
	if err != nil {
		return nil, err
	}
	r, err := lengthValue(part, in.Radius, "freeformQuadBall: radius")
	if err != nil {
		return nil, err
	}
	pf := feature.NewFreeformFeatures(part.Features()).AddQuadBall(r, freeformLevel(in.Level))
	return recomputeResult(part, pf)
}

func freeformLevel(level int) int {
	if level <= 0 {
		return 1
	}
	return level
}
