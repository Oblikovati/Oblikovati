// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// loopEdgeUse is one directed use of an undirected vertex pair by a face loop: the face/ring/pos
// that locate it, and whether the ring traverses the pair high→low (reversed, vs the canonical
// low→high orientation used to key the edge).
type loopEdgeUse struct {
	face, ring, pos int
	reversed        bool
	fromB           bool // operand of the using face, for cross-operand fusion of tangent contacts
}

// collectEdgeUses groups every directed loop edge by its canonical (low,high) vertex pair, so
// the stitch can resolve each pair's uses into shared topo edges.
func collectEdgeUses(faces []builtFace) map[[2]int][]loopEdgeUse {
	uses := map[[2]int][]loopEdgeUse{}
	for fi, f := range faces {
		for ri, r := range f.rings {
			n := len(r)
			for i := 0; i < n; i++ {
				a, b := r[i], r[(i+1)%n]
				key := canonEdge(a, b)
				uses[key] = append(uses[key], loopEdgeUse{face: fi, ring: ri, pos: i, reversed: a > b, fromB: f.fromB})
			}
		}
	}
	return uses
}

// allUsesPaired reports whether every vertex pair is used an even, non-zero number of times —
// the combinatorial closed-shell test. Exactly two is the manifold norm; more (a tangent
// contact) still closes once resolveEdgeUses splits it into twice-used edges, so even is the
// right test; an odd or zero count cannot close.
func allUsesPaired(uses map[[2]int][]loopEdgeUse) bool {
	for _, u := range uses {
		if len(u) == 0 || len(u)%2 != 0 {
			return false
		}
	}
	return true
}

// buildResolvedEdges creates the shared topo edges and maps each half-edge use (face,ring,pos)
// to its edge. A pair used twice makes one edge; a pair used more — coincident edges left by a
// tangent/grazing contact between the operands — is split by resolveEdgeUses into manifold
// pairs, each its own coincident edge between the same endpoints, so no edge ends up shared by
// more than two faces.
//
// Each edge is named by its GENERATING PARENTS when prov resolves it to an intersection of two
// faces (#1153) — a name invariant to the stitch's vertex ordering, unlike the ordinal index
// used as a fallback for edges with no provenance (original face boundaries, and callers that
// pass nil such as the drilled-hole path). Several edges sharing one parent pair (a face crossed
// twice) are disambiguated by their order ALONG the parents' intersection line, a
// transform-invariant characteristic (#1155) rather than a build-order counter.
func buildResolvedEdges(bld *topo.Builder, verts []math.Point3, tv []*topo.Vertex, uses map[[2]int][]loopEdgeUse, faces []builtFace, prov []imprintSeg) map[[3]int]*topo.Edge {
	builds := planEdgeBuilds(verts, uses, faces, prov)
	rankSamePairEdges(builds, prov)
	// Pinched vertices (a sub-grid contact patch, or the endpoints of resolved tangent-contact
	// edges) are cut apart into fan-specific coincident duplicates so the shell stays a true
	// closed 2-manifold (#1693); a manifold vertex resolves to its shared tv entry.
	endpoint := pinchedEndpoints(bld, verts, tv, builds)
	useEdge := make(map[[3]int]*topo.Edge)
	idx := 0
	for i := range builds {
		b := &builds[i]
		e := bld.AddEdge(geom.NewLineSegment(verts[b.k[0]], verts[b.k[1]]), endpoint[[2]int{i, b.k[0]}], endpoint[[2]int{i, b.k[1]}], edgeBuildLineage(b, &idx))
		for _, h := range b.group {
			useEdge[[3]int{h.face, h.ring, h.pos}] = e
		}
	}
	return useEdge
}

// edgeBuild is one planned topo edge: its vertex pair, the half-edge uses it carries, its
// midpoint, and the parent faces (when it is an intersection edge) plus its rank among edges of
// the same parent pair.
type edgeBuild struct {
	k        [2]int
	group    []loopEdgeUse
	mid      math.Point3
	lo, hi   topo.Lineage
	parented bool
	rank     int
}

// planEdgeBuilds enumerates the edges to create (resolving tangent contacts) with each one's
// parent pair and midpoint — the input the geometric disambiguator ranks.
func planEdgeBuilds(verts []math.Point3, uses map[[2]int][]loopEdgeUse, faces []builtFace, prov []imprintSeg) []edgeBuild {
	var builds []edgeBuild
	for _, k := range sortedPairKeys(uses) {
		for _, group := range resolveEdgeUses(k, uses[k], verts, faces) {
			p, q := verts[k[0]], verts[k[1]]
			lo, hi, ok := edgeParents(p, q, prov)
			builds = append(builds, edgeBuild{k: k, group: group, mid: p.TranslateBy(p.VectorTo(q).Scale(0.5)), lo: lo, hi: hi, parented: ok})
		}
	}
	return builds
}

