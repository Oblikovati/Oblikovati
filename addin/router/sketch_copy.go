// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Sketch-to-sketch copy (#151, #1083): re-instantiate one sketch's geometry in another. The
// source entities' sketch-local coordinates are cloned into the target sketch's plane (offset
// by Position), reusing the cross-sketch clone machinery. CopyEntitiesWithConstraints also
// carries over the geometric constraints and dimensions whose operands are entirely within the
// copied set, dropping any that reference geometry left behind (Inventor CopyEntitiesTo).

// sketchCopyTo copies geometry from one 2D sketch into another.
func sketchCopyTo(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.CopySketchArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	source, target, err := resolveCopySketches(s, in)
	if err != nil {
		return nil, err
	}
	ents, err := copySourceEntities(source, in.EntityIDs)
	if err != nil {
		return nil, err
	}
	offset, err := copyOffset(in.Position)
	if err != nil {
		return nil, err
	}
	created, warnings := target.CopyEntitiesWithConstraints(source, ents, offset)
	// Constraint kinds that cannot travel with a copy (#1637) are logged rather than
	// dropped silently; surfacing them in CopySketchResult needs a wire DTO field
	// (api bump), tracked as the #1637 follow-up.
	for _, w := range warnings {
		slog.Warn("sketch.copyTo constraint skipped", "warning", w)
	}
	ids := make([]uint64, len(created))
	for i, e := range created {
		ids[i] = uint64(e.EntityID())
	}
	return json.Marshal(wire.CopySketchResult{Created: ids, Count: len(ids)})
}

// resolveCopySketches resolves and validates the distinct source and target sketches.
func resolveCopySketches(s *app.Session, in wire.CopySketchArgs) (*sketch.Sketch, *sketch.Sketch, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, nil, err
	}
	source, err := sketchAtIndex(part, in.SourceIndex)
	if err != nil {
		return nil, nil, fmt.Errorf("sketch.copyTo: source: %w", err)
	}
	target, err := sketchAtIndex(part, in.TargetIndex)
	if err != nil {
		return nil, nil, fmt.Errorf("sketch.copyTo: target: %w", err)
	}
	if source == target {
		return nil, nil, fmt.Errorf("sketch.copyTo: source and target are the same sketch (%d) — use sketch.transform to copy within a sketch", in.SourceIndex)
	}
	return source, target, nil
}

// copySourceEntities resolves the entities to copy: the listed ids, or the whole sketch when
// none are given.
func copySourceEntities(source *sketch.Sketch, ids []uint64) ([]sketch.Entity, error) {
	if len(ids) == 0 {
		return source.Entities(), nil
	}
	out := make([]sketch.Entity, 0, len(ids))
	for _, id := range ids {
		e, ok := source.EntityByID(sketch.ID(id))
		if !ok {
			return nil, fmt.Errorf("sketch.copyTo: entity %d is not in the source sketch", id)
		}
		out = append(out, e)
	}
	return out, nil
}

// copyOffset turns the optional [x, y] position into a translation vector (zero when absent).
func copyOffset(pos []float64) (math.Vector2, error) {
	if len(pos) == 0 {
		return math.V2(0, 0), nil
	}
	if len(pos) != 2 {
		return math.Vector2{}, fmt.Errorf("sketch.copyTo: position must be [x, y], got %d values", len(pos))
	}
	return math.V2(math.Scalar(pos[0]), math.Scalar(pos[1])), nil
}
