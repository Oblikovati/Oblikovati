// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Filling a four-sided opening among neighbour surface bodies (M36-F07): each neighbour contributes
// the boundary edge facing the opening; the four edges are chained into a loop and filled with a
// single NURBS that meets each neighbour at the requested continuity (geom.FillSurface). The result
// is a new one-face surface body covering the opening.

// boundaryEdge is one neighbour's contribution to a fill: the curve along its inner edge, plus the
// surface and edge it came from (for the continuity match). nurbs reports whether the neighbour is a
// NURBS surface — only then can tangent/curvature (G1/G2) matching apply; a planar (or other) face
// contributes its boundary curve and is filled position-only (G0) on that side.
type boundaryEdge struct {
	curve   geom.BSplineCurve
	surface geom.BSplineSurface
	edge    geom.Boundary
	nurbs   bool
}

// endpoints of a boundary curve.
func (b boundaryEdge) start() math.Point3 { return b.curve.Ctrl[0] }
func (b boundaryEdge) end() math.Point3   { return b.curve.Ctrl[len(b.curve.Ctrl)-1] }

// boundaryWeldTol is the endpoint-coincidence tolerance for chaining boundary edges,
// scaled to the boundary's own extent (#1610, ADR-0042 — formerly an absolute 1e-7 cm).
func boundaryWeldTol(edges []boundaryEdge) float64 {
	var pts []math.Point3
	for _, e := range edges {
		pts = append(pts, e.start(), e.end())
	}
	return ResolutionForPoints(pts).Weld()
}

// FillFourSided fills the opening bounded by four neighbour surface bodies with a single NURBS at the
// given continuity order (0=G0..2=G2). Each neighbour must be a single surface face (NURBS or planar);
// its inner edge (nearest the opening centre) bounds the fill. A NURBS neighbour is matched to the
// requested continuity; a planar neighbour fills position-only on that side (it has no curvature to
// match). It errors when a neighbour has no surface face or the four edges do not chain into a loop.
func FillFourSided(neighbours [4]*topo.Body, order int) (*topo.Body, error) {
	edges, err := openingEdges(neighbours)
	if err != nil {
		return nil, err
	}
	c0, c1, d0, d1, err := chainLoop(edges)
	if err != nil {
		return nil, err
	}
	sides := [4]geom.FillSide{fillSide(c0, order), fillSide(c1, order), fillSide(d0, order), fillSide(d1, order)}
	fill, err := geom.FillSurface(c0.curve, c1.curve, d0.curve, d1.curve, sides)
	if err != nil {
		return nil, fmt.Errorf("ops.FillFourSided: %w", err)
	}
	return fullDomainBody(fill, "fill"), nil
}

// fillSide builds the geom.FillSide for one boundary: a NURBS neighbour matches to the requested
// continuity; any other (planar) neighbour fills position-only (Order 0) so the fill still
// interpolates its boundary.
func fillSide(b boundaryEdge, order int) geom.FillSide {
	if b.nurbs && matchableColumns(b) {
		return geom.FillSide{Adjacent: b.surface, AdjEdge: b.edge, Order: order}
	}
	return geom.FillSide{Order: 0}
}

// matchableColumns reports whether the fill side's control-row count still equals its neighbour
// edge's, the precondition for MatchSurface. It is false when an N-sided fill had to refine this side
// (to stay knot-compatible with a merged opposite side), so that side falls back to G0 — the
// documented N-sided G2 convergence limit.
func matchableColumns(b boundaryEdge) bool {
	return len(b.curve.Ctrl) == len(edgeCurve(b.surface, b.edge).Ctrl)
}

