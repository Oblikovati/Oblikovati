// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"sort"

	"oblikovati/math"
)

// faces.go's job: turn the planar graph (arrangement.go) into the minimal closed cells
// (regions). It is a standard half-edge (DCEL) face traversal: each undirected edge
// becomes two opposed half-edges; at each node the outgoing half-edges are sorted by
// direction angle; the face boundary is followed by, at each arrival, taking the
// next-clockwise outgoing half-edge (twin then previous in CCW order). Every minimal
// bounded cell is traced once; the unbounded face of each component is the opposite
// orientation and is dropped by keeping only positive-area cycles.

// regionAreaEps discards near-degenerate cycles (collinear spurs) that enclose no area.
const regionAreaEps = 1e-9

// detectRegions returns the closed minimal cells of the sketch as loops, with the
// originating entities recorded per loop.
func detectRegions(entities []Entity) []Loop {
	a := buildArrangement(entities)
	edges := a.prunedEdges()
	if len(edges) == 0 {
		return nil
	}
	hes := a.buildHalfEdges(edges)
	return a.walkFaces(hes)
}

// prunedEdges drops dangling edges (those with a degree-1 endpoint) repeatedly, so
// open chains and antennae do not pollute the face walk; only edges that lie on a cycle
// survive.
func (a *arrangement) prunedEdges() []arrEdge {
	alive := make([]bool, len(a.edges))
	for i := range alive {
		alive[i] = true
	}
	for changed := true; changed; {
		changed = false
		deg := a.degrees(alive)
		for i, e := range a.edges {
			if alive[i] && (deg[e.u] < 2 || deg[e.v] < 2) {
				alive[i], changed = false, true
			}
		}
	}
	var out []arrEdge
	for i, e := range a.edges {
		if alive[i] {
			out = append(out, e)
		}
	}
	return out
}

// degrees counts the live-edge degree of every node.
func (a *arrangement) degrees(alive []bool) []int {
	deg := make([]int, len(a.nodes))
	for i, e := range a.edges {
		if alive[i] {
			deg[e.u]++
			deg[e.v]++
		}
	}
	return deg
}

// halfEdge is one directed side of an edge: origin/dest nodes, the outgoing direction
// angle (for rotational sorting), its twin, the next half-edge around the face, and the
// source entity (for the loop's entity list).
type halfEdge struct {
	origin, dest int
	angle        float64
	twin, next   int
	entity       Entity
}

// buildHalfEdges creates the two half-edges per edge, sorts each node's outgoing
// half-edges by angle, and links every half-edge's next around its face.
func (a *arrangement) buildHalfEdges(edges []arrEdge) []halfEdge {
	hes := make([]halfEdge, 0, 2*len(edges))
	out := map[int][]int{}
	for _, e := range edges {
		i := len(hes)
		hes = append(hes,
			halfEdge{origin: e.u, dest: e.v, angle: segAngle(a.nodes[e.u], a.nodes[e.v]), twin: i + 1, entity: e.entity},
			halfEdge{origin: e.v, dest: e.u, angle: segAngle(a.nodes[e.v], a.nodes[e.u]), twin: i, entity: e.entity})
		out[e.u] = append(out[e.u], i)
		out[e.v] = append(out[e.v], i+1)
	}
	for v := range out {
		sort.Slice(out[v], func(i, j int) bool { return hes[out[v][i]].angle < hes[out[v][j]].angle })
	}
	for h := range hes {
		hes[h].next = nextAroundFace(hes, out, h)
	}
	return hes
}

// nextAroundFace returns the half-edge that continues h's face boundary: arrive at
// dest along h, then leave on the outgoing half-edge just clockwise of h's twin (the
// previous one in the CCW-sorted rotation), which keeps the bounded cell on one side.
func nextAroundFace(hes []halfEdge, out map[int][]int, h int) int {
	list := out[hes[h].dest]
	i := indexOf(list, hes[h].twin)
	return list[(i-1+len(list))%len(list)]
}

// indexOf returns the position of v in s (or -1, which cannot occur here as the twin is
// always present in its origin's outgoing list).
func indexOf(s []int, v int) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

// walkFaces traces every face once and returns those that enclose positive area (the
// bounded minimal cells; each component's unbounded face is negative and dropped).
func (a *arrangement) walkFaces(hes []halfEdge) []Loop {
	visited := make([]bool, len(hes))
	var loops []Loop
	for start := range hes {
		if visited[start] {
			continue
		}
		cycle := traceFace(hes, visited, start)
		poly := a.cyclePolygon(hes, cycle)
		if signedArea2d(poly) > regionAreaEps {
			loops = append(loops, Loop{entities: cycleEntities(hes, cycle), polygon: poly, closed: true})
		}
	}
	return loops
}

// traceFace follows next() from start until it returns, marking each half-edge visited.
func traceFace(hes []halfEdge, visited []bool, start int) []int {
	var cycle []int
	for cur := start; !visited[cur]; cur = hes[cur].next {
		visited[cur] = true
		cycle = append(cycle, cur)
	}
	return cycle
}

// cyclePolygon returns the polygon of a face cycle (its half-edges' origin points).
func (a *arrangement) cyclePolygon(hes []halfEdge, cycle []int) []math.Point2 {
	poly := make([]math.Point2, len(cycle))
	for i, h := range cycle {
		poly[i] = a.nodes[hes[h].origin]
	}
	return poly
}

// cycleEntities returns the loop's entities, collapsing consecutive facets of the same
// entity (and the wrap-around) into one ProfileEntity. The reversed flag is best-effort
// (false): the entity list is informational; geometry flows through the polygon.
func cycleEntities(hes []halfEdge, cycle []int) []ProfileEntity {
	var ents []ProfileEntity
	for _, h := range cycle {
		e := hes[h].entity
		if len(ents) > 0 && ents[len(ents)-1].Entity == e {
			continue
		}
		ents = append(ents, ProfileEntity{Entity: e})
	}
	if len(ents) > 1 && ents[0].Entity == ents[len(ents)-1].Entity {
		ents = ents[:len(ents)-1] // the loop wrapped on the same entity
	}
	return ents
}

// segAngle returns the angle of the from→to direction.
func segAngle(from, to math.Point2) float64 {
	v := from.VectorTo(to)
	return stdmath.Atan2(v.Y, v.X)
}

// signedArea2d is the shoelace signed area (positive counter-clockwise).
func signedArea2d(poly []math.Point2) float64 {
	area := 0.0
	for i, n := 0, len(poly); i < n; i++ {
		j := (i + 1) % n
		area += poly[i].X*poly[j].Y - poly[j].X*poly[i].Y
	}
	return area / 2
}
