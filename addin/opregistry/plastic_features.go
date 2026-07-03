// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/feature"
)

// Plastic-part features (M20-F10, #486): the cantilever snap-fit connector. Reachable over MCP via
// features.add like the other opregistry operations; the geometry + validation live in model/feature.

const snapFitSchema = `{
  "type": "object",
  "properties": {
    "length": {"type": "string", "description": "Cantilever arm length (along the beam), e.g. \"20 mm\"."},
    "width": {"type": "string", "description": "Arm width, e.g. \"6 mm\"."},
    "thickness": {"type": "string", "description": "Arm thickness, e.g. \"2 mm\"."},
    "catchLength": {"type": "string", "description": "Catch-lip length along the free end (must be <= length), e.g. \"3 mm\"."},
    "catchHeight": {"type": "string", "description": "Catch-lip height above the arm, e.g. \"1.5 mm\"."}
  },
  "required": ["length", "width", "thickness", "catchLength", "catchHeight"]
}`

func snapFitDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindSnapFit,
		Summary: "Add a cantilever snap-fit hook (a beam with a catch lip) to the part.",
		Schema:  json.RawMessage(snapFitSchema),
		Apply:   applySnapFit,
	}
}

const restSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndices": {"type": "array", "items": {"type": "integer", "minimum": 0}, "description": "Closed profiles bounding the rest; omit to use profileIndex."},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "depth": {"type": "string", "description": "Raise (or, when recessed, cut) depth, e.g. \"2 mm\"."},
    "recessed": {"type": "boolean", "default": false, "description": "Recess a pocket into the face instead of raising a pad."}
  },
  "required": ["sketchIndex", "depth"]
}`

func restDescriptor() *OperationDescriptor {
	return &OperationDescriptor{
		Name:    featureargs.KindRest,
		Summary: "Add a raised or recessed rest pad bounded by a closed sketch profile.",
		Schema:  json.RawMessage(restSchema),
		Apply:   applyRest,
	}
}

func applyRest(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in featureargs.Rest
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	depth, err := lengthClosure(part, in.Depth, "rest: depth")
	if err != nil {
		return nil, err
	}
	profiles := in.ProfileIndices
	if len(profiles) == 0 {
		profiles = []int{in.ProfileIndex}
	}
	pf := feature.NewPlasticFeatures(part.Features()).AddRest(sk, profiles, depth, in.Recessed, 0)
	return recomputeResult(part, pf)
}

func applySnapFit(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in featureargs.SnapFit
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	var l, w, t, cl, ch func() float64
	for _, d := range []struct {
		spelling, label string
		out             *func() float64
	}{
		{in.Length, "snapFit: length", &l},
		{in.Width, "snapFit: width", &w},
		{in.Thickness, "snapFit: thickness", &t},
		{in.CatchLength, "snapFit: catchLength", &cl},
		{in.CatchHeight, "snapFit: catchHeight", &ch},
	} {
		fn, err := lengthClosure(part, d.spelling, d.label)
		if err != nil {
			return nil, err
		}
		*d.out = fn
	}
	pf := feature.NewPlasticFeatures(part.Features()).AddCantileverSnapFit(l, w, t, cl, ch)
	return recomputeResult(part, pf)
}
