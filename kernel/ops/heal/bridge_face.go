// SPDX-License-Identifier: GPL-2.0-only

package heal

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/internal/retopo"
	"oblikovati.org/kernel/topo"
)

// Bridging two surface bodies (M36-F09): connect the two NURBS faces' FACING edges (each nearest the
// other body) with a clean NURBS transition that meets each at the chosen continuity (G0/G1/G2). The
// bridge math is geom.BridgeSurface; here we pick and orient the two boundary iso-curves.

// BridgeBodies builds a bridge surface body between the NURBS faces of bodyA and bodyB, meeting A at
// orderA and B at orderB (0=G0, 1=G1, 2=G2). Each body's bridged edge is the boundary facing the
// other body. It errors when either body has no NURBS face or the bridge is invalid.
func BridgeBodies(bodyA, bodyB *topo.Body, orderA, orderB int) (*topo.Body, error) {
	_, sA, ok := probe.FirstNurbsFace(bodyA)
	if !ok {
		return nil, fmt.Errorf("heal.BridgeBodies: body A has no NURBS surface face")
	}
	_, sB, ok := probe.FirstNurbsFace(bodyB)
	if !ok {
		return nil, fmt.Errorf("heal.BridgeBodies: body B has no NURBS surface face")
	}
	cenA, cenB := sA.PointAt(0.5, 0.5), sB.PointAt(0.5, 0.5)
	edgeA := innerEdge(sA, cenB) // A's boundary facing B
	edgeB := innerEdge(sB, cenA) // B's boundary facing A
	cA := edgeCurve(sA, edgeA)
	cB := edgeCurve(sB, edgeB)
	if !sameCurveDirection(cA, cB) { // orient cB to run the same way as cA, so the bridge net does not twist
		cB = reverseCurve(cB)
	}
	br, err := geom.BridgeSurface(cA, cB, sA, sB, edgeA, edgeB, orderA, orderB)
	if err != nil {
		return nil, fmt.Errorf("heal.BridgeBodies: %w", err)
	}
	return retopo.FullDomainBody(br, "bridge"), nil
}

// sameDirection reports whether cB already runs the same way as cA (its start is nearer cA's start
// than cA's end is) — so the per-row bridge interpolation pairs corresponding points.
func sameCurveDirection(cA, cB geom.BSplineCurve) bool {
	aStart := cA.Ctrl[0]
	bStart := cB.Ctrl[0]
	bEnd := cB.Ctrl[len(cB.Ctrl)-1]
	return aStart.DistanceTo(bStart) <= aStart.DistanceTo(bEnd)
}
