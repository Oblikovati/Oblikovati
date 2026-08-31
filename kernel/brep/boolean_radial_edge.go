// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Weiler radial-edge manifold extraction for the planar-boolean stitch (ADR-0047, #1726).
//
// A boolean of two solids that touch along a lower-dimensional locus — a tangent/grazing contact —
// welds more than two face half-edges onto one vertex pair (a NON-manifold edge) and, at the
// contact's endpoints, more than one face fan onto one vertex (a pinch). Neither is a valid closed
// 2-manifold. This file is the PURE topological core that resolves both: around each non-manifold
// edge it radially orders the half-edge uses and pairs the two boundaries of each filled dihedral
// wedge into manifold edge-groups (resolveEdgeUses); around each vertex it partitions the incident
// edge-groups into radial disks, each of which becomes one manifold vertex (partitionVertexDisks) —
// so two solids kissing along a line become two coincident-but-distinct shells.
//
// It is a total function over a naming-free plan: it takes welded vertices, built faces and the
// edge-use map and returns a sewPlan. It never moves a coordinate, never mints a topo entity and
// never names one (that is boolean_mint.go's job, downstream). A contact the radial sort cannot pair
// is encoded faithfully as an unpaired singleton group; the caller's solidity gate then declines the
// body rather than shipping an invalid one (the CSG-fallback path). See ADR-0047.

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

// sewPlan is the naming-free combinatorial result of the radial-edge sew: the manifold edge-groups,
// the per-welded-vertex disk partition, and the map from each half-edge use to its group.
type sewPlan struct {
	groups   []edgeGroup
	disks    map[int][]vertexDisk
	useGroup map[[3]int]int
}

// faceDirAt returns the outward material direction of a half-edge use at a point on the shared edge —
// the surface normal of the using face there. It is the ONLY surface-dependent input to the radial sew;
// injecting it (OCCT's GetFaceDir) is what makes the Weiler resolution surface-agnostic: a planar face
// returns its constant normal, a curved face its normal evaluated on the surface at the edge (ADR-0058).
type faceDirAt func(h loopEdgeUse, edgePoint math.Point3) math.Vector3

// radialSew resolves every tangent/grazing contact in the welded face set into a manifold sew plan.
// The common case (every vertex pair used exactly twice, every vertex one disk) passes through
// unchanged — the radial machinery engages only where a pair is over-used or a vertex holds more
// than one disk. normalAt supplies each using face's surface normal at the edge, so the resolution is
// surface-agnostic (planar and curved alike).
func radialSew(verts []math.Point3, uses map[[2]int][]loopEdgeUse, normalAt faceDirAt) sewPlan {
	groups := extractEdgeGroups(verts, uses, normalAt)
	return sewPlan{
		groups:   groups,
		disks:    partitionVertexDisks(groups),
		useGroup: indexUsesByGroup(groups),
	}
}

// planarFaceDir is the faceDirAt for the planar boolean: a planar face's constant outward normal (the
// edge point is unused). Passing this reproduces the pre-ADR-0058 azimuth bit-for-bit.
func planarFaceDir(faces []builtFace) faceDirAt {
	return func(h loopEdgeUse, _ math.Point3) math.Vector3 { return faces[h.face].normal }
}

// extractEdgeGroups walks the vertex pairs in sorted order and splits each into its manifold
// edge-groups (resolveEdgeUses), so a pair used twice yields one group and an over-used tangent
// contact yields one group per filled wedge. The order is deterministic (sorted pair keys) so the
// downstream ordinal naming is stable.
func extractEdgeGroups(verts []math.Point3, uses map[[2]int][]loopEdgeUse, normalAt faceDirAt) []edgeGroup {
	var groups []edgeGroup
	for _, k := range sortedPairKeys(uses) {
		for _, g := range resolveEdgeUses(k, uses[k], verts, normalAt) {
			groups = append(groups, edgeGroup{pair: k, uses: g})
		}
	}
	return groups
}

// indexUsesByGroup maps each half-edge use (face,ring,pos) to the index of the edge-group carrying
// it, so the loop builder can resolve every directed loop edge to its shared topo edge.
func indexUsesByGroup(groups []edgeGroup) map[[3]int]int {
	m := make(map[[3]int]int)
	for gi := range groups {
		for _, h := range groups[gi].uses {
			m[[3]int{h.face, h.ring, h.pos}] = gi
		}
	}
	return m
}

// partitionVertexDisks groups, per welded vertex, its incident edge-groups into radial disks: two
// groups share a disk iff some face uses both at that vertex (groupFans). A manifold vertex yields
// one disk; a pinch (a line/point kiss) yields one per touching shell. Vertices are keyed in sorted
// order so the duplicate-vertex lineage the mint step assigns is deterministic.
func partitionVertexDisks(groups []edgeGroup) map[int][]vertexDisk {
	incident := map[int][]int{}
	for gi := range groups {
		incident[groups[gi].pair[0]] = append(incident[groups[gi].pair[0]], gi)
		incident[groups[gi].pair[1]] = append(incident[groups[gi].pair[1]], gi)
	}
	disks := make(map[int][]vertexDisk, len(incident))
	for v, inc := range incident {
		for copyIdx, fan := range groupFans(groups, inc) {
			disks[v] = append(disks[v], vertexDisk{welded: v, copy: copyIdx, groups: fan})
		}
	}
	return disks
}

// groupFans partitions a vertex's incident edge-groups into disks connected by shared faces: two
// groups join iff some face loop uses both at that vertex. A clean manifold vertex yields one disk.
func groupFans(groups []edgeGroup, inc []int) [][]int {
	return topo.ComponentGroups(inc, func(join func(a, b int)) {
		byFace := map[int]int{} // face → first incident group seen using it
		for _, gi := range inc {
			for _, u := range groups[gi].uses {
				if first, ok := byFace[u.face]; ok {
					join(gi, first)
				} else {
					byFace[u.face] = gi
				}
			}
		}
	})
}

// sortedPairKeys returns the vertex pairs in ascending order for stable edge lineage.
func sortedPairKeys(uses map[[2]int][]loopEdgeUse) [][2]int {
	keys := make([][2]int, 0, len(uses))
	for k := range uses {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})
	return keys
}

// resolveEdgeUses partitions a vertex pair's uses into manifold groups of two. The common case
// (two uses) is one shared edge. A pair used more is a tangent/grazing contact where surfaces touch
// along this edge: collapsing all uses onto one edge would make it non-manifold (>2 faces). The
// half-edges are sorted by the azimuth of their face-interior direction about the edge axis, then
// paired by filled wedge (pairTangentDihedrals) so each group is one manifold dihedral.
func resolveEdgeUses(pair [2]int, uses []loopEdgeUse, verts []math.Point3, normalAt faceDirAt) [][]loopEdgeUse {
	if len(uses) <= 2 {
		return [][]loopEdgeUse{uses}
	}
	p0, p1 := verts[pair[0]], verts[pair[1]]
	axis := p0.VectorTo(p1).AsUnit().AsVector()
	mid := math.P3((p0.X+p1.X)/2, (p0.Y+p1.Y)/2, (p0.Z+p1.Z)/2)
	u, v := perpBasis(axis)
	sort.SliceStable(uses, func(i, j int) bool {
		return edgeAzimuth(uses[i], axis, u, v, mid, normalAt) < edgeAzimuth(uses[j], axis, u, v, mid, normalAt)
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
