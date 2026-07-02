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
// surface, sense (reversed) and lineage; loops[0] is the outer loop, the rest holes. The weld grid is
// the faces' own stitch resolution (ADR-0042, #1602): the seam points fed in carry SSI-tracer noise
// proportional to the operands' extent, so an absolute grid tears seams on parts it was never
// calibrated for.
func curvedStitch(faces []curvedFace) *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("curvedbool", "body", 0)))
	pw := newWelder3(geom.ResolutionForBox(curvedFaceBox(faces)).Stitch())
	w := &curveWelder{bld: bld, pw: pw, verts: map[int]*topo.Vertex{}, edges: map[[3]int]*topo.Edge{}}
	provByFace := map[*topo.Face]topo.Lineage{}
	for _, f := range faces {
		specs := w.loopSpecs(f.loops, f.outerless)
		var built *topo.Face
		if f.reversed {
			built = bld.AddReversedFace(f.surface, f.lineage, specs...)
		} else {
			built = bld.AddFace(f.surface, f.lineage, specs...)
		}
		if len(f.lineage.Key()) > 0 {
			provByFace[built] = f.lineage
		}
	}
	body := bld.Build()
	// ADR-0043 SSI-edge provenance: the welded edges are minted with build-order ordinals
	// (curvedbool:e#N) that renumber under any upstream edit. Each result face carries a stable
	// provenance lineage (an original face's key, or a wall/cap's parent-derived name), so rename the
	// surface-intersection edges by their bordering face pair — a build-order-independent name. The
	// caller's InheritOriginalEdges then restores the identity of original boundaries passed through
	// whole (a survivor must keep its OWN key, not a face-pair name), see booleanGeneral.
	if len(provByFace) > 0 {
		body.RelineageByFaceProvenance(provByFace, topo.Tok("curvedbool", "x", 0), topo.Tok("curvedbool", "seg", 0))
	}
	return body
}

// curveWelder dedups vertices (by position) and edges (by endpoints + midpoint) as faces are added.
// Positions canonicalise through a shared point welder (grid + 26-neighbour distance search), so two
// independently computed copies of one seam point weld even when they straddle a grid-cell boundary —
// the failure mode of the retired exact-cell string keys (#1602).
type curveWelder struct {
	bld   *topo.Builder
	pw    *welder3
	verts map[int]*topo.Vertex
	edges map[[3]int]*topo.Edge
	nv    int
	ne    int
}

// curvedFaceBox bounds the loop-edge endpoints of the faces being stitched — the geometry whose
// Resolution sets the stitch weld grid (#1602).
func curvedFaceBox(faces []curvedFace) math.Box {
	box := math.EmptyBox()
	for _, f := range faces {
		for _, loop := range f.loops {
			for _, le := range loop.edges {
				box = box.ExtendPoint(le.start()).ExtendPoint(le.end())
			}
		}
	}
	return box
}

// loopSpecs turns a face's curved loops into builder loop specs (outer first, the rest holes). When
// outerless, EVERY loop is a hole — a face on a closed surface that wraps the whole surface minus its holes
// (the torus complement, #1406), which has no outer loop.
func (w *curveWelder) loopSpecs(loops []curvedLoop, outerless bool) []topo.LoopSpec {
	specs := make([]topo.LoopSpec, 0, len(loops))
	for li, loop := range loops {
		uses := make([]topo.Use, 0, len(loop.edges))
		for _, le := range loop.edges {
			edge, reversed := w.edge(le)
			uses = append(uses, topo.Use{Edge: edge, Reversed: reversed})
		}
		if li == 0 && !outerless {
			specs = append(specs, topo.OuterLoop(uses...))
		} else {
			specs = append(specs, topo.InnerLoop(uses...))
		}
	}
	return specs
}

// vertex welds a point to a shared topo vertex by canonical welded position.
func (w *curveWelder) vertex(p math.Point3) *topo.Vertex {
	key := w.pw.add(p)
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
	closed := w.pw.add(a) == w.pw.add(b)
	key := w.edgeKey(a, b, mid)
	if e, ok := w.edges[key]; ok {
		if closed {
			return e, le.t1 < le.t0
		}
		return e, w.pw.add(e.StartVertex().Point()) != w.pw.add(a)
	}
	return w.newEdge(le, a, b, key, closed)
}

