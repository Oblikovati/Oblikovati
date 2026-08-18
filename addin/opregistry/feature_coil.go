// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"fmt"
	"strings"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// The coil: a profile swept along a helix (a spring or a thread), or — as a flat spiral — along a
// rail that grows radially instead of axially (#1883). Two of pitch/revolutions/height fix the
// helix; the spring end conditions and the winding sense are options on top.

const coilSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "axisRef": {"type": "string", "description": "Work-axis reference to coil about, e.g. \"origin/axis/z\" (default)."},
    "pitch": {"type": "string", "description": "Axial rise per revolution, e.g. \"5 mm\". Give exactly two of pitch/revolutions/height."},
    "revolutions": {"type": "string", "description": "Number of turns, e.g. \"4\"."},
    "height": {"type": "string", "description": "Total axial rise, e.g. \"30 mm\" — combines with pitch OR revolutions."},
    "taper": {"type": "string", "description": "Optional taper angle, e.g. \"5 deg\" — the helix radius grows with height."},
    "operation": {"type": "string", "enum": ["new", "join", "cut", "surface"], "default": "new", "description": "Boolean against existing bodies, or \"surface\" to coil the profile into an open swept-surface (sheet) body — Inventor's kSurfaceOperation."},
    "startTransitionAngle": {"type": "string", "description": "Spring start-end transition sweep (pitch winds down to zero), e.g. \"90 deg\". Grounds/flattens the coil start."},
    "startFlatAngle": {"type": "string", "description": "Spring start-end flat sweep (zero pitch) after the transition, e.g. \"180 deg\"."},
    "endTransitionAngle": {"type": "string", "description": "Spring end transition sweep (pitch winds down to zero), e.g. \"90 deg\"."},
    "endFlatAngle": {"type": "string", "description": "Spring end flat sweep (zero pitch) after the transition, e.g. \"180 deg\"."},
    "handedness": {"type": "string", "enum": ["right", "left"], "default": "right", "description": "Winding sense: \"right\" follows the right-hand rule about the axis while the coil rises along it (the ordinary thread/spring sense); \"left\" mirrors it. Independent of which way the axis points."},
    "type": {"type": "string", "enum": ["helical", "spiral"], "default": "helical", "description": "Coil flavour. \"spiral\" sweeps a FLAT spiral with no axial rise (Inventor's kSpiralCoilExtent) where pitch is the RADIAL step per turn — a clock spring. A spiral takes pitch + revolutions; height and taper are refused, having nothing to describe or act on."}
  },
  "required": ["sketchIndex"]
}`

func coilDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindCoil, Summary: "Sweep a profile along a helix into a spring/thread.", Schema: json.RawMessage(coilSchema), Apply: applyCoil}
}

func applyCoil(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Coil](s, raw)
	if err != nil {
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
	def, err := coilDefinitionFor(part, sk, axis, in)
	if err != nil {
		return nil, err
	}
	pf := feature.NewCoilFeatures(part.Features()).AddDefinition(def)
	return recomputeResult(part, pf)
}

// coilDefinitionFor assembles the coil recipe: the rail's shape, taper and operation, then the
// spring end conditions and the winding/flavour options.
func coilDefinitionFor(part *compdef.PartComponentDefinition, sk *sketch.Sketch,
	axis *feature.WorkAxis, in featureargs.Coil) (*feature.CoilDefinition, error) {
	pitch, revs, height, err := coilShapeArgs(part, in)
	if err != nil {
		return nil, err
	}
	taper, err := optionalAngleClosure(part, in.Taper, "coil: taper")
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	def := &feature.CoilDefinition{
		Sketch: sk, ProfileIndex: in.ProfileIndex, Axis: axis,
		Pitch: pitch, Revolutions: revs, Height: height,
		Taper: callOrZeroF(taper), Operation: op,
	}
	return def, bindCoilEndsAndWinding(part, def, in)
}

// bindCoilEndsAndWinding records the spring end treatment, the winding sense and the coil
// flavour — the options layered on top of the rail (#1883).
func bindCoilEndsAndWinding(part *compdef.PartComponentDefinition, def *feature.CoilDefinition,
	in featureargs.Coil) error {
	startEnd, err := coilEndCondition(part, in.StartTransitionAngle, in.StartFlatAngle, "coil: start")
	if err != nil {
		return err
	}
	endEnd, err := coilEndCondition(part, in.EndTransitionAngle, in.EndFlatAngle, "coil: end")
	if err != nil {
		return err
	}
	hand, err := parseCoilHandedness(in.Handedness)
	if err != nil {
		return err
	}
	spiral, err := parseCoilSpiral(in.Type)
	if err != nil {
		return err
	}
	def.StartEnd, def.EndEnd, def.Handedness, def.Spiral = startEnd, endEnd, hand, spiral
	return nil
}

// parseCoilHandedness maps the winding-sense spelling; the default is right-handed (#1883).
func parseCoilHandedness(spelling string) (feature.CoilHandedness, error) {
	switch strings.ToLower(strings.TrimSpace(spelling)) {
	case "", "right":
		return feature.RightHandedCoil, nil
	case "left":
		return feature.LeftHandedCoil, nil
	default:
		return 0, fmt.Errorf("coil: unknown handedness %q (want right or left)", spelling)
	}
}

// parseCoilSpiral maps the coil flavour onto the flat-spiral flag (#1883).
func parseCoilSpiral(spelling string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(spelling)) {
	case "", "helical":
		return false, nil
	case "spiral":
		return true, nil
	default:
		return false, fmt.Errorf("coil: unknown type %q (want helical or spiral)", spelling)
	}
}

// coilEndCondition parses one spring end's transition + flat sweep angles into a CoilEndCondition
// (radians). It is active (Flat) only when at least one angle is given; the transition sweeps the
// pitch down to zero and the flat sweep then holds zero pitch (a ground spring end). #1883.
func coilEndCondition(part *compdef.PartComponentDefinition, transition, flat, ctx string) (feature.CoilEndCondition, error) {
	if transition == "" && flat == "" {
		return feature.CoilEndCondition{}, nil
	}
	tf, err := optionalAngleClosure(part, transition, ctx+": transitionAngle")
	if err != nil {
		return feature.CoilEndCondition{}, err
	}
	ff, err := optionalAngleClosure(part, flat, ctx+": flatAngle")
	if err != nil {
		return feature.CoilEndCondition{}, err
	}
	return feature.CoilEndCondition{Flat: true, TransitionAngle: callOrZeroF(tf), FlatAngle: callOrZeroF(ff)}, nil
}

// callOrZeroF evaluates an optional closure (nil ⇒ 0).
func callOrZeroF(f func() float64) float64 {
	if f == nil {
		return 0
	}
	return f()
}

// coilAxis resolves the coil's work axis, defaulting to the Z origin axis when no ref is given
// (a coil's natural axis), unlike the revolve default of Y.
func coilAxis(part *compdef.PartComponentDefinition, ref string) (*feature.WorkAxis, error) {
	if ref == "" {
		ref = string(feature.OriginZAxis)
	}
	return axisFromRef(part, ref)
}

// coilShapeArgs resolves the two-of-three coil shape spec (#316): pitch and
// height are unit-bearing lengths, revolutions a plain number; absent fields
// stay nil (the model validates the combination).
func coilShapeArgs(part *compdef.PartComponentDefinition, in featureargs.Coil) (pitch, revs, height func() float64, err error) {
	if in.Pitch != "" {
		if pitch, err = lengthClosure(part, in.Pitch, "coil: pitch"); err != nil {
			return nil, nil, nil, err
		}
	}
	if in.Revolutions != "" {
		if revs, err = numberClosure(part, in.Revolutions, "coil: revolutions"); err != nil {
			return nil, nil, nil, err
		}
	}
	if in.Height != "" {
		if height, err = lengthClosure(part, in.Height, "coil: height"); err != nil {
			return nil, nil, nil, err
		}
	}
	return pitch, revs, height, nil
}
