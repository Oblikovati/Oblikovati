// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
	"sort"
)

// Sketch connectivity from the geometry points' own incidence lists — the exact
// SketchPoint→geometry association, decoded from the node graph rather than guessed.
//
// A geometry Point2d node carries, right after its coordinates, a connectivity header
// (curveMarker | deg | deg | curveTailMark) followed by `deg` entity references — the ids of the
// curves that pass through the point. Two points that name the SAME reference are the endpoints
// of that curve. Rebuilding lines this way needs no creation-order rank-alignment (the earlier
// resolveByRefs mechanism, which mis-maps coordinates when the reference and point sets aren't a
// clean bijection) and — because each reference is a globally-unique edge id linking its two
// endpoints by coordinate — it reunites a profile even when Inventor splits it across the
// 800-byte cluster gap, the failure that left most real shafts as open chains.

// incPoint is a geometry point with the incidence references listed in its node — the curve ids
// through it (a shared id between two points is the line joining them).
type incPoint struct {
	p   Point2D
	inc []uint32
}

// collectIncidencePoints reads every geometry Point2d node with its incidence references. The
// node shape matches collectItems' point branch (nameless node, marker high bit clear, an exact
// 0x800000XX geometry tag at +20, coordinates at +24/+32 that are not a curve header), plus the
// connectivity header at +40 (curveMarker | deg | deg | curveTailMark) and `deg` refs at +56.
func collectIncidencePoints(seg []byte) []incPoint {
	var pts []incPoint
	forEachNamelessNode(seg, func(j int) {
		if j+56 > len(seg) ||
			binary.LittleEndian.Uint32(seg[j+20:])&0xFFFFFF00 != nodeTagBase ||
			binary.LittleEndian.Uint32(seg[j+24:]) == curveMarker ||
			binary.LittleEndian.Uint32(seg[j+40:]) != curveMarker ||
			binary.LittleEndian.Uint32(seg[j+52:]) != curveTailMark {
			return
		}
		x, y := f64(seg, j+24), f64(seg, j+32)
		if math.IsNaN(x) || math.IsNaN(y) || absf(x) > 1e4 || absf(y) > 1e4 {
			return
		}
		deg := int(binary.LittleEndian.Uint32(seg[j+44:]))
		if deg < 1 || deg > maxPointDegree {
			return
		}
		ip := incPoint{p: Point2D{x, y}}
		for k := 0; k < deg && j+56+k*4+4 <= len(seg); k++ {
			if w := binary.LittleEndian.Uint32(seg[j+56+k*4:]); w&0xFFFFFF00 == refBit {
				ip.inc = append(ip.inc, w&^refBit)
			}
		}
		pts = append(pts, ip)
	})
	return pts
}

// maxPointDegree bounds a sketch point's incidence count — a sane cap that rejects a node whose
// +44 word is not really a small connectivity degree (it is 1 at a chain end, 2 at a corner).
const maxPointDegree = 8

// edge is an undirected connection between two collected points, by their slice index.
type edge struct{ a, b int }

// LineProfiles reconstructs the part's straight-line sketches from point incidence: each maximal
// set of points connected through shared incidence references becomes one Sketch (Resolved=true).
// This is used for the revolve path, where a profile split across the 800-byte cluster gap must
// still reunite into one closed loop. Circles/arcs are not covered (they are not two-endpoint
// edges); a filleted profile whose arcs would masquerade as chords is excluded by hasArcNodes.
func LineProfiles(seg []byte) []Sketch {
	if hasArcNodes(seg) {
		return nil // an arc would decode as a straight chord here — decline rather than distort
	}
	pts := collectIncidencePoints(seg)
	edges := cleanEdges(pts)
	edges = append(edges, collinearEdges(pts, edges)...)
	return componentSketches(pts, edges)
}

// cleanEdges returns the lines whose incidence reference is shared by EXACTLY two points — an
// unambiguous straight edge between them. A reference shared by one point is an open end; one
// shared by three or more is a collinear-constraint group, handled by collinearEdges.
func cleanEdges(pts []incPoint) []edge {
	var edges []edge
	byRef := groupByRef(pts)
	refs := sortedRefs(byRef)
	for _, r := range refs {
		if g := byRef[r]; len(g) == 2 {
			edges = append(edges, edge{g[0], g[1]})
		}
	}
	return edges
}

// collinearEdges recovers the edges hidden inside collinear-constraint groups. Inventor makes the
// segments along one straight run share a vertical/horizontal constraint, so a reference there is
// listed by 3+ collinear points (the run's corners plus points already joined by other edges). The
// real missing edge joins the two points still short of degree two — but only when they are
// axis-aligned with the group and adjacent along it, so a spurious cross-run chord is never added.
func collinearEdges(pts []incPoint, have []edge) []edge {
	deg := make([]int, len(pts))
	for _, e := range have {
		deg[e.a]++
		deg[e.b]++
	}
	var add []edge
	byRef := groupByRef(pts)
	for _, r := range sortedRefs(byRef) {
		g := byRef[r]
		if len(g) < 3 || !axisAligned(pts, g) {
			continue
		}
		var under []int
		for _, i := range g {
			if deg[i] < 2 {
				under = append(under, i)
			}
		}
		if len(under) == 2 && adjacentAlong(pts, g, under[0], under[1]) {
			add = append(add, edge{under[0], under[1]})
			deg[under[0]]++
			deg[under[1]]++
		}
	}
	return add
}

// groupByRef maps each incidence reference to the indices of the points that name it.
func groupByRef(pts []incPoint) map[uint32][]int {
	byRef := map[uint32][]int{}
	for i, p := range pts {
		for _, r := range p.inc {
			byRef[r] = append(byRef[r], i)
		}
	}
	return byRef
}

