// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"fmt"
	"strings"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// Exposing the hole's placement rules over the JSON API (Oblikovati#1861). The model does the
// resolving; this file only turns the request's references into the rule, so a placement that
// cannot be built is a caller error naming the missing field rather than a hole that silently
// falls back to drilling at a face centroid.

// buildHolePlacement turns the request's placement fields into the model rule, or nil for the plain
// face placement the older args describe.
func buildHolePlacement(part *compdef.PartComponentDefinition, in featureargs.Hole) (feature.HolePlacement, error) {
	face := feature.HoleFaceRef{Key: []byte(in.FaceRef)}
	if in.PlacementFaceGeom != nil {
		ref, err := geomFaceRef(*in.PlacementFaceGeom)
		if err != nil {
			return nil, err
		}
		face.Geom = &ref
	}
	switch strings.TrimSpace(in.Placement) {
	case "":
		return nil, nil
	case "sketch":
		return sketchPlacement(part, in)
	case "concentric":
		return concentricPlacement(face, in)
	case "linear":
		return linearPlacement(part, face, in)
	case "point":
		return pointPlacement(part, in)
	default:
		return nil, fmt.Errorf("hole: unknown placement %q (want sketch, linear, concentric or point)", in.Placement)
	}
}

// sketchPlacement drills one bore per centre point of the named sketch.
func sketchPlacement(part *compdef.PartComponentDefinition, in featureargs.Hole) (feature.HolePlacement, error) {
	sk, err := sketchAt(part, in.PlacementSketchIndex)
	if err != nil {
		return nil, fmt.Errorf("hole: sketch placement: %w", err)
	}
	return feature.SketchHolePlacement{Sketch: sk, Flipped: in.PlacementFlipped}, nil
}

// concentricPlacement centres the bore on a circular edge's axis.
func concentricPlacement(face feature.HoleFaceRef, in featureargs.Hole) (feature.HolePlacement, error) {
	if strings.TrimSpace(in.ConcentricRef) == "" {
		return nil, errMissingHoleRef("concentric", "concentricRef", "a circular edge reference key")
	}
	return feature.ConcentricHolePlacement{Face: face, RefEdge: []byte(in.ConcentricRef)}, nil
}

// linearPlacement locates the bore by two offsets from two reference edges of the placement face.
func linearPlacement(part *compdef.PartComponentDefinition, face feature.HoleFaceRef,
	in featureargs.Hole) (feature.HolePlacement, error) {
	if strings.TrimSpace(in.Edge1Ref) == "" || strings.TrimSpace(in.Edge2Ref) == "" {
		return nil, errMissingHoleRef("linear", "edge1Ref and edge2Ref", "two reference edge keys on the placement face")
	}
	first, err := lengthClosure(part, in.Offset1, "hole: offset1")
	if err != nil {
		return nil, err
	}
	second, err := lengthClosure(part, in.Offset2, "hole: offset2")
	if err != nil {
		return nil, err
	}
	return feature.LinearHolePlacement{Face: face,
		Edge1: []byte(in.Edge1Ref), Edge2: []byte(in.Edge2Ref), Offset1: first, Offset2: second}, nil
}

// pointPlacement drills at a work point along a work axis — the placement that needs no face.
func pointPlacement(part *compdef.PartComponentDefinition, in featureargs.Hole) (feature.HolePlacement, error) {
	if strings.TrimSpace(in.PointRef) == "" || strings.TrimSpace(in.AxisRef) == "" {
		return nil, errMissingHoleRef("point", "pointRef and axisRef", "a work point and the work axis to drill along")
	}
	point, ok := part.WorkGeometry().WorkPointByRef(feature.ParseWorkRef(in.PointRef))
	if !ok {
		return nil, fmt.Errorf("hole: point placement references work point %q, which does not exist", in.PointRef)
	}
	axis, err := axisFromRef(part, in.AxisRef)
	if err != nil {
		return nil, err
	}
	return feature.PointHolePlacement{Point: point, Axis: axis, Flipped: in.PlacementFlipped}, nil
}

// errMissingHoleRef is the shared shape for "this placement needs a reference you did not give".
func errMissingHoleRef(placement, fields, want string) error {
	return fmt.Errorf("hole: the %q placement needs %s (%s)", placement, fields, want)
}
