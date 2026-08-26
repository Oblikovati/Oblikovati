// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"slices"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// UnifyCoplanarFaces merges edge-adjacent faces that share a supporting plane (identical
// outward normal AND offset) into a single polygonal face carrying correct outer + hole loops.
// It is the planar analogue of OCCT's ShapeUpgrade_UnifySameDomain and undoes the fragmentation
// ops.Facet leaves behind: that faceter emits ONE planar face per tessellation triangle, so a
// flat region (a shelled tray's 2 mm top-frame annulus) comes back shattered into a triangle fan
// laced with interior diagonals. Every such diagonal is shared by two coplanar triangles, so its
// two directed half-edges cancel here and the diagonal vanishes, leaving the flat region as the
// single face it always was. A body that carries a non-planar face is returned unchanged (this
// only reasons about planes); nil-safe.
//
// Motivation (Oblikovati#1693): a sequential pattern boolean re-facets the WHOLE running body
// whenever one analytic cylinder wall survives (a drilled hole), shattering untouched rim faces
// into triangles with spurious diagonals; a later rolling-ball fillet on that rim then produced a
// non-manifold solid (an edge used by four faces). Unifying the cage back into maximal coplanar
// faces stops the cascade at the source.
func UnifyCoplanarFaces(b *topo.Body, feat string) *topo.Body {
	if b == nil {
		return b
	}
	groups, ok := groupCoplanarFaces(b)
	if !ok {
		return b // a curved face is present: this pass only reasons about planes
	}
	var merged []subFace
	for gi := range groups {
		merged = append(merged, groups[gi].subFaces(feat, gi)...)
	}
	if len(merged) == 0 {
		return b
	}
	body, _, err := stitch(merged, nil)
	if err != nil || body == nil {
		return b // never regress a valid input on a rebuild failure
	}
	return body
}

// coplanarGroup is the set of faces sharing one supporting plane, with the outward normal they
// all carry (so a zero-thickness sheet's two sides — same plane, opposite normals — never merge).
type coplanarGroup struct {
	normal math.Vector3
	rings  []coplanarRing // every constituent face's loops, as ordered welded vertices
}

// coplanarRing is one face loop of a group, tagged as an outer or hole loop so its winding is
// preserved (outer keeps material on the left CCW, a hole keeps it CW) when a polygonal
// face-with-hole is re-unified — not only a triangle fan whose rings are all outer.
type coplanarRing struct {
	verts []ringVertex
	hole  bool
}

// ringVertex pairs a boundary vertex's identity with its point, so half-edge cancellation keys on
// stable vertex identity while loop geometry stays available for orientation/nesting.
type ringVertex struct {
	id uint64
	p  math.Point3
}

// coplanarKeyGrid quantises a normal component / plane offset when hashing faces to a supporting
// plane. It matches the planar weld family (arrange2d arrTol / the stitch grid): two faces whose
// normals differ by more than ~1e-6 or whose offsets differ by more than the stitch grid are
// genuinely different planes, while a float-wobbled re-derivation of one plane stays within it.
const coplanarKeyGrid = 1e-6 // tol:calibrated — coplanar-face hash; see arrange2d arrTol

// groupCoplanarFaces buckets a body's faces by supporting plane (signed normal + offset). ok is
// false the moment a non-planar face is seen — the caller then leaves the body untouched.
func groupCoplanarFaces(b *topo.Body) ([]coplanarGroup, bool) {
	byKey := map[[4]int64]int{}
	var groups []coplanarGroup
	for _, f := range b.Faces() {
		pl, planar := f.Geometry().(geom.Plane)
		if !planar {
			return nil, false
		}
		n := outwardNormal(f, pl)
		key := planeKey(n, n.Dot(pl.Origin.AsVector()))
		gi, seen := byKey[key]
		if !seen {
			gi = len(groups)
			byKey[key] = gi
			groups = append(groups, coplanarGroup{normal: n})
		}
		groups[gi].rings = append(groups[gi].rings, faceRings(f)...)
	}
	return groups, true
}

