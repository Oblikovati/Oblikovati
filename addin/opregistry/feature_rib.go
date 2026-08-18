// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// The rib: an open sketch profile thickened into a supporting wall/web, plus the options that
// decide where that wall lands — which side of the profile it grows on, its draft, which end holds
// the nominal thickness, and whether the profile is extended onto the part (#1882).

const ribSchema = `{
  "type": "object",
  "properties": {
    "sketchIndex": {"type": "integer", "minimum": 0},
    "profileIndex": {"type": "integer", "minimum": 0, "default": 0},
    "thickness": {"type": "string", "description": "Rib wall thickness, e.g. \"2 mm\"."},
    "depth": {"type": "string", "description": "How far the rib grows toward the body, e.g. \"10 mm\" (sign picks the direction; omit with toNext)."},
    "toNext": {"type": "boolean", "default": false, "description": "Extend the wall until it fully lands on the existing material (the to-next rib)."},
    "operation": {"type": "string", "enum": ["new", "join", "surface"], "default": "join", "description": "Join the rib to existing material (default), a new body, or \"surface\" to build the rib walls as an open sheet — Inventor's kSurfaceOperation."},
    "thickenSide": {"type": "string", "enum": ["symmetric", "side1", "side2"], "default": "symmetric", "description": "Which side of the profile the wall grows on: symmetric (half each side), side1 (the path's left side walking it as drawn) or side2 (its right side)."},
    "draft": {"type": "string", "description": "Optional draft/taper angle on the wall, e.g. \"3 deg\" — it opens the wall toward the root, the end that lands on the part."},
    "thicknessPlane": {"type": "string", "enum": ["sketch", "root"], "default": "sketch", "description": "Which end honours the nominal thickness once draft tapers the wall: the profile's own plane (default) or the root that lands on the part. Without a draft the wall is prismatic, so this has no effect."},
    "extendProfile": {"type": "boolean", "default": false, "description": "Lengthen the profile's two ends along their end tangents until they reach existing material, so a wall sketched short of the part still lands on it. An end with nothing ahead of it stays put."}
  },
  "required": ["sketchIndex", "thickness"]
}`

func ribDescriptor() *OperationDescriptor {
	return &OperationDescriptor{Name: featureargs.KindRib, Summary: "Thicken an open sketch profile into a support rib.", Schema: json.RawMessage(ribSchema), Apply: applyRib}
}

func applyRib(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, in, err := decodeFeatureArgs[featureargs.Rib](s, raw)
	if err != nil {
		return nil, err
	}
	sk, err := sketchAt(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	def, err := ribDefinitionFor(part, sk, in)
	if err != nil {
		return nil, err
	}
	pf := feature.NewRibFeatures(part.Features()).AddDefinition(def)
	return recomputeResult(part, pf)
}

// ribDefinitionFor assembles the rib recipe: the wall's thickness and extent, then the options
// that shape its cross-section (#1882).
func ribDefinitionFor(part *compdef.PartComponentDefinition, sk *sketch.Sketch,
	in featureargs.Rib) (*feature.RibDefinition, error) {
	th, err := lengthClosure(part, in.Thickness, "rib: thickness")
	if err != nil {
		return nil, err
	}
	depth, err := ribDepthClosure(part, in)
	if err != nil {
		return nil, err
	}
	op, err := parseOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	def := &feature.RibDefinition{
		Sketch: sk, ProfileIndex: in.ProfileIndex,
		Thickness: th, Depth: depth, ToNext: in.ToNext, Operation: op,
		ExtendProfile: in.ExtendProfile,
	}
	return def, bindRibWallOptions(part, def, in)
}

// ribDepthClosure resolves the wall's finite extent. A to-next rib may omit the depth entirely
// (its sign then only picks the direction); anything else without one has no extent at all.
func ribDepthClosure(part *compdef.PartComponentDefinition, in featureargs.Rib) (func() float64, error) {
	if in.Depth != "" {
		return lengthClosure(part, in.Depth, "rib: depth")
	}
	if !in.ToNext {
		return nil, errors.New("rib: give depth or toNext")
	}
	return nil, nil
}

// bindRibWallOptions records the wall's cross-section options: which side the thickness lands on,
// the draft angle, and which end holds the nominal thickness (#1882).
func bindRibWallOptions(part *compdef.PartComponentDefinition, def *feature.RibDefinition,
	in featureargs.Rib) error {
	side, err := parseRibThickenSide(in.ThickenSide)
	if err != nil {
		return err
	}
	draft, err := optionalAngleClosure(part, in.Draft, "rib: draft")
	if err != nil {
		return err
	}
	atRoot, err := parseRibThicknessPlane(in.ThicknessPlane)
	if err != nil {
		return err
	}
	def.ThickenSide, def.Draft, def.HoldThicknessAtRoot = side, callOrZeroF(draft), atRoot
	return nil
}

// parseRibThickenSide maps the thicken-side spelling; the default is symmetric (#1882).
func parseRibThickenSide(spelling string) (feature.RibThickenSide, error) {
	switch strings.ToLower(strings.TrimSpace(spelling)) {
	case "", "symmetric":
		return feature.RibThickenSymmetric, nil
	case "side1":
		return feature.RibThickenSide1, nil
	case "side2":
		return feature.RibThickenSide2, nil
	default:
		return 0, fmt.Errorf("rib: unknown thickenSide %q (want symmetric, side1 or side2)", spelling)
	}
}

// parseRibThicknessPlane maps the thickness plane onto "hold at root"; the default is the sketch
// plane (Inventor's kRibThicknessAtSketchPlane, #1882).
func parseRibThicknessPlane(spelling string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(spelling)) {
	case "", "sketch":
		return false, nil
	case "root":
		return true, nil
	default:
		return false, fmt.Errorf("rib: unknown thicknessPlane %q (want sketch or root)", spelling)
	}
}