// openingEdges returns each neighbour's inner boundary edge (nearest the centre of the neighbour
// faces). Each neighbour must be a single surface face (NURBS or planar).
func openingEdges(neighbours [4]*topo.Body) ([4]boundaryEdge, error) {
	var faces [4]*topo.Face
	var surfs [4]geom.Surface
	var sum math.Vector3
	for i, b := range neighbours {
		f, s, ok := firstSurfaceFace(b)
		if !ok {
			return [4]boundaryEdge{}, fmt.Errorf("ops.FillFourSided: neighbour %d has no surface face", i)
		}
		faces[i], surfs[i] = f, s
		sum = sum.Add(faceCentroid(f).AsVector()) // face position (robust for planar faces, unlike surface.PointAt)
	}
	center := sum.Scale(0.25).AsPoint()
	var edges [4]boundaryEdge
	for i := range neighbours {
		edges[i] = innerBoundary(faces[i], surfs[i], center)
	}
	return edges, nil
}

// faceCentroid averages a face's outer-loop edge start points — the face's position, valid for any
// surface type (a planar face's surface.PointAt does not track the trimmed face's location).
func faceCentroid(f *topo.Face) math.Point3 {
	var sum math.Vector3
	n := 0
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			c := u.Edge().Geometry()
			lo, _ := c.Domain()
			sum = sum.Add(c.PointAt(lo).AsVector())
			n++
		}
	}
	if n == 0 {
		return math.P3(0, 0, 0)
	}
	return sum.Scale(1 / float64(n)).AsPoint()
}

// firstSurfaceFace returns the body's first face carrying a surface (any kind), its surface, and ok.
func firstSurfaceFace(b *topo.Body) (*topo.Face, geom.Surface, bool) {
	for _, f := range b.Faces() {
		if s := f.Geometry(); s != nil {
			return f, s, true
		}
	}
	return nil, nil, false
}

// innerBoundary returns the neighbour's inner boundary edge facing the opening centre. A NURBS face
// uses its inner iso-row (so tangent/curvature matching can apply); any other surface (e.g. a planar
// patch) uses the inner topo edge's curve and is filled position-only on that side.
func innerBoundary(f *topo.Face, s geom.Surface, center math.Point3) boundaryEdge {
	if bs, ok := s.(geom.BSplineSurface); ok {
		e := innerEdge(bs, center)
		return boundaryEdge{curve: edgeCurve(bs, e), surface: bs, edge: e, nurbs: true}
	}
	return boundaryEdge{curve: innerTopoEdge(f, center)}
}

// innerTopoEdge returns, as a B-spline curve, the face boundary edge whose midpoint is nearest center.
func innerTopoEdge(f *topo.Face, center math.Point3) geom.BSplineCurve {
	var best geom.Curve3
	bestD := math.Scalar(stdmath.Inf(1))
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			c := u.Edge().Geometry()
			lo, hi := c.Domain()
			if d := c.PointAt((lo + hi) / 2).DistanceTo(center); d < bestD {
				best, bestD = c, d
			}
		}
	}
	return curveAsBSpline(best)
}

// curveAsBSpline returns c as a B-spline curve: itself if already one, else a degree-1 segment
// through its endpoints (exact for the straight edges of a planar patch).
func curveAsBSpline(c geom.Curve3) geom.BSplineCurve {
	if bs, ok := c.(geom.BSplineCurve); ok {
		return bs
	}
	lo, hi := c.Domain()
	bc, _ := geom.NewBSplineCurve(1, []math.Point3{c.PointAt(lo), c.PointAt(hi)}, []float64{1, 1}, []float64{0, 0, 1, 1})
	return bc
}

