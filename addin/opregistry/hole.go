// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"

	"oblikovati/addin/modelaccess"
	"oblikovati/app"
	"oblikovati/model/compdef"
	"oblikovati/model/feature"
)

// The hole operation — a subtractive drilled hole on a picked face, referenced by key
// (get_reference_keys). With a depth it is a blind drilled hole; without one it drills through
// all. Counterbore/countersink/tapped variants follow the same shape (HoleFeatures.Add*).

type holeArgs struct {
	FaceRef         string `json:"faceRef"`
	Type            string `json:"type,omitempty"` // drilled (default) | counterbore | countersink | tapped
	Diameter        string `json:"diameter"`
	Depth           string `json:"depth,omitempty"` // omit (drilled) ⇒ through-all
	CounterDiameter string `json:"counterDiameter,omitempty"`
	CounterDepth    string `json:"counterDepth,omitempty"`
	SinkDiameter    string `json:"sinkDiameter,omitempty"`
	IncludedAngle   string `json:"includedAngle,omitempty"`
	Designation     string `json:"designation,omitempty"`
}

const holeSchema = `{
  "type": "object",
  "properties": {
    "faceRef": {"type": "string", "description": "Reference key of the planar face to drill into (from get_reference_keys)."},
    "type": {"type": "string", "enum": ["drilled", "counterbore", "countersink", "tapped"], "default": "drilled", "description": "Hole style."},
    "diameter": {"type": "string", "description": "Hole diameter with units, e.g. \"5 mm\"."},
    "depth": {"type": "string", "description": "Blind hole depth, e.g. \"8 mm\". Omit (drilled only) for a through-all hole."},
    "counterDiameter": {"type": "string", "description": "Counterbore diameter (type=counterbore)."},
    "counterDepth": {"type": "string", "description": "Counterbore depth (type=counterbore)."},
    "sinkDiameter": {"type": "string", "description": "Countersink top diameter (type=countersink)."},
    "includedAngle": {"type": "string", "description": "Countersink included angle, e.g. \"90 deg\" (type=countersink)."},
    "designation": {"type": "string", "description": "Thread designation, e.g. \"M5x0.8\" (type=tapped)."}
  },
  "required": ["faceRef", "diameter"]
}`

func holeDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "hole", Summary: "Drill a hole into a picked face: drilled (blind/through), counterbore, countersink, or tapped.", Schema: json.RawMessage(holeSchema), Apply: applyHole}
}

func applyHole(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in holeArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	if in.FaceRef == "" {
		return nil, errors.New("hole: faceRef is empty")
	}
	dia, err := lengthClosure(part, in.Diameter, "hole: diameter")
	if err != nil {
		return nil, err
	}
	pf, err := buildHole(part, in, dia)
	if err != nil {
		return nil, err
	}
	return recomputeResult(part, pf)
}

// buildHole dispatches on the hole type, resolving the extra dimensions each variant needs.
//
//nolint:funlen // one-case-per-hole-type dispatch switch (drilled/counterbore/countersink/tapped); length is the dispatch, like the serialize codecs.
func buildHole(part *compdef.PartComponentDefinition, in holeArgs, dia func() float64) (*feature.PartFeature, error) {
	holes := feature.NewHoleFeatures(part.Features())
	key := []byte(in.FaceRef)
	switch in.Type {
	case "", "drilled":
		if in.Depth == "" {
			return holes.AddDrilledThrough(key, dia), nil
		}
		depth, err := lengthClosure(part, in.Depth, "hole: depth")
		if err != nil {
			return nil, err
		}
		return holes.AddDrilled(key, dia, depth), nil
	case "counterbore":
		depth, cdia, cdepth, err := threeLengths(part, in.Depth, in.CounterDiameter, in.CounterDepth, "hole counterbore")
		if err != nil {
			return nil, err
		}
		return holes.AddCounterbore(key, dia, depth, cdia, cdepth), nil
	case "countersink":
		depth, sdia, err := twoLengths(part, in.Depth, in.SinkDiameter, "hole countersink")
		if err != nil {
			return nil, err
		}
		angle, err := angleClosure(part, in.IncludedAngle, "hole: includedAngle")
		if err != nil {
			return nil, err
		}
		return holes.AddCountersink(key, dia, depth, sdia, angle), nil
	case "tapped":
		depth, err := lengthClosure(part, in.Depth, "hole: depth")
		if err != nil {
			return nil, err
		}
		if in.Designation == "" {
			return nil, errors.New("hole: tapped needs a designation, e.g. \"M5x0.8\"")
		}
		return holes.AddTapped(key, dia, depth, in.Designation), nil
	default:
		return nil, fmt.Errorf("hole: unknown type %q (want drilled|counterbore|countersink|tapped)", in.Type)
	}
}

func twoLengths(part *compdef.PartComponentDefinition, a, b, ctx string) (func() float64, func() float64, error) {
	av, err := lengthClosure(part, a, ctx+": depth")
	if err != nil {
		return nil, nil, err
	}
	bv, err := lengthClosure(part, b, ctx+": diameter")
	if err != nil {
		return nil, nil, err
	}
	return av, bv, nil
}

func threeLengths(part *compdef.PartComponentDefinition, a, b, c, ctx string) (func() float64, func() float64, func() float64, error) {
	av, bv, err := twoLengths(part, a, b, ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	cv, err := lengthClosure(part, c, ctx+": counterDepth")
	if err != nil {
		return nil, nil, nil, err
	}
	return av, bv, cv, nil
}