// newEdge creates and records a welded edge oriented along this loop's traversal (so the creating loop
// uses it forward, except a reversed-sweep closed circle). An OPEN spiric branch is stored in its native
// direction (V0<V1, see spiricArcOf) and anchored to the curve's own endpoints, so the reversed flag — not a
// flipped curve — orients it; the direction-sensitive spiric mesher then meshes the same patch either way.
func (w *curveWelder) newEdge(le loopEdge, a, b math.Point3, key [3]int, closed bool) (*topo.Edge, bool) {
	curve := edgeCurveFor(le)
	if sa, ok := curve.(geom.SpiricArc); ok && !closed {
		ca, cb := sa.PointAt(0), sa.PointAt(1)
		e := w.bld.AddEdge(curve, w.vertex(ca), w.vertex(cb), topo.NewLineage(topo.Tok("curvedbool", "e", w.ne)))
		w.ne++
		w.edges[key] = e
		return e, w.pw.add(ca) != w.pw.add(a) // reversed when this loop starts at the arc's far end
	}
	va, vb := w.vertex(a), w.vertex(b)
	e := w.bld.AddEdge(curve, va, vb, topo.NewLineage(topo.Tok("curvedbool", "e", w.ne)))
	w.ne++
	w.edges[key] = e
	if closed {
		return e, le.t1 < le.t0
	}
	return e, false
}

// edgeKey is the canonical weld key for an edge: its two welded endpoints (sorted, so direction
// does not matter) plus its welded midpoint (so two curves joining the same endpoints stay distinct).
func (w *curveWelder) edgeKey(a, b, mid math.Point3) [3]int {
	ka, kb := w.pw.add(a), w.pw.add(b)
	if ka > kb {
		ka, kb = kb, ka
	}
	return [3]int{ka, kb, w.pw.add(mid)}
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
		return ellipticalSubArc(c, le.t0, le.t1) // restrict/re-anchor to the run's [t0,t1] (the reversed lobe walk)
	case geom.EllipseFull:
		return ellipseArcOf(c, le.t0, le.t1) // a section sub-arc of a full ellipse (the (u,v) cone split)
	case geom.SpiricArc:
		return spiricArcOf(c, le.t0, le.t1) // a torus-cut spiric branch, oriented to the loop's traversal
	default:
		return c
	}
}

// spiricArcOf restricts a SpiricArc to its loop sub-range [t0, t1], stored in its NATIVE tube-angle direction
// (V0 < V1) regardless of how this loop walks it — orientation is carried by the edge's reversed flag, not by
// flipping V0/V1. A reversed-range edge (V0 > V1) would mesh as a DIFFERENT region in the direction-sensitive
// spiric loft (the two branches of a bigon must both be native so the cap patch comes out the right size,
// #1406); newEdge anchors the edge to this native arc's endpoints so the reversed flag stays correct.
func spiricArcOf(sa geom.SpiricArc, t0, t1 float64) geom.Curve3 {
	v0 := sa.V0 + t0*(sa.V1-sa.V0)
	v1 := sa.V0 + t1*(sa.V1-sa.V0)
	if v0 > v1 {
		v0, v1 = v1, v0
	}
	sa.V0, sa.V1 = v0, v1
	return sa
}

// ellipticalSubArc restricts a partial EllipticalArc to its loop sub-range [t0, t1], re-anchored so the
// stored curve's PointAt(0) is the edge's StartVertex and PointAt(1) its EndVertex (EllipticalArc.PointAt(t)
// walks StartAngle+t·SweepAngle over t∈[0,1]). A lobe of the equal-radius Steinmetz bicylinder walks its
// shared arc in the arc's DECREASING-parameter direction (t0=1, t1=0); keeping the arc's original forward
// parameterisation left PointAt(0) at the FAR pinch, 2R from the edge's StartVertex, so the face's
// discretised boundary crossed the solid and the (u,v) trim loop self-intersected (#1403). For a run that
// already spans the whole arc forward (t0=0, t1=1, the oblique cone-cut rim/lid) this returns an identical
// arc, so those paths are unchanged.
func ellipticalSubArc(e geom.EllipticalArc, t0, t1 float64) geom.Curve3 {
	a, _ := geom.NewEllipticalArc(e.Center, e.Normal.AsVector(), e.MajorAxis.AsVector(), e.MajorRadius, e.MinorRadius,
		e.StartAngle+t0*e.SweepAngle, (t1-t0)*e.SweepAngle)
	return a
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