// outwardNormal is a face's outward normal: its plane normal flipped when the face is stored
// reversed relative to that plane.
func outwardNormal(f *topo.Face, pl geom.Plane) math.Vector3 {
	n := pl.Normal()
	if f.Reversed() {
		return n.Scale(-1)
	}
	return n
}

// planeKey hashes a supporting plane by its quantised signed normal and offset.
func planeKey(n math.Vector3, d math.Scalar) [4]int64 {
	q := func(v float64) int64 { return int64(stdmath.Round(v / coplanarKeyGrid)) }
	return [4]int64{q(float64(n.X)), q(float64(n.Y)), q(float64(n.Z)), q(float64(d))}
}

// faceRings returns a face's loops as ordered welded vertices (each directed edge's start
// vertex, in loop-traversal order), each tagged outer or hole so its winding is preserved.
func faceRings(f *topo.Face) []coplanarRing {
	var out []coplanarRing
	for _, lp := range f.Loops() {
		uses := lp.EdgeUses()
		ring := make([]ringVertex, 0, len(uses))
		for _, u := range uses {
			v := u.Edge().StartVertex()
			if u.Reversed() {
				v = u.Edge().EndVertex()
			}
			ring = append(ring, ringVertex{id: v.ID(), p: v.Point()})
		}
		if len(ring) >= 3 {
			out = append(out, coplanarRing{verts: ring, hole: !lp.IsOuter()})
		}
	}
	return out
}

// subFaces reconstructs the group's merged faces: it orients every constituent ring to leave
// material on its left, cancels the interior half-edges two coplanar faces share, chains the
// survivors into boundary loops, and nests holes inside their outer loop.
func (g coplanarGroup) subFaces(feat string, gi int) []subFace {
	net := map[[2]uint64]int{}
	pts := map[uint64]math.Point3{}
	for _, ring := range g.rings {
		oriented := orientRingVerts(ring.verts, g.normal, ring.hole)
		for i := range oriented {
			a, b := oriented[i], oriented[(i+1)%len(oriented)]
			pts[a.id], pts[b.id] = a.p, b.p
			net[[2]uint64{a.id, b.id}]++
			net[[2]uint64{b.id, a.id}]--
		}
	}
	loops := chainBoundary(net, pts)
	return nestCoplanarLoops(loops, g.normal, feat, gi)
}

// orientRingVerts winds a ring so the face material sits on the LEFT of every directed edge:
// an outer loop counter-clockwise (Newell·normal > 0), a hole clockwise (Newell·normal < 0),
// both relative to the group's outward normal. That shared convention makes two coplanar faces'
// common interior edge run in opposite directions so it cancels, and preserves a hole's winding
// when a polygonal face-with-hole (not only a triangle fan) is re-unified.
func orientRingVerts(ring []ringVertex, n math.Vector3, hole bool) []ringVertex {
	pts := make([]math.Point3, len(ring))
	for i, rv := range ring {
		pts[i] = rv.p
	}
	ccw := newell3(pts).Dot(n) >= 0
	if ccw == hole { // outer wanting CW, or hole wanting CCW: reverse
		out := make([]ringVertex, len(ring))
		for i, rv := range ring {
			out[len(ring)-1-i] = rv
		}
		return out
	}
	return ring
}

// chainBoundary walks the surviving directed half-edges (net use count > 0) into closed vertex
// loops. Every boundary vertex has matched in/out degree on a manifold cage, so the walk consumes
// each survivor exactly once and closes each loop.
//
// Both the adjacency build and the start selection run in SORTED id order. Go randomizes map
// iteration, and at a vertex where two boundary loops touch (out-degree > 1) BOTH the order the
// candidates were appended in and the vertex the walk starts from decide which outgoing edge pairs
// with which incoming one — so the loops themselves, not merely their order, varied run to run.
// ops.Facet unifies coplanar faces when it re-facets an analytic operand for the planar boolean, so
// that fed a differently-connected tool into every rebuild: the same part alternated between solid
// and surface, and its volume wandered, on identical input (#23).
func chainBoundary(net map[[2]uint64]int, pts map[uint64]math.Point3) [][]math.Point3 {
	next := map[uint64][]uint64{}
	for _, e := range sortedHalfEdges(net) {
		for c := net[e]; c > 0; c-- {
			next[e[0]] = append(next[e[0]], e[1])
		}
	}
	var loops [][]math.Point3
	for _, start := range sortedLoopStarts(next) {
		for len(next[start]) > 0 {
			loop := walkCoplanarLoop(start, next, pts)
			if len(loop) >= 3 {
				loops = append(loops, loop)
			}
		}
	}
	return loops
}

