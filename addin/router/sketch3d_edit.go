// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// transformSketch3D applies a move/copy/rotate/delete edit to a 3D-sketch selection.
func transformSketch3D(_ *app.Session, part *compdef.PartComponentDefinition, in wire.Transform3DArgs) (wire.Transform3DResult, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.Transform3DResult{}, err
	}
	ents, err := resolveEntities3D(sk, in.Entities)
	if err != nil {
		return wire.Transform3DResult{}, err
	}
	return applyTransform3D(part, sk, ents, in)
}

// applyTransform3D dispatches the requested edit op and returns the result.
func applyTransform3D(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, ents []sketch.Entity, in wire.Transform3DArgs) (wire.Transform3DResult, error) {
	switch in.Op {
	case "move", "copy":
		return moveOrCopy3D(sk, ents, in)
	case "rotate":
		if err := rotate3D(part, sk, ents, in); err != nil {
			return wire.Transform3DResult{}, err
		}
	case "delete":
		for _, e := range ents {
			sk.DeleteEntity3D(e)
		}
	default:
		return wire.Transform3DResult{}, fmt.Errorf("sketch3d.transform: unknown op %q (want move|copy|rotate|delete)", in.Op)
	}
	return transform3DResult(sk, nil)
}

// moveOrCopy3D applies a translation (move) or a translated duplication (copy).
func moveOrCopy3D(sk *sketch.Sketch3D, ents []sketch.Entity, in wire.Transform3DArgs) (wire.Transform3DResult, error) {
	v, err := vector3Arg(in.Vector)
	if err != nil {
		return wire.Transform3DResult{}, err
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
	a, err := resolveQuantity(part, in.Angle, param.Angle)
	if err != nil {
		return fmt.Errorf("sketch3d.transform: angle %q: %w", in.Angle, err)
	}
	sk.RotateEntities3D(ents, center, axis, float64(a.Value))
	return nil
}

// transform3DResult reports the created ids (copy) and the resulting entity count.
func transform3DResult(sk *sketch.Sketch3D, created []sketch.Entity) (wire.Transform3DResult, error) {
	ids := make([]uint64, len(created))
	for i, e := range created {
		ids[i] = uint64(e.EntityID())
	}
	return wire.Transform3DResult{Created: ids, EntityCount: sk.EntityCount()}, nil
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
