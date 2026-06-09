// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/sketch"
)

// includeSketch2DInto3D links geometry of an existing 2D sketch (points/curves, by session
// id) into a 3D sketch as associative reference geometry, lifted through the 2D sketch's
// host plane. It reuses the model-side sketch2D source adapters, which re-resolve their
// entity on every read so the include tracks edits to the source sketch through recompute.
func includeSketch2DInto3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.IncludeSketch2DArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	target, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	source, err := sketchAtIndex(part, in.SourceSketchIndex)
	if err != nil {
		return nil, err
	}
	created, healthy := includeSketch2DEntities(target, source, in.EntityIDs)
	return json.Marshal(wire.IncludeSketch3DResult{Created: created, Healthy: healthy})
}

// includeSketch2DEntities includes each source 2D entity into the target 3D sketch — a
// point as a constrainable anchor, any other curve as a reference polyline. An id that
// doesn't resolve in the source sketch is skipped and the result reported unhealthy.
func includeSketch2DEntities(target *sketch.Sketch3D, source *sketch.Sketch, ids []uint64) ([]uint64, bool) {
	var created []uint64
	healthy := true
	for _, id := range ids {
		e, ok := includeSketch2DEntity(target, source, sketch.ID(id))
		if !ok {
			healthy = false
			continue
		}
		created = append(created, uint64(e.EntityID()))
	}
	return created, healthy
}

// includeSketch2DEntity includes one source 2D entity (resolved by id) into target. A
// standalone sketch point becomes an included anchor point; every other entity becomes an
// included reference polyline.
func includeSketch2DEntity(target *sketch.Sketch3D, source *sketch.Sketch, id sketch.ID) (sketch.Entity, bool) {
	ent, ok := source.EntityByID(id)
	if !ok {
		return nil, false
	}
	if _, isPoint := ent.(*sketch.Point); isPoint {
		return target.IncludePoint3D(sketch.NewSketch2DPointSource(source, id)), true
	}
	return target.IncludeCurve3D(sketch.NewSketch2DCurveSource(source, id)), true
}
