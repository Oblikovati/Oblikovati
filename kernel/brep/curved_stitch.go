// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved stitch (M2 Phase 1, Oblikovati/Oblikovati#1334). The general curved boolean's split stage
// produces a set of curvedFaces (each an analytic surface + curved boundary loops); this welds them
// into one topology. It is the curved analogue of the planar boolean's stitch (boolean_stitch.go):
// vertices weld by position, edges weld by their endpoints PLUS a midpoint (so the two different curves
// that can join the same pair of points — a boundary arc and the imprint arc between the same two cut
// vertices — stay distinct, not merged). A sub-range of a circle is stored as an Arc3d so the edge
// tessellates over the arc, not the whole circle (TessellateEdge walks the curve's whole Domain).

// curvedStitch welds curvedFaces into a body, sharing welded vertices and edges. Each face keeps its
// surface, sense (reversed) and lineage; loops[0] is the outer loop, the rest holes.
func curvedStitch(faces []curvedFace) *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("curvedbool", "body", 0)))
	w := &curveWelder{bld: bld, verts: map[string]*topo.Vertex{}, edges: map[string]*topo.Edge{}}
	for _, f := range faces {
		specs := w.loopSpecs(f.loops)
		if f.reversed {
			bld.AddReversedFace(f.surface, f.lineage, specs...)
		} else {
			bld.AddFace(f.surface, f.lineage, specs...)
		}
	}
	return bld.Build()
}

// curveWelder dedups vertices (by position) and edges (by endpoints + midpoint) as faces are added.
type curveWelder struct {
	bld   *topo.Builder
	verts map[string]*topo.Vertex
	edges map[string]*topo.Edge
	nv    int
	ne    int
}

// loopSpecs turns a face's curved loops into builder loop specs (outer first, the rest holes).
func (w *curveWelder) loopSpecs(loops []curvedLoop) []topo.LoopSpec {
	specs := make([]topo.LoopSpec, 0, len(loops))
	for li, loop := range loops {
		uses := make([]topo.Use, 0, len(loop.edges))
		for _, le := range loop.edges {
			edge, reversed := w.edge(le)
			uses = append(uses, topo.Use{Edge: edge, Reversed: reversed})
		}
		if li == 0 {
			specs = append(specs, topo.OuterLoop(uses...))
		} else {
			specs = append(specs, topo.InnerLoop(uses...))
		}
	}
	return specs
}

// vertex welds a point to a shared topo vertex by rounded position.
func (w *curveWelder) vertex(p math.Point3) *topo.Vertex {
	key := roundKey(p)
	if v, ok := w.verts[key]; ok {
		return v
	}
	v := w.bld.AddVertex(p, topo.NewLineage(topo.Tok("curvedbool", "v", w.nv)))
	w.nv++
	w.verts[key] = v
	return v
}

// edge welds a loop edge to a shared topo edge and reports whether THIS loop traverses it reversed. A
// closed edge (a full seam circle, start point == end point) is oriented by its sweep sign (t1 < t0);
// an open edge by whether this loop starts at the edge's stored start vertex.
func (w *curveWelder) edge(le loopEdge) (*topo.Edge, bool) {
	a, b := le.start(), le.end()
	mid := le.curve.PointAt((le.t0 + le.t1) / 2)
	closed := roundKey(a) == roundKey(b)
	key := edgeKey(a, b, mid)
	if e, ok := w.edges[key]; ok {
		if closed {
			return e, le.t1 < le.t0
		}
		return e, roundKey(e.StartVertex().Point()) != roundKey(a)
	}
	return w.newEdge(le, a, b, key, closed)
}

// newEdge creates and records a welded edge oriented along this loop's traversal (so the creating loop
// uses it forward, except a reversed-sweep closed circle).
func (w *curveWelder) newEdge(le loopEdge, a, b math.Point3, key string, closed bool) (*topo.Edge, bool) {
	va, vb := w.vertex(a), w.vertex(b)
	e := w.bld.AddEdge(edgeCurveFor(le), va, vb, topo.NewLineage(topo.Tok("curvedbool", "e", w.ne)))
	w.ne++
	w.edges[key] = e
	if closed {
		return e, le.t1 < le.t0
	}
	return e, false
}

