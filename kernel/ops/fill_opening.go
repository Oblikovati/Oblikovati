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
// surface and edge it came from (for the continuity match).
type boundaryEdge struct {
	curve   geom.BSplineCurve
	surface geom.BSplineSurface
	edge    geom.Boundary
}

// endpoints of a boundary curve.
func (b boundaryEdge) start() math.Point3 { return b.curve.Ctrl[0] }
func (b boundaryEdge) end() math.Point3   { return b.curve.Ctrl[len(b.curve.Ctrl)-1] }

// FillFourSided fills the opening bounded by four neighbour surface bodies with a single NURBS at the
// given continuity order (0=G0..2=G2). Each neighbour must be a single NURBS face; its inner edge
// (nearest the opening centre) bounds the fill. It errors when a neighbour is not a NURBS face or the
// four edges do not chain into a closed loop.
func FillFourSided(neighbours [4]*topo.Body, order int) (*topo.Body, error) {
	edges, err := openingEdges(neighbours)
	if err != nil {
		return nil, err
	}
	c0, c1, d0, d1, err := chainLoop(edges)
	if err != nil {
		return nil, err
	}
	sides := [4]geom.FillSide{
		{Adjacent: c0.surface, AdjEdge: c0.edge, Order: order},
		{Adjacent: c1.surface, AdjEdge: c1.edge, Order: order},
		{Adjacent: d0.surface, AdjEdge: d0.edge, Order: order},
		{Adjacent: d1.surface, AdjEdge: d1.edge, Order: order},
	}
	fill, err := geom.FillSurface(c0.curve, c1.curve, d0.curve, d1.curve, sides)
	if err != nil {
		return nil, fmt.Errorf("ops.FillFourSided: %w", err)
	}
	return fullDomainBody(fill, "fill"), nil
}

// openingEdges returns each neighbour's inner boundary edge (midpoint nearest the centre of all
// neighbour centroids).
func openingEdges(neighbours [4]*topo.Body) ([4]boundaryEdge, error) {
	var edges [4]boundaryEdge
	var surfs [4]geom.BSplineSurface
	var sum math.Vector3
	for i, b := range neighbours {
		_, s, ok := firstNurbsFace(b)
		if !ok {
			return edges, fmt.Errorf("ops.FillFourSided: neighbour %d is not a NURBS face", i)
		}
		surfs[i] = s
		sum = sum.Add(s.PointAt(0.5, 0.5).AsVector())
	}
	center := sum.Scale(0.25).AsPoint()
	for i, s := range surfs {
		edge := innerEdge(s, center)
		edges[i] = boundaryEdge{curve: edgeCurve(s, edge), surface: s, edge: edge}
	}
	return edges, nil
}

// chainLoop orders the four boundary edges into c0 (v=0), c1 (v=1), d0 (u=0), d1 (u=1) with curves
// reversed so the corners chain (c0.start=d0.start, c0.end=d1.start, c1.start=d0.end, c1.end=d1.end).
func chainLoop(edges [4]boundaryEdge) (c0, c1, d0, d1 boundaryEdge, err error) {
	c0 = edges[0]
	rest := []boundaryEdge{edges[1], edges[2], edges[3]}
	d0, ok0 := takeSharing(&rest, c0.start())
	d1, ok1 := takeSharing(&rest, c0.end())
	if !ok0 || !ok1 || len(rest) != 1 {
		return c0, c1, d0, d1, fmt.Errorf("ops.FillFourSided: the four edges do not form a closed loop")
	}
	c1 = rest[0]
	d0 = orient(d0, c0.start()) // d0.start = corner00
	d1 = orient(d1, c0.end())   // d1.start = corner10
	c1 = orient(c1, d0.end())   // c1.start = corner01 (= d0.end)
	return c0, c1, d0, d1, nil
}

// takeSharing removes and returns the edge from rest that shares an endpoint with p (within tol).
func takeSharing(rest *[]boundaryEdge, p math.Point3) (boundaryEdge, bool) {
	for i, e := range *rest {
		if e.start().IsEqualTo(p, 1e-7) || e.end().IsEqualTo(p, 1e-7) {
			*rest = append((*rest)[:i], (*rest)[i+1:]...)
			return e, true
		}
	}
	return boundaryEdge{}, false
}

// orient returns the edge with its curve reversed if needed so its start equals p.
func orient(e boundaryEdge, p math.Point3) boundaryEdge {
	if e.start().IsEqualTo(p, 1e-7) {
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
	for i := 0; i < n; i++ {
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
