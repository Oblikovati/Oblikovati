// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Wire topology (M07-F06, Oblikovati/Oblikovati#629): an ordered chain of
// edges a body carries WITHOUT faces — the section/profile currency of ruled
// surfaces, silhouette results and plane sections. Wires reuse [Use] for
// orientation (a wire edge has no loop, so [EdgeUse] does not apply).

// Wire is an ordered, oriented chain of edges owned by a body.
type Wire struct {
	id      uint64
	body    *Body
	uses    []Use
	lineage Lineage
}

// KindWire continues the EntityKind block (values are persisted — never renumber).
const KindWire EntityKind = 7

func (w *Wire) ID() uint64       { return w.id }
func (w *Wire) Kind() EntityKind { return KindWire }
func (w *Wire) Body() *Body      { return w.body }
func (w *Wire) Lineage() Lineage { return w.lineage }

// ReferenceKey returns the wire's persistent reference key (M03 scheme).
func (w *Wire) ReferenceKey() []byte { return referenceKey(KindWire, w.lineage) }

// Uses returns the ordered, oriented edge uses of the chain.
func (w *Wire) Uses() []Use { return append([]Use(nil), w.uses...) }

// Edges returns the chain's edges in traversal order.
func (w *Wire) Edges() []*Edge {
	out := make([]*Edge, len(w.uses))
	for i, u := range w.uses {
		out[i] = u.Edge
	}
	return out
}

// IsClosed reports whether the chain returns to its start point.
func (w *Wire) IsClosed() bool {
	if len(w.uses) == 0 {
		return false
	}
	first, _ := useEnds(w.uses[0])
	_, last := useEnds(w.uses[len(w.uses)-1])
	return first.Point().IsEqualTo(last.Point(), wirePlanarTol)
}

// useEnds returns a use's traversal start and end vertices.
func useEnds(u Use) (*Vertex, *Vertex) {
	if u.Reversed {
		return u.Edge.EndVertex(), u.Edge.StartVertex()
	}
	return u.Edge.StartVertex(), u.Edge.EndVertex()
}

// wirePlanarTol is the deviation under which a wire counts as planar (and a
// chain as closed) — model units (cm).
const wirePlanarTol = 1e-7

// wireSamplesPerEdge is how densely IsPlanar/PlaneFrame sample each curve.
const wireSamplesPerEdge = 16

// IsPlanar reports whether every edge curve lies in one plane. A collinear
// chain (no unique plane) counts as planar — it lies in infinitely many.
func (w *Wire) IsPlanar() bool {
	if _, _, ok := w.PlaneFrame(); ok {
		return true
	}
	return w.collinear()
}

// PlaneFrame returns the wire's UNIQUE plane (origin, unit normal) when the
// sampled chain is planar within tolerance — the frame OffsetPlanarWire works
// in. A collinear chain reports false (no unique plane; pass the normal
// explicitly to the offset instead).
func (w *Wire) PlaneFrame() (math.Point3, math.Vector3, bool) {
	pts := w.samplePoints()
	if len(pts) < 3 {
		return math.Point3{}, math.Vector3{}, false
	}
	origin, normal, ok := newellPlane(pts)
	if !ok {
		return math.Point3{}, math.Vector3{}, false
	}
	for _, p := range pts {
		if stdmath.Abs(float64(origin.VectorTo(p).Dot(normal))) > wirePlanarTol {
			return math.Point3{}, math.Vector3{}, false
		}
	}
	return origin, normal, true
}

// collinear reports whether every sampled point lies on one line. The line
// direction comes from the FARTHEST sample (a closed chain ends where it
// starts, so first→last would degenerate to zero).
func (w *Wire) collinear() bool {
	pts := w.samplePoints()
	if len(pts) < 2 {
		return true
	}
	dir := farthestDirection(pts)
	if l := float64(dir.Length()); l == 0 {
		return true // all samples coincide
	} else {
		dir = dir.Scale(math.Scalar(1 / l))
	}
	for _, p := range pts {
		v := pts[0].VectorTo(p)
		if float64(v.Cross(dir).Length()) > wirePlanarTol {
			return false
		}
	}
	return true
}

// farthestDirection is the vector from the first sample to its farthest peer.
func farthestDirection(pts []math.Point3) math.Vector3 {
	best, bestD := math.Vector3{}, math.Scalar(0)
	for _, p := range pts {
		if d := pts[0].DistanceSquaredTo(p); d > bestD {
			best, bestD = pts[0].VectorTo(p), d
		}
	}
	return best
}

// samplePoints samples each edge curve at wire density.
func (w *Wire) samplePoints() []math.Point3 {
	var pts []math.Point3
	for _, u := range w.uses {
		c := u.Edge.Geometry()
		lo, hi := c.Domain()
		for i := 0; i <= wireSamplesPerEdge; i++ {
			pts = append(pts, c.PointAt(lo+(hi-lo)*float64(i)/wireSamplesPerEdge))
		}
	}
	return pts
}

// newellPlane fits a plane through the points (Newell's method: robust normal
// from the projected-area components, centroid as origin).
func newellPlane(pts []math.Point3) (math.Point3, math.Vector3, bool) {
	var n math.Vector3
	var c math.Vector3
	for i, p := range pts {
		q := pts[(i+1)%len(pts)]
		n.X += (p.Y - q.Y) * (p.Z + q.Z)
		n.Y += (p.Z - q.Z) * (p.X + q.X)
		n.Z += (p.X - q.X) * (p.Y + q.Y)
		c = c.Add(p.AsVector())
	}
	l := float64(n.Length())
	if l == 0 || l < 1e-12*spreadSquared(pts) {
		// Vanishing projected area RELATIVE to the chain's size: the closed
		// walk encloses nothing. A straight open chain lands here (collinear
		// handles it); so does a degenerate back-and-forth chain — neither has
		// a unique plane. The relative test keeps tiny-but-planar wires valid.
		return math.Point3{}, math.Vector3{}, false
	}
	origin := c.Scale(math.Scalar(1 / float64(len(pts)))).AsPoint()
	return origin, n.Scale(math.Scalar(1 / l)), true
}

// spreadSquared is the squared extent of the points about their first — the
// size reference for the relative degenerate-area test.
func spreadSquared(pts []math.Point3) float64 {
	best := 0.0
	for _, p := range pts {
		if d := float64(pts[0].DistanceSquaredTo(p)); d > best {
			best = d
		}
	}
	return best
}

// AttachWire appends an ordered edge chain to the body's wires. The uses' edges
// belong to the wire only (no faces); consecutive uses must chain end-to-start.
func (b *Body) AttachWire(lineage Lineage, uses []Use) *Wire {
	w := &Wire{id: nextID(), body: b, uses: append([]Use(nil), uses...), lineage: lineage}
	b.wires = append(b.wires, w)
	return w
}

// Wires returns the body's wires.
func (b *Body) Wires() []*Wire { return append([]*Wire(nil), b.wires...) }
