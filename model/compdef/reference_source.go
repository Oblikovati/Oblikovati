// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
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

// SourceKind tags this as an edge reference for projection persistence/rebind (#1268).
func (s EdgeRefSource) SourceKind() string { return "edge" }

// SamplePoints re-resolves the edge by key and samples its curve; ok=false when lost. It is the
// FALLBACK path: projection prefers the analytic SourceCurve below and only samples a source with
// no single analytic curve (ADR-0055).
func (s EdgeRefSource) SamplePoints() ([]math.Point3, bool) {
	if c, ok := s.SourceCurve(); ok {
		return geom.SampleCurve3(c, referenceSampleSteps), true
	}
	return nil, false
}

// SourceCurve re-resolves the edge by key and returns its exact analytic curve, so a projection
// keeps the curve analytic (a projected circle is a circle) instead of a sampled polyline
// (ADR-0055). ok=false when the reference is lost.
func (s EdgeRefSource) SourceCurve() (geom.Curve3, bool) {
	key := []byte(s.ref)
	for _, b := range s.bodies() {
		if edge, ok := b.FindEdgeByKey(key); ok {
			return edge.Geometry(), true
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

// SourceKind tags this as a vertex reference for projection persistence/rebind (#1268).
func (s VertexRefSource) SourceKind() string { return "vertex" }

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

// Surface re-resolves the face by key and returns its surface, or nil when lost. It recovers a
// lone ancestral sibling when the exact face is gone (ADR-0043 P6 / #1579) so an associative
// surface source survives an upstream edit that renames its face, rather than silently projecting
// onto nothing. FaceKeyResolves below deliberately does NOT recover — it is the health probe.
func (s FaceRefSource) Surface() geom.Surface {
	key := []byte(s.ref)
	for _, b := range s.bodies() {
		if face, ok := feature.FindOrRecoverFace(b, key); ok {
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
