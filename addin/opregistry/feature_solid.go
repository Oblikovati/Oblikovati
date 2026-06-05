// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/feature"
)

// The additive sketch-profile solid features beyond extrude: revolve, rib, emboss, coil, and
// loft. Each consumes one or more closed/open sketch profiles (by sketchIndex/profileIndex)
// and follows the extrude descriptor shape, so add_feature can drive the whole additive set.

// --- revolve ---------------------------------------------------------------

type revolveArgs struct {
	SketchIndex  int    `json:"sketchIndex"`
	ProfileIndex int    `json:"profileIndex"`
	AxisRef      string `json:"axisRef,omitempty"`
	Angle        string `json:"angle"`
	Operation    string `json:"operation,omitempty"`
}

const revolveSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "axisRef": {"type": "string", "description": "Work-axis reference to revolve about, e.g. \"origin/axis/y\" (default). See get_reference_keys / list_work_planes."},
    "angle": {"type": "string", "description": "Revolve angle with units, e.g. \"360 deg\"."},
    "operation": {"type": "string", "enum": ["new", "join", "cut", "intersect"], "default": "new"}
  },
  "required": ["sketchIndex", "angle"]
}`

func revolveDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "revolve", Summary: "Revolve a closed sketch profile about an axis into a solid.", Schema: json.RawMessage(revolveSchema), Apply: applyRevolve}
}

func applyRevolve(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in revolveArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	axis, err := axisFromRef(part, in.AxisRef)
	if err != nil {
		return nil, err
	}
	angle, err := angleValue(part, in.Angle, "revolve: angle")
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	pf := feature.NewRevolveFeatures(part.Features()).Add(sk, in.ProfileIndex, axis, constFn(angle), op)
	return recomputeResult(part, pf)
}

// --- rib -------------------------------------------------------------------

type ribArgs struct {
	SketchIndex  int    `json:"sketchIndex"`
	ProfileIndex int    `json:"profileIndex"`
	Thickness    string `json:"thickness"`
	Depth        string `json:"depth"`
	Operation    string `json:"operation,omitempty"`
}

const ribSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "thickness": {"type": "string", "description": "Rib wall thickness, e.g. \"2 mm\"."},
    "depth": {"type": "string", "description": "How far the rib grows toward the body, e.g. \"10 mm\"."},
    "operation": {"type": "string", "enum": ["new", "join"], "default": "join"}
  },
  "required": ["sketchIndex", "thickness", "depth"]
}`

func ribDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "rib", Summary: "Thicken an open sketch profile into a support rib.", Schema: json.RawMessage(ribSchema), Apply: applyRib}
}

func applyRib(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in ribArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	th, err := lengthValue(part, in.Thickness, "rib: thickness")
	if err != nil {
		return nil, err
	}
	depth, err := lengthValue(part, in.Depth, "rib: depth")
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	pf := feature.NewRibFeatures(part.Features()).Add(sk, in.ProfileIndex, constFn(th), constFn(depth), op)
	return recomputeResult(part, pf)
}

// --- emboss ----------------------------------------------------------------

type embossArgs struct {
	SketchIndex    int    `json:"sketchIndex"`
	ProfileIndices []int  `json:"profileIndices,omitempty"`
	ProfileIndex   int    `json:"profileIndex"`
	Depth          string `json:"depth"`
	Engrave        bool   `json:"engrave,omitempty"`
}

const embossSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndices": {"type": "array", "items": {"type": "integer", "minimum": 0}, "description": "Profiles to emboss; omit to use profileIndex."},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "depth": {"type": "string", "description": "Raise (or, with engrave, cut) depth, e.g. \"1 mm\"."},
    "engrave": {"type": "boolean", "default": false, "description": "Cut into the face instead of raising from it."}
  },
  "required": ["sketchIndex", "depth"]
}`

func embossDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "emboss", Summary: "Raise or engrave a sketch profile on a face.", Schema: json.RawMessage(embossSchema), Apply: applyEmboss}
}

func applyEmboss(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in embossArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	depth, err := lengthValue(part, in.Depth, "emboss: depth")
	if err != nil {
		return nil, err
	}
	profiles := in.ProfileIndices
	if len(profiles) == 0 {
		profiles = []int{in.ProfileIndex}
	}
	pf := feature.NewEmbossFeatures(part.Features()).Add(sk, profiles, constFn(depth), in.Engrave, 0)
	return recomputeResult(part, pf)
}

// --- coil ------------------------------------------------------------------

type coilArgs struct {
	SketchIndex  int    `json:"sketchIndex"`
	ProfileIndex int    `json:"profileIndex"`
	AxisRef      string `json:"axisRef,omitempty"`
	Pitch        string `json:"pitch"`
	Revolutions  string `json:"revolutions"`
	Operation    string `json:"operation,omitempty"`
}

const coilSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "axisRef": {"type": "string", "description": "Work-axis reference to coil about, e.g. \"origin/axis/z\" (default)."},
    "pitch": {"type": "string", "description": "Axial rise per revolution, e.g. \"5 mm\"."},
    "revolutions": {"type": "string", "description": "Number of turns, e.g. \"4\"."},
    "operation": {"type": "string", "enum": ["new", "join", "cut"], "default": "new"}
  },
  "required": ["sketchIndex", "pitch", "revolutions"]
}`

func coilDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "coil", Summary: "Sweep a profile along a helix into a spring/thread.", Schema: json.RawMessage(coilSchema), Apply: applyCoil}
}

func applyCoil(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in coilArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	axisRef := in.AxisRef
	if axisRef == "" {
		axisRef = string(feature.OriginZAxis)
	}
	axis, err := axisFromRef(part, axisRef)
	if err != nil {
		return nil, err
	}
	pitch, err := lengthValue(part, in.Pitch, "coil: pitch")
	if err != nil {
		return nil, err
	}
	revs, err := strconv.ParseFloat(strings.TrimSpace(in.Revolutions), 64)
	if err != nil {
		return nil, fmt.Errorf("coil: revolutions %q must be a number: %w", in.Revolutions, err)
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	pf := feature.NewCoilFeatures(part.Features()).Add(sk, in.ProfileIndex, axis, constFn(pitch), constFn(revs), 0, op)
	return recomputeResult(part, pf)
}

// --- loft ------------------------------------------------------------------

type loftSectionRef struct {
	SketchIndex  int `json:"sketchIndex"`
	ProfileIndex int `json:"profileIndex"`
}

type loftArgs struct {
	Sections  []loftSectionRef `json:"sections"`
	Closed    bool             `json:"closed,omitempty"`
	Operation string           `json:"operation,omitempty"`
}

const loftSchema = `{
  "type": "object",
  "properties": {
    "sections": {"type": "array", "minItems": 2, "items": {"type": "object", "properties": {"sketchIndex": {"type": "integer"}, "profileIndex": {"type": "integer"}}, "required": ["sketchIndex"]}, "description": "Ordered cross-section profiles (>= 2) to loft through."},
    "closed": {"type": "boolean", "default": false},
    "operation": {"type": "string", "enum": ["new", "join", "cut"], "default": "new"}
  },
  "required": ["sections"]
}`

func loftDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: "loft", Summary: "Blend two or more profiles into a solid (loft).", Schema: json.RawMessage(loftSchema), Apply: applyLoft}
}

func applyLoft(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in loftArgs
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	if len(in.Sections) < 2 {
		return nil, errors.New("loft: needs >= 2 sections")
	}
	sections := make([]feature.LoftSection, len(in.Sections))
	for i, r := range in.Sections {
		sk, serr := sketchAt(part, r.SketchIndex)
		if serr != nil {
			return nil, serr
		}
		sections[i] = feature.LoftSection{Sketch: sk, ProfileIndex: r.ProfileIndex}
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	pf := feature.NewLoftFeatures(part.Features()).Add(sections, in.Closed, op)
	return recomputeResult(part, pf)
}
