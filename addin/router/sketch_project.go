// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/sketch"
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
	key := []byte(ref)
	for _, body := range part.SurfaceBodies().All() {
		if edge, ok := body.FindEdgeByKey(key); ok {
			return sk.ProjectCurve(edgeSource{ref: ref, edge: edge}), true
		}
		if vertex, ok := body.FindVertexByKey(key); ok {
			return sk.ProjectPoint(vertexSource{ref: ref, vertex: vertex}), true
		}
	}
	return nil, false
}

// edgeSource adapts a topo edge to a sketch CurveSource (sampling its 3D curve).
type edgeSource struct {
	ref  string
	edge *topo.Edge
}

func (s edgeSource) SourceID() string { return s.ref }

// SamplePoints samples the edge's curve over its domain into a 3D polyline.
func (s edgeSource) SamplePoints() []math.Point3 {
	const n = 16
	c := s.edge.Geometry()
	lo, hi := c.Domain()
	pts := make([]math.Point3, n+1)
	for i := range pts {
		pts[i] = c.PointAt(lo + (hi-lo)*float64(i)/float64(n))
	}
	return pts
}

// vertexSource adapts a topo vertex to a sketch PointSource.
type vertexSource struct {
	ref    string
	vertex *topo.Vertex
}

func (s vertexSource) SourceID() string      { return s.ref }
func (s vertexSource) Position() math.Point3 { return s.vertex.Point() }
