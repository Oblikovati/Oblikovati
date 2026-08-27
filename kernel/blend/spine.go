// SPDX-License-Identifier: GPL-2.0-only

// Package blend is the OCCT-parity blend engine (ADR-0050): one section-driven pipeline for
// fillets and chamfers built on a guideline (the Spine), a section functional, an ODE marcher,
// and a B-spline approximation. It imports only geom and topo — the fillet/chamfer features in
// kernel/ops drive it, never the reverse, so the dependency points inward toward the domain.
package blend

import (
	"fmt"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Spine is the guideline a fillet or chamfer follows: an ordered run of tangent-continuous solid
// edges concatenated into a single arc-length parameterization, over which the section functional
// is swept. It mirrors OCCT ChFiDS_Spine — the same role (First/Last parameter, D0/D1, closed/open,
// single-edge fast-path accessors) on our geom.Curve3 edges. The abscissa runs 0..Length in the
// chain's traversal order; each edge is traversed entry→exit even when its intrinsic curve runs
// the other way (see forward).
//
// Build it from a tangent chain (ops.TangentEdgeChain):
//
//	sp, err := blend.NewSpine(edges, closed)
//	p := sp.PointAt(sp.Length() / 2) // midpoint of the guideline
type Spine struct {
	edges   []*topo.Edge
	forward []bool    // per edge: true ⇒ traverse lo→hi, false ⇒ hi→lo
	starts  []float64 // cumulative spine abscissa at each edge's entry vertex
	lengths []float64 // arc length of each edge
	total   float64
	closed  bool
}

// NewSpine concatenates an ordered, tangent-continuous edge run into a guideline. closed marks a
// loop (last edge's exit rejoins the first edge's entry). It errors on an empty run.
func NewSpine(edges []*topo.Edge, closed bool) (*Spine, error) {
	if len(edges) == 0 {
		return nil, fmt.Errorf("blend.NewSpine: empty edge run")
	}
	s := &Spine{
		edges:   edges,
		closed:  closed,
		forward: edgeDirections(edges),
		starts:  make([]float64, len(edges)),
		lengths: make([]float64, len(edges)),
	}
	for i, e := range edges {
		lo, hi := e.Geometry().Domain()
		s.starts[i] = s.total
		s.lengths[i] = arcLength(e.Geometry(), lo, hi)
		s.total += s.lengths[i]
	}
	return s, nil
}

// Length is the total guideline arc length; FirstParameter/LastParameter bound the abscissa.
func (s *Spine) Length() float64         { return s.total }
func (s *Spine) FirstParameter() float64 { return 0 }
func (s *Spine) LastParameter() float64  { return s.total }
func (s *Spine) IsClosed() bool          { return s.closed }
func (s *Spine) NbEdges() int            { return len(s.edges) }
func (s *Spine) Edge(i int) *topo.Edge   { return s.edges[i] }

// IsSingleEdge reports the known-part fast path: a lone elementary edge (a line or a circle)
// whose blend the analytic catalog can build without marching (ADR-0050, cf. ChFiKPart).
func (s *Spine) IsSingleEdge() bool { return len(s.edges) == 1 }

// PointAt returns the guideline position at spine abscissa absc (clamped to 0..Length).
func (s *Spine) PointAt(absc float64) math.Point3 {
	i, t := s.locate(absc)
	return s.edges[i].Geometry().PointAt(t)
}

// TangentAt returns the unit guideline tangent at absc, oriented in the chain's traversal
// direction (negated on edges traversed hi→lo).
func (s *Spine) TangentAt(absc float64) math.Vector3 {
	i, t := s.locate(absc)
	v := s.edges[i].Geometry().TangentAt(t)
	if !s.forward[i] {
		v = v.Negate()
	}
	if l := float64(v.Length()); l > 0 {
		v = v.Scale(math.Scalar(1 / l))
	}
	return v
}

// EdgeSpineRange returns the spine-abscissa span [first,last] that edge i covers — the guide range
// the marcher builds one blend segment over.
func (s *Spine) EdgeSpineRange(i int) (first, last float64) {
	return s.starts[i], s.starts[i] + s.lengths[i]
}

// SupportFaces returns the two solid faces edge i bounds — the pair the section functional
// blends between at that stretch of the guideline. Orientation (which is "left") is the
// marcher's concern; this returns them as the topology records.
func (s *Spine) SupportFaces(i int) []*topo.Face { return s.edges[i].Faces() }

// locate maps a spine abscissa to the (edge index, curve parameter) that realizes it, inverting
// each edge's arc length and honoring its traversal direction.
func (s *Spine) locate(absc float64) (int, float64) {
	absc = math.Clamp(absc, 0, s.total)
	i := s.edgeIndex(absc)
	local := absc - s.starts[i]
	lo, hi := s.edges[i].Geometry().Domain()
	if s.forward[i] {
		return i, paramAtArcLength(s.edges[i].Geometry(), lo, hi, local)
	}
	return i, paramAtArcLength(s.edges[i].Geometry(), lo, hi, s.lengths[i]-local)
}

// edgeIndex returns the edge covering abscissa absc (the last edge whose entry is ≤ absc).
func (s *Spine) edgeIndex(absc float64) int {
	for i := len(s.edges) - 1; i > 0; i-- {
		if absc >= s.starts[i] {
			return i
		}
	}
	return 0
}
