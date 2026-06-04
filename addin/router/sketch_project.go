// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/kernel/geom"
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
		if _, ok := body.FindEdgeByKey(key); ok {
			return sk.ProjectCurve(newEdgeSource(part, ref)), true
		}
		if _, ok := body.FindVertexByKey(key); ok {
			return sk.ProjectPoint(newVertexSource(part, ref)), true
		}
	}
	return nil, false
}

// edgeSource adapts a topo edge to a sketch CurveSource. It is self-resolving — it re-finds
// the edge by reference key among the part's current bodies on each sample — so a
// projection/include stays associative across recompute (the rebuilt B-rep has fresh edge
// pointers under the same keys), and reports lost when the key no longer resolves.
type edgeSource struct {
	ref    string
	bodies func() []*topo.Body
}

func newEdgeSource(part *compdef.PartComponentDefinition, ref string) edgeSource {
	// A closure (not the bound All method) so it sees the bodies *replaced* by recompute.
	return edgeSource{ref: ref, bodies: func() []*topo.Body { return part.SurfaceBodies().All() }}
}

func (s edgeSource) SourceID() string { return s.ref }

// SamplePoints re-resolves the edge by key and samples its curve; ok=false when lost.
func (s edgeSource) SamplePoints() ([]math.Point3, bool) {
	key := []byte(s.ref)
	for _, b := range s.bodies() {
		if edge, ok := b.FindEdgeByKey(key); ok {
			return sampleCurve3(edge.Geometry()), true
		}
	}
	return nil, false
}

// sampleCurve3 samples a 3D curve over its domain into a polyline.
func sampleCurve3(c geom.Curve3) []math.Point3 {
	const n = 16
	lo, hi := c.Domain()
	pts := make([]math.Point3, n+1)
	for i := range pts {
		pts[i] = c.PointAt(lo + (hi-lo)*float64(i)/float64(n))
	}
	return pts
}

// vertexSource adapts a topo vertex to a self-resolving sketch PointSource.
type vertexSource struct {
	ref    string
	bodies func() []*topo.Body
}

func newVertexSource(part *compdef.PartComponentDefinition, ref string) vertexSource {
	return vertexSource{ref: ref, bodies: func() []*topo.Body { return part.SurfaceBodies().All() }}
}

func (s vertexSource) SourceID() string { return s.ref }

// Position re-resolves the vertex by key; ok=false when lost.
func (s vertexSource) Position() (math.Point3, bool) {
	key := []byte(s.ref)
	for _, b := range s.bodies() {
		if v, ok := b.FindVertexByKey(key); ok {
			return v.Point(), true
		}
	}
	return math.Point3{}, false
}