// sortedRefs returns the map keys in ascending order so edge construction is deterministic.
func sortedRefs(byRef map[uint32][]int) []uint32 {
	refs := make([]uint32, 0, len(byRef))
	for r := range byRef {
		refs = append(refs, r)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i] < refs[j] })
	return refs
}

// axisAligned reports whether every point in the group shares one X (a vertical run) or one Y (a
// horizontal run) — the shape of a vertical/horizontal-constraint group. A non-axis-aligned group
// (e.g. an arc's construction points) is left alone so collinearEdges never invents a wrong edge.
func axisAligned(pts []incPoint, g []int) bool {
	sameX, sameY := true, true
	for _, i := range g {
		if absf(pts[i].p.X-pts[g[0]].p.X) > 1e-3 {
			sameX = false
		}
		if absf(pts[i].p.Y-pts[g[0]].p.Y) > 1e-3 {
			sameY = false
		}
	}
	return sameX != sameY
}

// adjacentAlong reports whether points u and v are neighbours in the group's collinear order (no
// other group point lies strictly between them), so the recovered edge spans a single segment.
func adjacentAlong(pts []incPoint, g []int, u, v int) bool {
	ord := append([]int(nil), g...)
	sort.Slice(ord, func(i, j int) bool {
		a, b := pts[ord[i]].p, pts[ord[j]].p
		if absf(a.X-b.X) > 1e-9 {
			return a.X < b.X
		}
		return a.Y < b.Y
	})
	iu, iv := indexOfInt(ord, u), indexOfInt(ord, v)
	if iu < 0 || iv < 0 {
		return false
	}
	return absf(float64(iu-iv)) == 1
}

func indexOfInt(s []int, v int) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

// componentSketches groups the edges into connected components (one per sketch) and returns each
// as a resolved line Sketch. Points that share a coincident coordinate are unioned first, so a
// component is a run of geometry connected either by an edge or by a shared vertex.
func componentSketches(pts []incPoint, edges []edge) []Sketch {
	// coin unions only COINCIDENT points (distinct point nodes at one corner) — used for vertex
	// degree, so a leaf vertex keeps its own identity. comp additionally unions edge endpoints —
	// used to group edges into connected components. Two separate maps matter: unioning edge
	// endpoints into the degree map would collapse a dangling leaf into the ring it dangles off,
	// hiding it from pruneLeaves (a diameter-dimension leaf survived that way).
	coin := newUnionFind(len(pts))
	for i := range pts {
		for j := i + 1; j < len(pts); j++ {
			if samePoint2D(pts[i].p, pts[j].p) {
				coin.union(i, j)
			}
		}
	}
	comp := newUnionFind(len(pts))
	for i := range pts {
		comp.union(i, coin.find(i))
	}
	for _, e := range edges {
		comp.union(e.a, e.b)
	}
	byRoot := map[int][]edge{}
	for _, e := range edges {
		byRoot[comp.find(e.a)] = append(byRoot[comp.find(e.a)], e)
	}
	roots := make([]int, 0, len(byRoot))
	for r := range byRoot {
		roots = append(roots, r)
	}
	sort.Ints(roots)
	var out []Sketch
	for _, r := range roots {
		var lines []Line
		for _, e := range pruneLeaves(byRoot[r], coin) {
			lines = append(lines, Line{A: pts[e.a].p, B: pts[e.b].p})
		}
		if len(lines) > 0 {
			out = append(out, Sketch{Lines: lines, Resolved: true})
		}
	}
	return out
}

// pruneLeaves removes dangling edges — those with a vertex reached by no other edge — from a
// component, leaving its closed ring. A dimension's witness point produces such a leaf (a diameter
// dimension adds a mirrored point joined to a profile corner by a single edge), which would
// otherwise make the ring non-simple and force the honest revolve gate to reject the whole
// profile. A lone single-edge component (a separate centreline sketch) is left intact — it has no
// ring to isolate and is the axis case B needs — so pruning stops once one edge remains. Degree is
// counted per coincident vertex (union-find root) so distinct point nodes at one corner count once.
func pruneLeaves(edges []edge, uf *unionFind) []edge {
	for len(edges) > 1 {
		deg := map[int]int{}
		for _, e := range edges {
			deg[uf.find(e.a)]++
			deg[uf.find(e.b)]++
		}
		kept := make([]edge, 0, len(edges))
		for _, e := range edges {
			if deg[uf.find(e.a)] >= 2 && deg[uf.find(e.b)] >= 2 {
				kept = append(kept, e)
			}
		}
		if len(kept) == len(edges) {
			break
		}
		edges = kept
	}
	return edges
}

// hasArcNodes reports whether any decoded curve is an arc — a curve with two endpoint refs, a
// centre ref, and a positive radius. LineProfiles declines an arc-bearing part so a fillet arc is
// never silently emitted as a straight chord (tessellation-correctness: no plausible-but-wrong solid).
func hasArcNodes(seg []byte) bool {
	for _, it := range collectItems(seg) {
		if it.arc != nil {
			return true
		}
	}
	return false
}

// unionFind is a tiny disjoint-set over point indices for grouping edges into sketches.
type unionFind struct{ parent []int }

func newUnionFind(n int) *unionFind {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	return &unionFind{parent: p}
}

func (u *unionFind) find(x int) int {
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]]
		x = u.parent[x]
	}
	return x
}

func (u *unionFind) union(a, b int) { u.parent[u.find(a)] = u.find(b) }