// chainLoop orders the four boundary edges into c0 (v=0), c1 (v=1), d0 (u=0), d1 (u=1) with curves
// reversed so the corners chain (c0.start=d0.start, c0.end=d1.start, c1.start=d0.end, c1.end=d1.end).
func chainLoop(edges [4]boundaryEdge) (c0, c1, d0, d1 boundaryEdge, err error) {
	tol := boundaryWeldTol(edges[:])
	c0 = edges[0]
	rest := []boundaryEdge{edges[1], edges[2], edges[3]}
	d0, ok0 := takeSharing(&rest, c0.start(), tol)
	d1, ok1 := takeSharing(&rest, c0.end(), tol)
	if !ok0 || !ok1 || len(rest) != 1 {
		return c0, c1, d0, d1, fmt.Errorf("ops.FillFourSided: the four edges do not form a closed loop")
	}
	c1 = rest[0]
	d0 = orient(d0, c0.start(), tol) // d0.start = corner00
	d1 = orient(d1, c0.end(), tol)   // d1.start = corner10
	c1 = orient(c1, d0.end(), tol)   // c1.start = corner01 (= d0.end)
	return c0, c1, d0, d1, nil
}

// takeSharing removes and returns the edge from rest that shares an endpoint with p (within tol).
func takeSharing(rest *[]boundaryEdge, p math.Point3, tol float64) (boundaryEdge, bool) {
	for i, e := range *rest {
		if e.start().IsEqualTo(p, tol) || e.end().IsEqualTo(p, tol) {
			*rest = append((*rest)[:i], (*rest)[i+1:]...)
			return e, true
		}
	}
	return boundaryEdge{}, false
}

// orient returns the edge with its curve reversed if needed so its start equals p.
func orient(e boundaryEdge, p math.Point3, tol float64) boundaryEdge {
	if e.start().IsEqualTo(p, tol) {
		return e
	}
	e.curve = reverseCurve(e.curve)
	return e
}

// reverseCurve reverses a B-spline curve's parameterization (control points and knots flipped).
func reverseCurve(c geom.BSplineCurve) geom.BSplineCurve {
	n := len(c.Ctrl)
	ctrl := make([]math.Point3, n)
	w := make([]float64, n)
	for i := range n {
		ctrl[i], w[i] = c.Ctrl[n-1-i], c.Weights[n-1-i]
	}
	knots := c.Knots
	lo, hi := knots[0], knots[len(knots)-1]
	rk := make([]float64, len(knots))
	for i := range knots {
		rk[i] = lo + hi - knots[len(knots)-1-i]
	}
	r, _ := geom.NewBSplineCurve(c.Degree, ctrl, w, rk)
	return r
}

// innerEdge returns the surface boundary whose midpoint is closest to center.
func innerEdge(s geom.BSplineSurface, center math.Point3) geom.Boundary {
	best := geom.UMinEdge
	bestD := math.Scalar(stdmath.Inf(1))
	for _, e := range []geom.Boundary{geom.UMinEdge, geom.UMaxEdge, geom.VMinEdge, geom.VMaxEdge} {
		if d := innerEdgeMidpoint(s, e).DistanceTo(center); d < bestD {
			best, bestD = e, d
		}
	}
	return best
}

// innerEdgeMidpoint returns the surface point at the middle of the given boundary edge.
func innerEdgeMidpoint(s geom.BSplineSurface, e geom.Boundary) math.Point3 {
	switch e {
	case geom.UMinEdge:
		return s.PointAt(0, 0.5)
	case geom.UMaxEdge:
		return s.PointAt(1, 0.5)
	case geom.VMinEdge:
		return s.PointAt(0.5, 0)
	default:
		return s.PointAt(0.5, 1)
	}
}

// edgeCurve returns the boundary iso-curve of a surface as a B-spline curve, reusing the untrim
// iso-curve extractors (which return the same concrete BSplineCurve).
func edgeCurve(s geom.BSplineSurface, e geom.Boundary) geom.BSplineCurve {
	switch e {
	case geom.UMinEdge:
		return uIsoCurve(s, false).(geom.BSplineCurve)
	case geom.UMaxEdge:
		return uIsoCurve(s, true).(geom.BSplineCurve)
	case geom.VMinEdge:
		return vIsoCurve(s, false).(geom.BSplineCurve)
	default:
		return vIsoCurve(s, true).(geom.BSplineCurve)
	}
}
