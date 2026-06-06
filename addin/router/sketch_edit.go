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

// transformSketch applies an edit operation (move/rotate/copy/mirror) to a selection.
func transformSketch(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.TransformSketchArgs
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
	created, err := applyTransform(part, sk, ents, in)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.TransformSketchResult{Created: entityIDs(created)})
}

// applyTransform dispatches the edit operation, returning any created entities.
func applyTransform(part *compdef.PartComponentDefinition, sk *sketch.Sketch, ents []sketch.Entity, in wire.TransformSketchArgs) ([]sketch.Entity, error) {
	switch in.Op {
	case "move":
		v, err := vector2Of(in.Vector, "vector")
		if err != nil {
			return nil, err
		}
		sk.MoveEntities(ents, v)
		return nil, nil
	case "copy":
		v, err := vector2Of(in.Vector, "vector")
		if err != nil {
			return nil, err
		}
		return sk.CopyEntities(ents, v), nil
	case "rotate":
		return nil, rotateOp(part, sk, ents, in)
	case "mirror":
		return mirrorOp(sk, ents, in.MirrorLine)
	case "trim", "split", "extend":
		return curveEditOp(sk, ents, in)
	default:
		return nil, fmt.Errorf("sketch.transform: unknown op %q (want move|copy|rotate|mirror|trim|split|extend)", in.Op)
	}
}

// curveEditOp dispatches the single-curve edit ops (trim/split/extend), which act on the
// first selected line at a pick point (Vector).
func curveEditOp(sk *sketch.Sketch, ents []sketch.Entity, in wire.TransformSketchArgs) ([]sketch.Entity, error) {
	pick, err := vector2AsPoint(in.Vector)
	if err != nil {
		return nil, err
	}
	// Trim accepts any curve target (line/circle/arc); split/extend act on lines.
	if in.Op == "trim" {
		switch e := ents[0].(type) {
		case *sketch.Line:
			return sk.TrimLine(e, pick)
		case *sketch.Circle:
			return sk.TrimCircle(e, pick)
		case *sketch.Arc:
			return sk.TrimArc(e, pick)
		default:
			return nil, fmt.Errorf("sketch.transform: trim unsupported target %T", ents[0])
		}
	}
	l, ok := ents[0].(*sketch.Line)
	if !ok {
		return nil, fmt.Errorf("sketch.transform: %s currently supports lines only (got %T)", in.Op, ents[0])
	}
	if in.Op == "split" {
		return sk.SplitLine(l, pick)
	}
	_, err = sk.ExtendLine(l, pickNearerEnd(l, pick)) // extend
	return nil, err
}

// vector2AsPoint reads the [x,y] pick point from the transform's Vector field.
func vector2AsPoint(v []float64) (math.Point2, error) {
	if len(v) != 2 {
		return math.Point2{}, fmt.Errorf("sketch.transform: pick point needs 2 components, got %d", len(v))
	}
	return math.P2(math.Scalar(v[0]), math.Scalar(v[1])), nil
}

// pickNearerEnd reports whether pick is closer to the line's B end (true) than its A end.
func pickNearerEnd(l *sketch.Line, pick math.Point2) bool {
	return pick.DistanceTo(l.EndPoint().Position()) < pick.DistanceTo(l.StartPoint().Position())
}

// rotateOp rotates the selection about a center by a unit-bearing angle.
func rotateOp(part *compdef.PartComponentDefinition, sk *sketch.Sketch, ents []sketch.Entity, in wire.TransformSketchArgs) error {
	c, err := point2Of(in.Center, "center")
	if err != nil {
		return err
	}
	angle, err := modelAngle(part, in.Angle)
	if err != nil {
		return err
	}
	sk.RotateEntities(ents, c, angle)
	return nil
}

// mirrorOp reflects the selection across the referenced mirror line.
func mirrorOp(sk *sketch.Sketch, ents []sketch.Entity, lineRefID uint64) ([]sketch.Entity, error) {
	line, err := lineRef(sk, lineRefID)
	if err != nil {
		return nil, err
	}
	return sk.MirrorEntities(ents, line), nil
}

// entityRefs resolves a selection of entity ids to entities.
func entityRefs(sk *sketch.Sketch, refs []uint64) ([]sketch.Entity, error) {
	if len(refs) == 0 {
		return nil, fmt.Errorf("sketch.transform: empty selection")
	}
	out := make([]sketch.Entity, len(refs))
	for i, id := range refs {
		e, ok := sk.EntityByID(sketch.ID(id))
		if !ok {
			return nil, fmt.Errorf("sketch.transform: no entity with id %d", id)
		}
		out[i] = e
	}
	return out, nil
}

// entityIDs collects entity ids.
func entityIDs(ents []sketch.Entity) []uint64 {
	out := make([]uint64, len(ents))
	for i, e := range ents {
		out[i] = uint64(e.EntityID())
	}
	return out
}

// vector2Of requires a 2-component [dx,dy] slice.
func vector2Of(s []float64, what string) (math.Vector2, error) {
	if len(s) != 2 {
		return math.Vector2{}, fmt.Errorf("sketch.transform: %s needs 2 components, got %d", what, len(s))
	}
	return math.V2(math.Scalar(s[0]), math.Scalar(s[1])), nil
}

// point2Of requires a 2-component [x,y] slice.
func point2Of(s []float64, what string) (math.Point2, error) {
	if len(s) != 2 {
		return math.Point2{}, fmt.Errorf("sketch.transform: %s needs 2 components, got %d", what, len(s))
	}
	return math.P2(math.Scalar(s[0]), math.Scalar(s[1])), nil
}
