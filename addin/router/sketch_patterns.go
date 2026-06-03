// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/param"
	"github.com/Oblikovati/oblikovati/model/sketch"
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
	step1, err := stepVector(part, in.Dir1, []float64{1, 0}, in.Spacing1)
	if err != nil {
		return nil, err
	}
	step2, err := stepVector(part, in.Dir2, []float64{0, 1}, in.Spacing2)
	if err != nil {
		return nil, err
	}
	return sk.RectangularPattern(ents, step1, in.Count1, step2, in.Count2)
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

// stepVector returns a grid step: the (defaulted, normalized) direction scaled by the
// unit-bearing spacing.
func stepVector(part *compdef.PartComponentDefinition, dir, fallback []float64, spacing string) (math.Vector2, error) {
	if len(dir) != 2 {
		dir = fallback
	}
	d := math.V2(math.Scalar(dir[0]), math.Scalar(dir[1]))
	if d.Length() == 0 {
		return math.Vector2{}, fmt.Errorf("sketch.addPattern: zero direction vector")
	}
	q, err := part.Units().Parse(spacing, param.Length)
	if err != nil {
		return math.Vector2{}, fmt.Errorf("sketch.addPattern: spacing %q: %w", spacing, err)
	}
	return d.Scale(float64(q.Value) / d.Length()), nil
}