// edgeKey is the canonical weld key for an edge: its two endpoints (sorted, so direction does not
// matter) plus its midpoint (so two curves joining the same endpoints stay distinct).
func edgeKey(a, b, mid math.Point3) string {
	ka, kb := roundKey(a), roundKey(b)
	if ka > kb {
		ka, kb = kb, ka
	}
	return ka + "|" + kb + "|" + roundKey(mid)
}

// edgeCurveFor returns the curve to store on the topo edge so its WHOLE domain is exactly the loop
// edge's [t0, t1] segment: a circle/arc sub-range becomes an Arc3d (else TessellateEdge would walk the
// full circle), a line sub-range a LineSegment between the endpoints, a full closed curve is kept.
func edgeCurveFor(le loopEdge) geom.Curve3 {
	switch c := le.curve.(type) {
	case geom.Circle:
		if isFullDomain(le.t0, le.t1) {
			return c
		}
		return arcOfCircle(c, le.t0, le.t1)
	case geom.Arc3d:
		return subArc(c, le.t0, le.t1)
	case geom.LineSegment:
		return geom.NewLineSegment(le.start(), le.end())
	case geom.Line:
		return geom.NewLineSegment(le.start(), le.end())
	default:
		return conicEdgeCurveFor(le)
	}
}

// conicEdgeCurveFor restricts the analytic conic edges (the oblique cone-cut sections) to their loop
// sub-range: a hyperbola/parabola to its bounded arc, an elliptical arc as-is, a full ellipse to the
// elliptical arc over [t0, t1]. Any other curve is stored whole.
func conicEdgeCurveFor(le loopEdge) geom.Curve3 {
	switch c := le.curve.(type) {
	case geom.Hyperbola:
		return c.Arc(le.t0, le.t1) // a hyperbola loop edge's params are θ; the bounded arc is what the edge stores
	case geom.Parabola:
		return c.Arc(le.t0, le.t1) // a parabola loop edge's params are the cross coordinate t; store the bounded arc
	case geom.EllipticalArc:
		return c // the re-anchored elliptical rim/lid of an oblique cone cut tessellates over its sweep
	case geom.EllipseFull:
		return ellipseArcOf(c, le.t0, le.t1) // a section sub-arc of a full ellipse (the (u,v) cone split)
	default:
		return c
	}
}

// ellipseArcOf builds the EllipticalArc covering a full ellipse's parameter sub-range [t0, t1]
// (EllipseFull.PointAt(t) is the point at angle 2πt), so the edge tessellates over that arc alone.
func ellipseArcOf(e geom.EllipseFull, t0, t1 float64) geom.Curve3 {
	const twoPi = 2 * stdmath.Pi
	a, _ := geom.NewEllipticalArc(e.Center, e.Normal.AsVector(), e.MajorAxis.AsVector(), e.MajorRadius, e.MinorRadius, twoPi*t0, twoPi*(t1-t0))
	return a
}

// isFullDomain reports whether [t0, t1] spans a curve's whole [0, 1] domain (a closed seam circle),
// in either direction.
func isFullDomain(t0, t1 float64) bool {
	lo, hi := stdmath.Min(t0, t1), stdmath.Max(t0, t1)
	return lo < 1e-9 && hi > 1-1e-9
}

// arcOfCircle builds the Arc3d covering a circle's parameter sub-range [t0, t1] (Circle.PointAt(t) is
// the point at angle 2πt), so the edge tessellates over that arc alone.
func arcOfCircle(c geom.Circle, t0, t1 float64) geom.Curve3 {
	const twoPi = 2 * stdmath.Pi
	a, _ := geom.NewArc3d(c.Center, c.Normal.AsVector(), c.RefDir.AsVector(), c.Radius, twoPi*t0, twoPi*(t1-t0))
	return a
}

// subArc restricts an Arc3d to a parameter sub-range [t0, t1].
func subArc(a geom.Arc3d, t0, t1 float64) geom.Curve3 {
	return geom.Arc3d{
		Center: a.Center, Normal: a.Normal, RefDir: a.RefDir, Radius: a.Radius,
		StartAngle: a.StartAngle + t0*a.SweepAngle, SweepAngle: (t1 - t0) * a.SweepAngle,
	}
}
