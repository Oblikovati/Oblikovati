// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Weiler radial-edge manifold extraction — the tangent-contact core of the ONE boolean stitch
// (ADR-0047 #1726; surface-agnostic since ADR-0058, consumed by curved_stitch_plan.go for planar and
// curved faces alike).
//
// A boolean of two solids that touch along a lower-dimensional locus — a tangent/grazing contact —
// welds more than two face half-edges onto one geometric edge (a NON-manifold edge) and, at the
// contact's endpoints, more than one face fan onto one vertex (a pinch). Neither is a valid closed
// 2-manifold. This file is the PURE topological core that resolves both: around each non-manifold
// edge it radially orders the half-edge uses — by the dihedral azimuth of each face's outward surface
// normal at the edge, injected via faceDirAt (OCCT GetFaceOff/GetFaceDir) — and pairs the two
// boundaries of each filled dihedral wedge into manifold edge-groups (resolveEdgeUses); around each
// vertex it partitions the incident edge-groups into radial disks, each of which becomes one manifold
// vertex (partitionVertexDisks) — so two solids kissing along a line become two
// coincident-but-distinct shells.
//
// These are total functions over a naming-free plan. They never move a coordinate, never mint a topo
// entity and never name one (naming: boolean_mint.go hooks; construction: curved_stitch.go). A
// contact the radial sort cannot pair is encoded faithfully as an unpaired singleton group; the
// caller's solidity gate then declines the body rather than shipping an invalid one (the CSG-fallback
// path). See ADR-0047.

// edgeGroup is one manifold edge extracted from a (possibly non-manifold) radial edge: exactly two
// half-edge uses that bound one filled dihedral wedge — or, degenerately, a lone unpairable use.
type edgeGroup struct {
	pair [2]int
	uses []loopEdgeUse
}

// vertexDisk is one radial disk-cycle of edge-groups incident to a welded vertex — one manifold
// vertex. copy 0 reuses the shared welded vertex; copy>0 is a coincident duplicate that separates a
// line/point kiss into its own shell (the generalization of the #1693 pinched-vertex fan split).
type vertexDisk struct {
	welded int
	copy   int
	groups []int // indices into sewPlan.groups incident to this disk at `welded`
}

// sewPlan is the naming-free combinatorial result of the radial-edge sew: the manifold edge-groups and
// the per-welded-vertex disk partition. buildCurvedStitchPlan assembles it as it walks the geometric
// edges, in first-encounter order — which is what the edge lineage ordinals are built from, so the
// walk's order is part of the contract and not an implementation detail.
type sewPlan struct {
	groups []edgeGroup
	disks  map[int][]vertexDisk
}

// faceDirAt returns the outward material direction of a half-edge use at a point on the shared edge —
// the surface normal of the using face there. It is the ONLY surface-dependent input to the radial sew;
// injecting it (OCCT's GetFaceDir) is what makes the Weiler resolution surface-agnostic: a planar face
// returns its constant normal, a curved face its normal evaluated on the surface at the edge (ADR-0058).
type faceDirAt func(h loopEdgeUse, edgePoint math.Point3) math.Vector3

// partitionVertexDisks groups, per welded vertex, its incident edge-groups into radial disks: two
// groups share a disk iff some face uses both at that vertex (groupFans). A manifold vertex yields
// one disk; a pinch (a line/point kiss) yields one per touching shell. Vertices are keyed in sorted
// order so the duplicate-vertex lineage the mint step assigns is deterministic.
func partitionVertexDisks(groups []edgeGroup) map[int][]vertexDisk {
	incident := map[int][]int{}
	for gi := range groups {
		incident[groups[gi].pair[0]] = append(incident[groups[gi].pair[0]], gi)
		if groups[gi].pair[1] != groups[gi].pair[0] { // a closed seam edge touches its vertex once
			incident[groups[gi].pair[1]] = append(incident[groups[gi].pair[1]], gi)
		}
	}
	disks := make(map[int][]vertexDisk, len(incident))
	for v, inc := range incident {
		for copyIdx, fan := range groupFans(groups, inc) {
			disks[v] = append(disks[v], vertexDisk{welded: v, copy: copyIdx, groups: fan})
		}
	}
	return disks
}

// groupFans partitions a vertex's incident edge-groups into disks connected by shared LOOPS: two
// groups join iff some face loop uses both at that vertex. A clean manifold vertex yields one disk.
//
// The connector is the loop, not the face. A face whose boundary passes through one vertex on TWO of
// its loops is pinched there — the two loops are two separate fans on that face, exactly as two faces
// kissing at a point are two fans on the body — and joining them by face identity alone merged them
// into one vertex. That is what a torus cut by a plane tangent to its inner equator produces: the two
// lobes of the figure-eight bound one torus face through two loops, so the pinch welded to a single
// vertex and the body came out with an odd Euler characteristic (V−E+2F−L = 1), which the validity
// gate rightly refuses. Two loops of one face are still unioned transitively wherever another face
// genuinely joins their groups, so this only ever REFINES the partition (ADR-0061).
func groupFans(groups []edgeGroup, inc []int) [][]int {
	return topo.ComponentGroups(inc, func(join func(a, b int)) {
		byLoop := map[[2]int]int{} // (face, ring) → first incident group seen using it
		for _, gi := range inc {
			for _, u := range groups[gi].uses {
				key := [2]int{u.face, u.ring}
				if first, ok := byLoop[key]; ok {
					join(gi, first)
				} else {
					byLoop[key] = gi
				}
			}
		}
	})
}

