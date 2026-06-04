// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// Projecting/including part geometry into a sketch (2D or 3D) and deriving surface curves
// all need an associative handle on a part edge/vertex/face: one that survives recompute
// (the rebuilt B-rep has fresh pointers under the same reference keys). These adapters are
// self-resolving — they re-find their target by reference key among the part's current
// bodies on every read — and report lost when the key no longer resolves. They live here,
// on the part, so the router and the interactive app tools share one implementation rather
// than each carrying its own (they structurally satisfy sketch.CurveSource / PointSource /
// SurfaceSource without this package importing the sketch seam types).

// referenceSampleSteps is how many straight segments a referenced edge curve is sampled to.
const referenceSampleSteps = 16

// EdgeRefSource adapts a part edge (by reference key) to a sketch curve source: it re-finds
// the edge among the part's current bodies and samples its curve into a polyline.
type EdgeRefSource struct {
	ref    string
	bodies func() []*topo.Body
}

// NewEdgeRefSource binds an associative curve source to the edge with reference key ref on
// part. The closure (not the bound method) is used so it sees the bodies replaced by
// recompute.
func NewEdgeRefSource(part *PartComponentDefinition, ref string) EdgeRefSource {
	return EdgeRefSource{ref: ref, bodies: func() []*topo.Body { return part.SurfaceBodies().All() }}
}

// SourceID returns the edge's reference key (its stable cross-recompute identity).
func (s EdgeRefSource) SourceID() string { return s.ref }

// SamplePoints re-resolves the edge by key and samples its curve; ok=false when lost.
func (s EdgeRefSource) SamplePoints() ([]math.Point3, bool) {
	key := []byte(s.ref)
	for _, b := range s.bodies() {
		if edge, ok := b.FindEdgeByKey(key); ok {
			return sampleReferenceCurve(edge.Geometry()), true
		}
	}
	return nil, false
}

// VertexRefSource adapts a part vertex (by reference key) to a sketch point source.
type VertexRefSource struct {
	ref    string
	bodies func() []*topo.Body
}

// NewVertexRefSource binds an associative point source to the vertex with reference key ref.
func NewVertexRefSource(part *PartComponentDefinition, ref string) VertexRefSource {
	return VertexRefSource{ref: ref, bodies: func() []*topo.Body { return part.SurfaceBodies().All() }}
}

// SourceID returns the vertex's reference key.
func (s VertexRefSource) SourceID() string { return s.ref }

// Position re-resolves the vertex by key; ok=false when lost.
func (s VertexRefSource) Position() (math.Point3, bool) {
	key := []byte(s.ref)
	for _, b := range s.bodies() {
		if v, ok := b.FindVertexByKey(key); ok {
			return v.Point(), true
		}
	}
	return math.Point3{}, false
}

// FaceRefSource adapts a part face (by reference key) to a sketch surface source: it
// re-finds the face among the part's current bodies and yields its surface. A lost key
// yields a nil surface, which the kernel tracer treats as no intersection.
type FaceRefSource struct {
	ref    string
	bodies func() []*topo.Body
}

// NewFaceRefSource binds an associative surface source to the face with reference key ref.
func NewFaceRefSource(part *PartComponentDefinition, ref string) FaceRefSource {
	return FaceRefSource{ref: ref, bodies: func() []*topo.Body { return part.SurfaceBodies().All() }}
}

// SourceID returns the face's reference key.
func (s FaceRefSource) SourceID() string { return s.ref }

// Surface re-resolves the face by key and returns its surface, or nil when lost.
func (s FaceRefSource) Surface() geom.Surface {
	key := []byte(s.ref)
	for _, b := range s.bodies() {
		if face, ok := b.FindFaceByKey(key); ok {
			return face.Geometry()
		}
	}
	return nil
}

// EdgeKeyResolves reports whether an edge reference key currently binds to a part edge;
// VertexKeyResolves and FaceKeyResolves are the vertex/face equivalents. Tools use these to
// classify a picked reference and to report a lost reference as unhealthy (not an error).
func (d *PartComponentDefinition) EdgeKeyResolves(ref string) bool {
	key := []byte(ref)
	for _, b := range d.SurfaceBodies().All() {
		if _, ok := b.FindEdgeByKey(key); ok {
			return true
		}
	}
	return false
}

// VertexKeyResolves reports whether a vertex reference key currently binds to a part vertex.
func (d *PartComponentDefinition) VertexKeyResolves(ref string) bool {
	key := []byte(ref)
	for _, b := range d.SurfaceBodies().All() {
		if _, ok := b.FindVertexByKey(key); ok {
			return true
		}
	}
	return false
}

// FaceKeyResolves reports whether a face reference key currently binds to a part face.
func (d *PartComponentDefinition) FaceKeyResolves(ref string) bool {
	key := []byte(ref)
	for _, b := range d.SurfaceBodies().All() {
		if _, ok := b.FindFaceByKey(key); ok {
			return true
		}
	}
	return false
}

// sampleReferenceCurve samples a 3D curve over its domain into a polyline.
func sampleReferenceCurve(c geom.Curve3) []math.Point3 {
	lo, hi := c.Domain()
	pts := make([]math.Point3, referenceSampleSteps+1)
	for i := range pts {
		pts[i] = c.PointAt(lo + (hi-lo)*float64(i)/float64(referenceSampleSteps))
	}
	return pts
}
