// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati/api/wire"

	"oblikovati/addin/modelaccess"
	"oblikovati/app"
	"oblikovati/math"
	"oblikovati/model/compdef"
	"oblikovati/model/param"
	"oblikovati/model/sketch"
)

// transformSketch3D applies a move/copy/rotate/delete edit to a 3D-sketch selection.
func transformSketch3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.Transform3DArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	ents, err := resolveEntities3D(sk, in.Entities)
	if err != nil {
		return nil, err
	}
	return applyTransform3D(part, sk, ents, in)
}

// applyTransform3D dispatches the requested edit op and returns the result.
func applyTransform3D(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, ents []sketch.Entity, in wire.Transform3DArgs) (json.RawMessage, error) {
	switch in.Op {
	case "move", "copy":
		return moveOrCopy3D(sk, ents, in)
	case "rotate":
		if err := rotate3D(part, sk, ents, in); err != nil {
			return nil, err
		}
	case "delete":
		for _, e := range ents {
			sk.DeleteEntity3D(e)
		}
	default:
		return nil, fmt.Errorf("sketch3d.transform: unknown op %q (want move|copy|rotate|delete)", in.Op)
	}
	return transform3DResult(sk, nil)
}

// moveOrCopy3D applies a translation (move) or a translated duplication (copy).
func moveOrCopy3D(sk *sketch.Sketch3D, ents []sketch.Entity, in wire.Transform3DArgs) (json.RawMessage, error) {
	v, err := vector3Arg(in.Vector)
	if err != nil {
		return nil, err
	}
	if in.Op == "copy" {
		return transform3DResult(sk, sk.CopyEntities3D(ents, v))
	}
	sk.MoveEntities3D(ents, v)
	return transform3DResult(sk, nil)
}

// rotate3D rotates the selection about the requested axis by a unit-bearing angle.
func rotate3D(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, ents []sketch.Entity, in wire.Transform3DArgs) error {
	center, err := point3Arg(in.Center)
	if err != nil {
		return err
	}
	axis, err := axisOrZ(in.Axis)
	if err != nil {
		return err
	}
	a, err := part.Units().Parse(in.Angle, param.Angle)
	if err != nil {
		return fmt.Errorf("sketch3d.transform: angle %q: %w", in.Angle, err)
	}
	sk.RotateEntities3D(ents, center, axis, float64(a.Value))
	return nil
}

// transform3DResult marshals the created ids (copy) and the resulting entity count.
func transform3DResult(sk *sketch.Sketch3D, created []sketch.Entity) (json.RawMessage, error) {
	ids := make([]uint64, len(created))
	for i, e := range created {
		ids[i] = uint64(e.EntityID())
	}
	return json.Marshal(wire.Transform3DResult{Created: ids, EntityCount: sk.EntityCount()})
}

// resolveEntities3D resolves a list of entity ids to entities.
func resolveEntities3D(sk *sketch.Sketch3D, ids []uint64) ([]sketch.Entity, error) {
	out := make([]sketch.Entity, len(ids))
	for i, id := range ids {
		e, ok := sk.EntityByID(sketch.ID(id))
		if !ok {
			return nil, fmt.Errorf("sketch3d.transform: no entity with id %d", id)
		}
		out[i] = e
	}
	return out, nil
}

// vector3Arg reads a required [x,y,z] vector.
func vector3Arg(v []float64) (math.Vector3, error) {
	if len(v) != 3 {
		return math.Vector3{}, fmt.Errorf("sketch3d.transform: vector must be [x,y,z], got %d components", len(v))
	}
	return math.V3(math.Scalar(v[0]), math.Scalar(v[1]), math.Scalar(v[2])), nil
}

// point3Arg reads a required [x,y,z] point.
func point3Arg(p []float64) (math.Point3, error) {
	if len(p) != 3 {
		return math.Point3{}, fmt.Errorf("sketch3d.transform: point must be [x,y,z], got %d components", len(p))
	}
	return math.P3(math.Scalar(p[0]), math.Scalar(p[1]), math.Scalar(p[2])), nil
}
