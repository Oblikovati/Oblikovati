// SPDX-License-Identifier: GPL-2.0-only

package heal

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/retopo"
	"oblikovati.org/kernel/topo"
)

// Boundary-patch fill over an explicit 3D edge loop (#1867). Where FillNSided derives its bounding
// edges from whole neighbour bodies (their inner iso-edges), a boundary patch is handed the loop
// directly — a set of surface-body boundary edges, possibly non-planar. Each edge contributes its
// curve and, when it borders a NURBS face, that face for the tangent/curvature (G1/G2) match; the
// loop then feeds the same Coons + MatchSurface fill as the N-sided opening (fillFromBoundaryEdges).

// FillEdgeLoop fills the closed loop of surface-body edges with one NURBS at the requested continuity
// order (0=G0…2=G2), matching to each edge's adjacent surface face where it borders a NURBS surface.
// An edge bordering only a planar/other face fills position-only (G0) on its side. It errors when the
// edges do not chain into a single closed loop.
func FillEdgeLoop(edges []*topo.Edge, order int) (*topo.Body, error) {
	if len(edges) < 3 {
		return nil, fmt.Errorf("heal.FillEdgeLoop: needs at least 3 boundary edges, have %d", len(edges))
	}
	bes := make([]boundaryEdge, len(edges))
	for i, e := range edges {
		bes[i] = boundaryFromEdge(e)
	}
	return fillFromBoundaryEdges(bes, order)
}

// boundaryFromEdge builds the fill boundary for a topo edge: its curve, and — when the edge borders a
// NURBS surface face — that surface plus the nearest of its four iso-boundaries (so MatchSurface can
// impose tangent/curvature on that side). An edge bordering only planar/other faces fills G0.
func boundaryFromEdge(e *topo.Edge) boundaryEdge {
	curve := curveAsBSpline(e.Geometry())
	lo, hi := curve.Domain()
	mid := curve.PointAt((lo + hi) / 2)
	for _, f := range e.Faces() {
		if bs, ok := f.Geometry().(geom.BSplineSurface); ok {
			return boundaryEdge{curve: curve, surface: bs, edge: innerEdge(bs, mid), nurbs: true}
		}
	}
	return boundaryEdge{curve: curve}
}

// fillFromBoundaryEdges fills the opening bounded by the given inner boundary edges (each already
// carrying its adjacent surface for the continuity match) with one NURBS: chain into a loop, map onto
// four compatible quad sides (splitting when <4, merging when >4), then Coons + MatchSurface. It is
// the shared tail of both the N-sided-opening fill and the edge-loop boundary patch.
func fillFromBoundaryEdges(edges []boundaryEdge, order int) (*topo.Body, error) {
	loop, err := orderLoop(edges)
	if err != nil {
		return nil, err
	}
	for len(loop) < 4 {
		loop = splitLongestEdge(loop)
	}
	c0, c1, d0, d1, err := chainLoop(groupToFourSides(loop))
	if err != nil {
		return nil, fmt.Errorf("ops.fillFromBoundaryEdges: %w", err)
	}
	if err = makeSidesCompatible(&c0, &c1, &d0, &d1); err != nil {
		return nil, fmt.Errorf("ops.fillFromBoundaryEdges: %w", err)
	}
	sides := [4]geom.FillSide{fillSide(c0, order), fillSide(c1, order), fillSide(d0, order), fillSide(d1, order)}
	fill, err := geom.FillSurface(c0.curve, c1.curve, d0.curve, d1.curve, sides)
	if err != nil {
		return nil, fmt.Errorf("ops.fillFromBoundaryEdges: %w", err)
	}
	return retopo.FullDomainBody(fill, "fill"), nil
}
