// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// includeSketch3D links referenced part edges/vertices into a 3D sketch as associative
// reference geometry (re-derived through recompute via their source keys), reusing the
// edge/vertex source adapters of the 2D project-geometry path.
func includeSketch3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.IncludeSketch3DArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	created, healthy := includeRefs3D(part, sk, in.Refs)
	return json.Marshal(wire.IncludeSketch3DResult{Created: created, Healthy: healthy})
}

// includeRefs3D resolves each reference to a part edge/vertex and includes it; an
// unresolved reference is skipped and the result reported unhealthy (not an error).
func includeRefs3D(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, refs []string) ([]uint64, bool) {
	var created []uint64
	healthy := true
	for _, ref := range refs {
		e, ok := includeRef3D(part, sk, ref)
		if !ok {
			healthy = false
			continue
		}
		created = append(created, uint64(e.EntityID()))
	}
	return created, healthy
}

// includeRef3D resolves one reference key to an edge or vertex among the part's bodies and
// includes it into the 3D sketch (an edge as a reference polyline, a vertex as a point).
func includeRef3D(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, ref string) (sketch.Entity, bool) {
	key := []byte(ref)
	for _, body := range part.SurfaceBodies().All() {
		if _, ok := body.FindEdgeByKey(key); ok {
			return sk.IncludeCurve3D(newEdgeSource(part, ref)), true
		}
		if _, ok := body.FindVertexByKey(key); ok {
			return sk.IncludePoint3D(newVertexSource(part, ref)), true
		}
	}
	return nil, false
}
