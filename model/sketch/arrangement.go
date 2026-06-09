// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Region detection is a planar-subdivision problem: the regions a feature can extrude
// are the minimal closed cells the sketch curves carve the plane into, so a curve that
// divides a shape (an arc splitting a rectangle, two crossing lines, overlapping
// circles) yields several regions, not one. The naive degree-2 chain walk could not do
// this. Instead every curve is faceted into straight segments (the same faceting the
// extrude prism uses — see curvesample.go), every crossing reduces to a segment
// crossing (geom.Segment2dIntersection), and the faceted segments form a planar graph
// (an "arrangement"); faces.go then extracts the cells.

// arrMergeTol is the distance under which two arrangement endpoints are treated as the
// same node. It is coarser than DefaultTolerance so a computed crossing point and a
// curve endpoint that land on it snap together despite float error.
const arrMergeTol = 1e-6

// arrEdge is one straight sub-segment of a faceted curve, between two merged nodes,
// remembering the sketch entity it came from (for the loop's entity list).
type arrEdge struct {
	u, v   int
	entity Entity
}

// arrangement is the planar graph: merged nodes (unique positions) and the straight
// sub-segments connecting them, with a coordinate index for node merging and a set to
// drop duplicate edges (which would break the rotational face walk).
type arrangement struct {
	nodes []math.Point2
	edges []arrEdge
	index map[[2]int64]int
	seen  map[[2]int]bool
}

// buildArrangement facets the entities, splits every facet at its crossings with other
// entities, and merges coincident endpoints into a planar graph.
func buildArrangement(entities []Entity) *arrangement {
	segs := collectSegments(entities)
	cuts := intersectionCuts(segs)
	a := &arrangement{index: map[[2]int64]int{}, seen: map[[2]int]bool{}}
	for i, s := range segs {
		pts := splitPoints(s, cuts[i])
		for k := 0; k+1 < len(pts); k++ {
			a.addEdge(pts[k], pts[k+1], s.entity)
		}
	}
	return a
}

// taggedSeg is a single faceted segment of an entity (the unit the arrangement is
// built from).
type taggedSeg struct {
	a, b   math.Point2
	entity Entity
}

// collectSegments facets every entity into straight segments (closed curves wrap back
// to their first point), each tagged with its source entity.
func collectSegments(entities []Entity) []taggedSeg {
	var out []taggedSeg
	for _, e := range entities {
		pts, closed := entityPolyline(e)
		for k := 0; k+1 < len(pts); k++ {
			out = append(out, taggedSeg{pts[k], pts[k+1], e})
		}
		if closed && len(pts) >= 2 {
			out = append(out, taggedSeg{pts[len(pts)-1], pts[0], e})
		}
	}
	return out
}

// entityPolyline returns an entity's faceted polyline and whether it is inherently
// closed (so the caller wraps it). Reuses the curve samplers in curvesample.go.
func entityPolyline(e Entity) (pts []math.Point2, closed bool) {
	switch t := e.(type) {
	case *Circle:
		return sampleCircle(t), true
	case *Ellipse:
		return sampleEllipseEntity(t), true
	case *Spline:
		if t.Closed {
			return sampleSplineEntity(t), true
		}
		return naturalPolyline(t), false
	default:
		return naturalPolyline(e), false // Line, Arc, open Spline
	}
}

// cut is a crossing along a segment: the parameter t∈[0,1] where it occurs and the
// canonical crossing point (shared by both segments so they merge to one node).
type cut struct {
	param float64
	pt    math.Point2
}

// intersectionCuts finds, for every segment, the points where other entities' segments
// cross it. Facets of the same entity are skipped (they only share endpoints, they do
// not cross), which also avoids O(n²) self-pairs within one curve.
func intersectionCuts(segs []taggedSeg) [][]cut {
	cuts := make([][]cut, len(segs))
	for i := 0; i < len(segs); i++ {
		for j := i + 1; j < len(segs); j++ {
			if segs[i].entity == segs[j].entity {
				continue
			}
			pt, s, t, ok := geom.Segment2dIntersection(
				geom.NewLineSegment2d(segs[i].a, segs[i].b),
				geom.NewLineSegment2d(segs[j].a, segs[j].b), arrMergeTol,
			)
			if !ok {
				continue
			}
			cuts[i] = append(cuts[i], cut{s, pt})
			cuts[j] = append(cuts[j], cut{t, pt})
		}
	}
	return cuts
}

// splitPoints returns a segment's points in order: its start, the interior crossing
// points (sorted, endpoint-coincident ones dropped), and its end.
func splitPoints(s taggedSeg, cuts []cut) []math.Point2 {
	sort.Slice(cuts, func(i, j int) bool { return cuts[i].param < cuts[j].param })
	pts := []math.Point2{s.a}
	for _, c := range cuts {
		if c.param <= arrMergeTol || c.param >= 1-arrMergeTol {
			continue // a crossing at an endpoint splits nothing
		}
		if pts[len(pts)-1].IsEqualTo(c.pt, arrMergeTol) {
			continue
		}
		pts = append(pts, c.pt)
	}
	return append(pts, s.b)
}

// node returns the index of the merged node at p, creating it on first sight. Points
// within arrMergeTol quantize to the same grid cell and so share a node.
func (a *arrangement) node(p math.Point2) int {
	key := [2]int64{
		int64(stdmath.Round(p.X / arrMergeTol)),
		int64(stdmath.Round(p.Y / arrMergeTol)),
	}
	if id, ok := a.index[key]; ok {
		return id
	}
	id := len(a.nodes)
	a.nodes = append(a.nodes, p)
	a.index[key] = id
	return id
}

// addEdge adds an undirected edge between the nodes at p and q, dropping zero-length
// and duplicate edges (a repeated node pair would give the face walk two identical
// rotational directions).
func (a *arrangement) addEdge(p, q math.Point2, e Entity) {
	u, v := a.node(p), a.node(q)
	if u == v {
		return
	}
	key := [2]int{u, v}
	if u > v {
		key = [2]int{v, u}
	}
	if a.seen[key] {
		return
	}
	a.seen[key] = true
	a.edges = append(a.edges, arrEdge{u, v, e})
}
