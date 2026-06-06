// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"

	"oblikovati/addin/modelaccess"
	"oblikovati/app"
	"oblikovati/math"
	"oblikovati/model/compdef"
	"oblikovati/model/feature"
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
	angle, err := angleClosure(part, in.Angle, "revolve: angle")
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	pf := feature.NewRevolveFeatures(part.Features()).Add(sk, in.ProfileIndex, axis, angle, op)
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
	th, err := lengthClosure(part, in.Thickness, "rib: thickness")
	if err != nil {
		return nil, err
	}
	depth, err := lengthClosure(part, in.Depth, "rib: depth")
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	pf := feature.NewRibFeatures(part.Features()).Add(sk, in.ProfileIndex, th, depth, op)
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
	depth, err := lengthClosure(part, in.Depth, "emboss: depth")
	if err != nil {
		return nil, err
	}
	profiles := in.ProfileIndices
	if len(profiles) == 0 {
		profiles = []int{in.ProfileIndex}
	}
	pf := feature.NewEmbossFeatures(part.Features()).Add(sk, profiles, depth, in.Engrave, 0)
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
	axis, err := coilAxis(part, in.AxisRef)
	if err != nil {
		return nil, err
	}
	pitch, revs, err := coilPitchRevs(part, in)
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	pf := feature.NewCoilFeatures(part.Features()).Add(sk, in.ProfileIndex, axis, pitch, revs, 0, op)
	return recomputeResult(part, pf)
}

// coilAxis resolves the coil's work axis, defaulting to the Z origin axis when no ref is given
// (a coil's natural axis), unlike the revolve default of Y.
func coilAxis(part *compdef.PartComponentDefinition, ref string) (*feature.WorkAxis, error) {
	if ref == "" {
		ref = string(feature.OriginZAxis)
	}
	return axisFromRef(part, ref)
}

// coilPitchRevs resolves a coil's pitch (a unit-bearing length) and revolution count (a plain
// number), naming the field on a parse error.
func coilPitchRevs(part *compdef.PartComponentDefinition, in coilArgs) (pitch, revs func() float64, err error) {
	if pitch, err = lengthClosure(part, in.Pitch, "coil: pitch"); err != nil {
		return nil, nil, err
	}
	if revs, err = numberClosure(part, in.Revolutions, "coil: revolutions"); err != nil {
		return nil, nil, err
	}
	return pitch, revs, nil
}

// --- loft ------------------------------------------------------------------

type loftSectionRef struct {
	SketchIndex  int       `json:"sketchIndex"`
	ProfileIndex int       `json:"profileIndex"`
	Point        []float64 `json:"point,omitempty"` // [x,y] on the sketch plane → an apex (point) section
}

// loftEndArgs is a loft end-section condition: how the surface leaves the first (or arrives at
// the last) section. "angle"/"direction" tilt the takeoff at angle to the section's sketch
// plane, weighted by impact, and curve a two-section loft; the default "free" is ruled.
type loftEndArgs struct {
	Condition string  `json:"condition,omitempty"`
	Angle     string  `json:"angle,omitempty"`
	Impact    float64 `json:"impact,omitempty"`
	Reversed  bool    `json:"reversed,omitempty"`
}

type loftArgs struct {
	Sections  []loftSectionRef `json:"sections"`
	Closed    bool             `json:"closed,omitempty"`
	Operation string           `json:"operation,omitempty"`
	First     *loftEndArgs     `json:"first,omitempty"`
	Last      *loftEndArgs     `json:"last,omitempty"`
}

const loftSchema = `{
  "type": "object",
  "properties": {
    "sections": {"type": "array", "minItems": 2, "items": {"type": "object", "properties": {"sketchIndex": {"type": "integer"}, "profileIndex": {"type": "integer"}, "point": {"type": "array", "items": {"type": "number"}, "minItems": 2, "maxItems": 2, "description": "[x,y] on the sketch plane → an apex (point) section so the loft tapers to a tip; only valid first or last."}}, "required": ["sketchIndex"]}, "description": "Ordered cross-section profiles (>= 2) to loft through."},
    "closed": {"type": "boolean", "default": false},
    "operation": {"type": "string", "enum": ["new", "join", "cut"], "default": "new"},
    "first": {"type": "object", "description": "Start-section condition (curves the loft).", "properties": {"condition": {"type": "string", "enum": ["free", "angle", "direction", "sharp", "tangent-to-plane"], "default": "free"}, "angle": {"type": "string", "description": "Takeoff angle to the section plane (angle/direction), e.g. \"45 deg\"."}, "impact": {"type": "number", "description": "Takeoff weight (default 1); larger curves more."}, "reversed": {"type": "boolean", "description": "Flip the takeoff through the section plane (undercut/dish)."}}},
    "last": {"type": "object", "description": "End-section condition (curves the loft).", "properties": {"condition": {"type": "string", "enum": ["free", "angle", "direction", "sharp", "tangent-to-plane"], "default": "free"}, "angle": {"type": "string", "description": "Takeoff angle to the section plane (angle/direction), e.g. \"45 deg\"."}, "impact": {"type": "number", "description": "Takeoff weight (default 1); larger curves more."}, "reversed": {"type": "boolean", "description": "Flip the takeoff through the section plane (undercut/dish)."}}}
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
		ls := feature.LoftSection{Sketch: sk, ProfileIndex: r.ProfileIndex}
		if len(r.Point) == 2 { // an apex section: [x,y] on the sketch plane
			mp := sk.Plane().ToModel(math.P2(r.Point[0], r.Point[1]))
			ls.Point = &mp
		}
		sections[i] = ls
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	first, err := loftEndProvider(part, in.First, "first")
	if err != nil {
		return nil, err
	}
	last, err := loftEndProvider(part, in.Last, "last")
	if err != nil {
		return nil, err
	}
	liveEnds := func() (feature.LoftEnd, feature.LoftEnd) { return first(), last() }
	pf := feature.NewLoftFeatures(part.Features()).AddConditionedLive(sections, in.Closed, op, liveEnds)
	return recomputeResult(part, pf)
}

// loftEndProvider turns an end-condition arg into a live LoftEnd provider (re-read each recompute
// so a parameter-driven angle reshapes the loft). A nil/free condition yields a Free end; the
// face/point conditions are not yet supported and are rejected with the offending value.
func loftEndProvider(part *compdef.PartComponentDefinition, a *loftEndArgs, which string) (func() feature.LoftEnd, error) {
	if a == nil || feature.LoftCondition(a.Condition).IsFree() {
		return func() feature.LoftEnd { return feature.LoftEnd{} }, nil
	}
	cond := feature.LoftCondition(a.Condition)
	impact, reversed := a.Impact, a.Reversed
	switch {
	case cond.CurvesViaAngle(): // angle/direction takeoff on a profile section
		angle, err := optionalAngleClosure(part, a.Angle, "loft: "+which+" angle")
		if err != nil {
			return nil, err
		}
		return func() feature.LoftEnd {
			return feature.LoftEnd{Condition: cond, Angle: angle(), Impact: impact, Reversed: reversed}
		}, nil
	case cond.IsPointCondition(): // sharp / tangent-to-plane on an apex section
		return func() feature.LoftEnd {
			return feature.LoftEnd{Condition: cond, Impact: impact, Reversed: reversed}
		}, nil
	default: // tangent / smooth need an adjacent face (not yet supported)
		return nil, fmt.Errorf("loft: %s condition %q is not yet supported (use \"free\", \"angle\", \"direction\", \"sharp\" or \"tangent-to-plane\")", which, a.Condition)
	}
}