// resolveEdgeUses partitions a vertex pair's uses into manifold groups of two. The common case
// (two uses) is one shared edge. A pair used more is a tangent/grazing contact where surfaces touch
// along this edge: collapsing all uses onto one edge would make it non-manifold (>2 faces). The
// half-edges are sorted by the azimuth of their face-interior direction about the edge axis, then
// paired by filled wedge (pairTangentDihedrals) so each group is one manifold dihedral.
// resolveEdgeUses partitions ONE edge's uses into manifold groups of two. edgeAxis lazily supplies the
// edge's tangent direction and a point on it (the chord + midpoint for a straight edge, the curve tangent
// + midpoint for a curved one) — evaluated only for an over-used (tangent-contact) edge.
func resolveEdgeUses(uses []loopEdgeUse, edgeAxis func() (math.Vector3, math.Point3), normalAt faceDirAt) [][]loopEdgeUse {
	if len(uses) <= 2 {
		return [][]loopEdgeUse{uses}
	}
	axis, edgePoint := edgeAxis()
	u, v := perpBasis(axis)
	sort.SliceStable(uses, func(i, j int) bool {
		return edgeAzimuth(uses[i], axis, u, v, edgePoint, normalAt) < edgeAzimuth(uses[j], axis, u, v, edgePoint, normalAt)
	})
	return pairTangentDihedrals(uses)
}

// pairTangentDihedrals pairs the azimuth-sorted uses by filled dihedral wedge. The loop orientation
// already encodes the material side: a REVERSED use is the ENTER boundary of a filled wedge (the
// wedge lies on its +azimuth side) and a non-reversed use is an EXIT. Walking from each enter
// boundary to the next exit pairs the two boundaries of one filled wedge into a manifold dihedral —
// operand-agnostic. Both outcomes fall out of this single rule: a coplanar flush overlap fuses the
// two operands' continued surfaces (leaving no coincident-edge crack a re-weld would collapse),
// while a non-coplanar bowtie kiss pairs each operand's own dihedral, so the solids stay two
// coincident shells rather than a χ-odd pinch (ADR-0047, #1726).
func pairTangentDihedrals(uses []loopEdgeUse) [][]loopEdgeUse {
	used := make([]bool, len(uses))
	groups := make([][]loopEdgeUse, 0, len(uses)/2)
	for i := range uses {
		if used[i] || !uses[i].reversed {
			continue // pair FROM each enter boundary; an exit is claimed as some enter's partner
		}
		if j := nextFilledBoundary(uses, used, i); j >= 0 {
			groups = append(groups, []loopEdgeUse{uses[i], uses[j]})
			used[i], used[j] = true, true
		}
	}
	// Any half-edge left unpaired (an odd over-use from a near-degenerate contact) gets its
	// own edge so every use still resolves to a real edge — the body will then be open at that
	// edge, which the caller detects (not solid) and rejects, rather than crashing on a nil edge.
	for i := range uses {
		if !used[i] {
			groups = append(groups, []loopEdgeUse{uses[i]})
		}
	}
	return groups
}

// nextFilledBoundary returns the nearest following (cyclic) unused half-edge that closes the filled
// dihedral wedge opened by the ENTER boundary i — the next EXIT (a non-reversed use) in +azimuth
// order. The two boundaries of one filled wedge become one manifold edge. Returns -1 if none remain
// (an odd, unpairable over-use the caller rejects as non-solid).
func nextFilledBoundary(uses []loopEdgeUse, used []bool, i int) int {
	for d := 1; d < len(uses); d++ {
		j := (i + d) % len(uses)
		if used[j] || uses[j].reversed {
			continue
		}
		return j
	}
	return -1
}

// edgeAzimuth is the angle, about the edge axis, of a half-edge's face-interior direction —
// the unit normal crossed with the ring's traversal direction (which keeps face material on
// its left for both outer and hole loops). Used to order the faces radially around the edge.
func edgeAzimuth(h loopEdgeUse, axis, u, v math.Vector3, edgePoint math.Point3, normalAt faceDirAt) float64 {
	travel := axis
	if h.reversed {
		travel = axis.Scale(-1)
	}
	interior := normalAt(h, edgePoint).Cross(travel)
	return stdmath.Atan2(interior.Dot(v), interior.Dot(u))
}

// perpBasis returns two orthonormal vectors spanning the plane perpendicular to axis.
func perpBasis(axis math.Vector3) (math.Vector3, math.Vector3) {
	ref := math.V3(1, 0, 0)
	if stdmath.Abs(axis.X) > 0.9 {
		ref = math.V3(0, 1, 0)
	}
	u := axis.Cross(ref).AsUnit().AsVector()
	return u, axis.Cross(u)
}