// rankSamePairEdges assigns each parented edge its rank among edges sharing the same parent pair,
// ordered by the transform-invariant characteristic along the pair's intersection line. A lone
// edge of a pair keeps rank 0 (no disambiguator); the common case is therefore untouched.
func rankSamePairEdges(builds []edgeBuild, prov []imprintSeg) {
	groups := map[string][]int{}
	for i := range builds {
		if builds[i].parented {
			key := string(builds[i].lo.Key()) + "\x00" + string(builds[i].hi.Key())
			groups[key] = append(groups[key], i)
		}
	}
	for _, idxs := range groups {
		if len(idxs) < 2 {
			continue
		}
		d, ok := pairLineDir(builds[idxs[0]].lo, builds[idxs[0]].hi, prov)
		sort.SliceStable(idxs, func(a, b int) bool {
			return ok && lineCharacteristic(builds[idxs[a]].mid, d) < lineCharacteristic(builds[idxs[b]].mid, d)
		})
		for r, i := range idxs {
			builds[i].rank = r
		}
	}
}

// edgeBuildLineage is an edge's lineage: its parent-pair name (with the disambiguating rank) when
// it is an intersection edge, else the ordinal fallback (incrementing idx).
func edgeBuildLineage(b *edgeBuild, idx *int) topo.Lineage {
	if !b.parented {
		lin := topo.NewLineage(topo.Tok("brep", "edge", *idx))
		*idx++
		return lin
	}
	return intersectionLineage(b.lo, b.hi, b.rank)
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
// (two uses) is one shared edge. A pair used more is a tangent/grazing contact where the two
// operands' surfaces touch along this edge: collapsing all uses onto one edge would make it
// non-manifold (>2 faces). The half-edges are sorted by the azimuth of their face-interior
// direction about the edge axis, then paired (pairTangentDihedrals) so each group is one
// manifold dihedral — and, where it can, FUSING the operands (an A face with a B face) so the
// joined surface is continuous and leaves no coincident-edge crack for a later re-weld (a
// fillet/shell) to collapse back to non-manifold.
func resolveEdgeUses(pair [2]int, uses []loopEdgeUse, verts []math.Point3, faces []builtFace) [][]loopEdgeUse {
	if len(uses) <= 2 {
		return [][]loopEdgeUse{uses}
	}
	axis := verts[pair[0]].VectorTo(verts[pair[1]]).AsUnit().AsVector()
	u, v := perpBasis(axis)
	sort.SliceStable(uses, func(i, j int) bool {
		return edgeAzimuth(uses[i], axis, u, v, faces) < edgeAzimuth(uses[j], axis, u, v, faces)
	})
	return pairTangentDihedrals(uses)
}

// pairTangentDihedrals walks the azimuth-sorted uses and pairs each unpaired half-edge with
// the nearest following (cyclically) one of opposite traversal direction — so each group is a
// pair of faces meeting at a real dihedral (used once each way), manifold and closed. A
// cross-operand partner (one operand's face meeting the other's) is preferred over a
// same-operand one at equal reach: fusing the operands' surfaces into one continuous shell
// avoids a zero-width crack between two coincident edges that a downstream re-weld would merge.
func pairTangentDihedrals(uses []loopEdgeUse) [][]loopEdgeUse {
	used := make([]bool, len(uses))
	groups := make([][]loopEdgeUse, 0, len(uses)/2)
	for i := range uses {
		if used[i] {
			continue
		}
		if j := pickPartner(uses, used, i); j >= 0 {
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

// pickPartner returns the index of the half-edge to pair with use i: the nearest following
// (cyclic) one of opposite traversal direction, preferring a cross-operand partner so the two
// solids fuse. Returns -1 if none remain (an odd, unpairable count).
func pickPartner(uses []loopEdgeUse, used []bool, i int) int {
	fallback := -1
	for d := 1; d < len(uses); d++ {
		j := (i + d) % len(uses)
		if used[j] || uses[j].reversed == uses[i].reversed {
			continue
		}
		if uses[j].fromB != uses[i].fromB {
			return j // cross-operand: fuse the surfaces
		}
		if fallback < 0 {
			fallback = j
		}
	}
	return fallback
}

// edgeAzimuth is the angle, about the edge axis, of a half-edge's face-interior direction —
// the unit normal crossed with the ring's traversal direction (which keeps face material on
// its left for both outer and hole loops). Used to order the faces radially around the edge.
func edgeAzimuth(h loopEdgeUse, axis, u, v math.Vector3, faces []builtFace) float64 {
	travel := axis
	if h.reversed {
		travel = axis.Scale(-1)
	}
	interior := faces[h.face].normal.Cross(travel)
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

// loopSpecResolved builds a face loop from a ring of vertex indices, resolving each directed
// edge to the topo edge assigned to that exact use (so coincident, tangent-contact edges stay
// distinct), reversed when the ring traverses the pair high→low.
func loopSpecResolved(outer bool, ring []int, fi, ri int, useEdge map[[3]int]*topo.Edge) topo.LoopSpec {
	uses := make([]topo.Use, len(ring))
	for i := range ring {
		a, b := ring[i], ring[(i+1)%len(ring)]
		uses[i] = topo.Use{Edge: useEdge[[3]int{fi, ri, i}], Reversed: a > b}
	}
	if outer {
		return topo.OuterLoop(uses...)
	}
	return topo.InnerLoop(uses...)
}
