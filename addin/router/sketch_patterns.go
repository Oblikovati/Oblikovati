// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati/addin/modelaccess"
	"oblikovati/api/wire"
	"oblikovati/app"
	"oblikovati/math"
	"oblikovati/model/compdef"
	"oblikovati/model/sketch"
)

// addSketchPattern duplicates a selection in a rectangular grid or circular array.
func addSketchPattern(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddSketchPatternArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	ents, err := entityRefs(sk, in.Entities)
	if err != nil {
		return nil, err
	}
	created, err := buildPattern(part, sk, ents, in)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.AddSketchPatternResult{Created: entityIDs(created)})
}

// buildPattern dispatches the pattern kind to its model builder.
func buildPattern(part *compdef.PartComponentDefinition, sk *sketch.Sketch, ents []sketch.Entity, in wire.AddSketchPatternArgs) ([]sketch.Entity, error) {
	switch in.Kind {
	case "rectangular":
		return rectangularPattern(part, sk, ents, in)
	case "circular":
		return circularPattern(part, sk, ents, in)
	default:
		return nil, fmt.Errorf("sketch.addPattern: unknown kind %q (want rectangular|circular)", in.Kind)
	}
}

// rectangularPattern builds the grid step vectors (direction×spacing) and patterns.
func rectangularPattern(part *compdef.PartComponentDefinition, sk *sketch.Sketch, ents []sketch.Entity, in wire.AddSketchPatternArgs) ([]sketch.Entity, error) {
	step1, err := stepVectorClosure(part, in.Dir1, []float64{1, 0}, in.Spacing1)
	if err != nil {
		return nil, err
	}
	step2, err := stepVectorClosure(part, in.Dir2, []float64{0, 1}, in.Spacing2)
	if err != nil {
		return nil, err
	}
	return sk.RectangularPatternLive(ents, step1, in.Count1, step2, in.Count2)
}

// stepVectorClosure is stepVector returning a live provider: the (defaulted, normalized)
// direction scaled by a parameter-aware spacing, so editing the spacing parameter
// re-spaces the pattern on the next solve.
func stepVectorClosure(part *compdef.PartComponentDefinition, dir, fallback []float64, spacing string) (func() math.Vector2, error) {
	if len(dir) != 2 {
		dir = fallback
	}
	d := math.V2(math.Scalar(dir[0]), math.Scalar(dir[1]))
	if d.Length() == 0 {
		return nil, fmt.Errorf("sketch.addPattern: zero direction vector")
	}
	unit := d.Scale(1 / d.Length())
	spc, err := modelLengthClosure(part, spacing)
	if err != nil {
		return nil, fmt.Errorf("sketch.addPattern: spacing %q: %w", spacing, err)
	}
	return func() math.Vector2 { return unit.Scale(spc()) }, nil
}

// circularPattern resolves the center and total angle and patterns around it.
func circularPattern(part *compdef.PartComponentDefinition, sk *sketch.Sketch, ents []sketch.Entity, in wire.AddSketchPatternArgs) ([]sketch.Entity, error) {
	center, err := point2Of(in.Center, "center")
	if err != nil {
		return nil, err
	}
	angle, err := modelAngle(part, in.Angle)
	if err != nil {
		return nil, err
	}
	return sk.CircularPattern(ents, center, in.Count, angle)
}
