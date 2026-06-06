// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati/api/wire"

	"oblikovati/addin/modelaccess"
	"oblikovati/app"
	"oblikovati/model/compdef"
	"oblikovati/model/sketch"
)

// projectGeometry projects the referenced part edges/vertices onto the sketch plane as
// associative reference entities (re-derived through recompute via their source keys).
func projectGeometry(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.ProjectGeometryArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	created, healthy := projectRefs(part, sk, in.Refs)
	return json.Marshal(wire.ProjectGeometryResult{Created: created, Healthy: healthy})
}

// projectRefs resolves each reference to a part edge/vertex and projects it; an
// unresolved reference is skipped and the result reported unhealthy (not an error).
func projectRefs(part *compdef.PartComponentDefinition, sk *sketch.Sketch, refs []string) ([]uint64, bool) {
	var created []uint64
	healthy := true
	for _, ref := range refs {
		e, ok := projectRef(part, sk, ref)
		if !ok {
			healthy = false
			continue
		}
		created = append(created, uint64(e.EntityID()))
	}
	return created, healthy
}

// projectRef resolves one reference key to an edge or vertex among the part's bodies and
// projects it, returning the created sketch entity.
func projectRef(part *compdef.PartComponentDefinition, sk *sketch.Sketch, ref string) (sketch.Entity, bool) {
	if part.EdgeKeyResolves(ref) {
		return sk.ProjectCurve(compdef.NewEdgeRefSource(part, ref)), true
	}
	if part.VertexKeyResolves(ref) {
		return sk.ProjectPoint(compdef.NewVertexRefSource(part, ref)), true
	}
	return nil, false
}