// sortedHalfEdges returns net's directed edges in ascending (from, to) order.
func sortedHalfEdges(net map[[2]uint64]int) [][2]uint64 {
	keys := make([][2]uint64, 0, len(net))
	for e := range net {
		keys = append(keys, e)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})
	return keys
}

// sortedLoopStarts returns the vertices carrying outgoing half-edges, in ascending id order. The
// keys are fixed before the walk consumes them, so mutating next below cannot perturb the order.
func sortedLoopStarts(next map[uint64][]uint64) []uint64 {
	keys := make([]uint64, 0, len(next))
	for v := range next {
		keys = append(keys, v)
	}
	slices.Sort(keys)
	return keys
}

// walkCoplanarLoop follows the next-vertex adjacency from start until it returns, consuming each directed
// edge it uses.
func walkCoplanarLoop(start uint64, next map[uint64][]uint64, pts map[uint64]math.Point3) []math.Point3 {
	loop := []math.Point3{pts[start]}
	for cur := start; ; {
		outs := next[cur]
		if len(outs) == 0 {
			return nil // open chain (should not happen on a closed cage): drop it
		}
		nxt := outs[len(outs)-1]
		next[cur] = outs[:len(outs)-1]
		if nxt == start {
			return loop
		}
		loop = append(loop, pts[nxt])
		cur = nxt
	}
}

// nestCoplanarLoops splits a plane's boundary loops into outer loops and the hole loops nested
// inside them, emitting one subFace per outer loop. Outer loops wind CCW wrt the outward normal
// (positive Newell projection), holes CW; a hole is assigned to the outer loop that contains a
// point of it.
func nestCoplanarLoops(loops [][]math.Point3, n math.Vector3, feat string, gi int) []subFace {
	pl, err := planeForNormal(loops, n)
	if err != nil {
		return nil
	}
	var outers, holes [][]math.Point3
	for _, lp := range loops {
		if newell3(lp).Dot(n) >= 0 {
			outers = append(outers, lp)
		} else {
			holes = append(holes, lp)
		}
	}
	out := make([]subFace, len(outers))
	for i, o := range outers {
		out[i] = subFace{outer: o, normal: n, point: centroid3(o),
			lineage: topo.NewLineage(topo.Tok(feat, "coplanar", gi*1024+i))}
	}
	assignHoles(out, holes, pl)
	return out
}

// planeForNormal builds the group's supporting plane from its normal and any loop point.
func planeForNormal(loops [][]math.Point3, n math.Vector3) (geom.Plane, error) {
	if len(loops) == 0 || len(loops[0]) == 0 {
		return geom.Plane{}, errEmptyGroup
	}
	return geom.NewPlane(loops[0][0], n)
}

var errEmptyGroup = errString("brep: coplanar group has no boundary loop")

type errString string

func (e errString) Error() string { return string(e) }

// assignHoles places each hole loop inside the outer subface that contains it (a point-in-loop
// test on the shared plane), so a face with several openings keeps them all.
func assignHoles(outers []subFace, holes [][]math.Point3, pl geom.Plane) {
	for _, h := range holes {
		hp := to2D(pl, h[0])
		for oi := range outers {
			if pointInPolygon2D(hp, coplanarRing2D(pl, outers[oi].outer)) {
				outers[oi].holes = append(outers[oi].holes, h)
				break
			}
		}
	}
}

// coplanarRing2D projects a 3D loop to the plane's 2D frame.
func coplanarRing2D(pl geom.Plane, loop []math.Point3) []math.Point2 {
	out := make([]math.Point2, len(loop))
	for i, p := range loop {
		out[i] = to2D(pl, p)
	}
	return out
}
