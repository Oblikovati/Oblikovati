// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// includeSketch3D links referenced part edges/vertices into a 3D sketch as associative
// reference geometry (re-derived through recompute via their source keys), reusing the
// edge/vertex source adapters of the 2D project-geometry path.
func includeSketch3D(_ *app.Session, part *compdef.PartComponentDefinition, in wire.IncludeSketch3DArgs) (wire.IncludeSketch3DResult, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.IncludeSketch3DResult{}, err
	}
	created, healthy := includeRefs3D(part, sk, in.Refs)
	return wire.IncludeSketch3DResult{Created: created, Healthy: healthy}, nil
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
	if part.EdgeKeyResolves(ref) {
		return sk.IncludeCurve3D(compdef.NewEdgeRefSource(part, ref)), true
	}
	if part.VertexKeyResolves(ref) {
		return sk.IncludePoint3D(compdef.NewVertexRefSource(part, ref)), true
	}
	return nil, false
}
