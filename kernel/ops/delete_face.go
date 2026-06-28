// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// DeleteFaces removes the selected faces from a planar solid and heals the openings by
// extending the neighbouring faces until they meet again. Each vertex of a deleted face
// slides along the line of its two surviving neighbour planes to the nearest OTHER neighbour
// plane (the face the heal extends to meet); coincident results weld, collapsing the deleted
// face's loop — e.g. deleting a chamfer face restores the sharp edge. It errors when the
// heal does not produce a valid closed solid (a non-healable selection), so the feature can
// go Sick rather than ship an open body.
func DeleteFaces(solid *topo.Body, faceKeys [][]byte) (*topo.Body, error) {
	del, err := resolveFaceSet(solid, faceKeys)
	if err != nil {
		return nil, err
	}
	moved := healedPositions(solid, del)
	body := buildSolidFromLoops(survivingLoops(solid, del, moved))
	if r := Validate(body); !r.Valid || !body.IsSolid() {
		return nil, fmt.Errorf("delete-face: heal did not close the body %v", r.Issues)
	}
	// Provenance (ADR-0043): a surviving edge the heal carried through unchanged keeps its original
	// identity rather than the rebuild's build-order name; the new healed edges keep the fallback.
	body.InheritOriginalEdges(solid.Edges())
	return body, nil
}

// healedPositions returns, per vertex touching a deleted face, the position it heals to.
func healedPositions(solid *topo.Body, del map[uint64]bool) map[uint64]math.Point3 {
	vf := vertexFaceMap(solid)
	ring := ringPlanes(solid, del)
	moved := map[uint64]math.Point3{}
	for _, v := range solid.Vertices() {
		if touchesDeleted(vf[v.ID()], del) {
			moved[v.ID()] = healedVertex(v, vf[v.ID()], del, ring)
		}
	}
	return moved
}

// touchesDeleted reports whether any face at the vertex is being deleted.
func touchesDeleted(faces []*topo.Face, del map[uint64]bool) bool {
	for _, f := range faces {
		if del[f.ID()] {
			return true
		}
	}
	return false
}

// ringPlanes returns the planes of every face that neighbours a deleted face (shares an
// edge) but is not itself deleted — the surfaces the heal extends to.
func ringPlanes(solid *topo.Body, del map[uint64]bool) []geom.Plane {
	seen := map[uint64]bool{}
	var out []geom.Plane
	for _, f := range solid.Faces() {
		if !del[f.ID()] {
			continue
		}
		for _, e := range f.Edges() {
			for _, nb := range e.Faces() {
				if del[nb.ID()] || seen[nb.ID()] {
					continue
				}
				seen[nb.ID()] = true
				out = append(out, nb.Geometry().(geom.Plane))
			}
		}
	}
	return out
}

// healedVertex returns where a vertex on a deleted face moves: the meet of its surviving
// face planes when ≥3 still pin it; otherwise it slides along the line of its two surviving
// planes to the nearest ring plane (the neighbour the heal extends to meet).
func healedVertex(v *topo.Vertex, faces []*topo.Face, del map[uint64]bool, ring []geom.Plane) math.Point3 {
	survivors := survivingPlanes(faces, del)
	if len(survivors) >= 3 {
		if p, ok := meetOfPlanes(survivors); ok {
			return p
		}
	}
	if len(survivors) == 2 {
		if p, ok := slideToNearest(survivors[0], survivors[1], ring, v.Point()); ok {
			return p
		}
	}
	return v.Point()
}

// survivingPlanes returns the planes of the vertex's faces that are not deleted.
func survivingPlanes(faces []*topo.Face, del map[uint64]bool) []geom.Plane {
	var out []geom.Plane
	for _, f := range faces {
		if !del[f.ID()] {
			out = append(out, f.Geometry().(geom.Plane))
		}
	}
	return out
}

// slideToNearest intersects the line of planes a,b with each ring plane and returns the
// intersection nearest the original vertex (the face the heal extends to meet).
func slideToNearest(a, b geom.Plane, ring []geom.Plane, v math.Point3) (math.Point3, bool) {
	p0, dir, ok := twoPlaneLine(a, b)
	if !ok {
		return math.Point3{}, false
	}
	best, bestD, found := math.Point3{}, stdmath.Inf(1), false
	for _, c := range ring {
		t, hit := lineHitsPlane(p0, dir, c)
		if !hit {
			continue
		}
		p := p0.TranslateBy(dir.Scale(t))
		if d := p.DistanceTo(v); d < bestD {
			best, bestD, found = p, d, true
		}
	}
	return best, found
}

// meetOfPlanes returns the least-squares meeting point of ≥3 planes (exact for 3).
func meetOfPlanes(planes []geom.Plane) (math.Point3, bool) {
	var a [3][3]float64
	var b [3]float64
	for _, pl := range planes {
		n := pl.Normal()
		d := n.Dot(pl.Origin.AsVector())
		nv := [3]float64{n.X, n.Y, n.Z}
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				a[i][j] += nv[i] * nv[j]
			}
			b[i] += nv[i] * d
		}
	}
	x, ok := solve3(a, b)
	return math.P3(x[0], x[1], x[2]), ok
}

// twoPlaneLine returns a point and direction of the intersection line of two planes, or
// ok=false when they are parallel.
func twoPlaneLine(a, b geom.Plane) (math.Point3, math.Vector3, bool) {
	na, nb := a.Normal(), b.Normal()
	dir := na.Cross(nb)
	if dir.LengthSquared() < 1e-18 {
		return math.Point3{}, math.Vector3{}, false
	}
	da, db := na.Dot(a.Origin.AsVector()), nb.Dot(b.Origin.AsVector())
	num := nb.Cross(dir).Scale(da).Add(dir.Cross(na).Scale(db))
	return math.P3(0, 0, 0).TranslateBy(num.Scale(1 / dir.LengthSquared())), dir, true
}

// lineHitsPlane returns the parameter t where line p0+t·dir meets plane c.
func lineHitsPlane(p0 math.Point3, dir math.Vector3, c geom.Plane) (float64, bool) {
	n := c.Normal()
	den := dir.Dot(n)
	if stdmath.Abs(den) < singularSolveTol {
		return 0, false
	}
	return (n.Dot(c.Origin.AsVector()) - n.Dot(p0.AsVector())) / den, true
}

// ploop is a surviving face as 3D point rings (outer first) plus its normal, ready for
// welding into a body.
type ploop struct {
	normal  math.Vector3
	rings   [][]math.Point3
	lineage topo.Lineage // provenance: the surviving original face this loop came from (ADR-0043)
}

// survivingLoops returns every non-deleted face as a point-ring loop with its vertices
// remapped to their healed positions.
func survivingLoops(solid *topo.Body, del map[uint64]bool, moved map[uint64]math.Point3) []ploop {
	var out []ploop
	for _, f := range solid.Faces() {
		if del[f.ID()] {
			continue
		}
		pl := ploop{normal: f.Geometry().NormalAt(0, 0), lineage: f.Lineage()}
		for _, l := range f.Loops() {
			pl.rings = append(pl.rings, healedRing(l, moved))
		}
		out = append(out, pl)
	}
	return out
}

// healedRing returns a loop's vertices with healed positions substituted.
func healedRing(l *topo.Loop, moved map[uint64]math.Point3) []math.Point3 {
	var pts []math.Point3
	for _, u := range l.EdgeUses() {
		v := u.Edge().StartVertex()
		if u.Reversed() {
			v = u.Edge().EndVertex()
		}
		if p, ok := moved[v.ID()]; ok {
			pts = append(pts, p)
		} else {
			pts = append(pts, v.Point())
		}
	}
	return pts
}

// buildSolidFromLoops welds coincident loop points into a body, dropping the degenerate
// (zero-length) edges that a heal collapses. One shared edge per undirected vertex pair; a
// closed body (every edge used twice) is a solid.
func buildSolidFromLoops(faces []ploop) *topo.Body {
	var pts []math.Point3
	for _, f := range faces {
		for _, r := range f.rings {
			pts = append(pts, r...)
		}
	}
	w := newPointWelder(ResolutionForPoints(pts).Weld())
	rings := make([][][]int, len(faces))
	for i, f := range faces {
		for _, r := range f.rings {
			rings[i] = append(rings[i], dropRepeats(w.weldRing(r)))
		}
	}
	edgeUse := map[[2]int]int{}
	for _, fr := range rings {
		for _, r := range fr {
			for k := 0; k < len(r); k++ {
				edgeUse[canon2(r[k], r[(k+1)%len(r)])]++
			}
		}
	}
	return assembleLoops(w.points, faces, rings, edgeUse)
}

// assembleLoops builds the topo body from welded points, per-face loop index rings, and the
// edge-use counts (every edge twice ⇒ solid).
func assembleLoops(verts []math.Point3, faces []ploop, rings [][][]int, edgeUse map[[2]int]int) *topo.Body {
	solid := true
	for _, c := range edgeUse {
		if c != 2 {
			solid = false
		}
	}
	bld := topo.NewBuilder(solid, topo.NewLineage(topo.Tok("delface", "body", 0)))
	tv := make([]*topo.Vertex, len(verts))
	for i, p := range verts {
		tv[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok("delface", "v", i)))
	}
	edges := buildSharedEdges(bld, verts, tv, edgeUse)
	for fi, fr := range rings {
		specs := make([]topo.LoopSpec, 0, len(fr))
		for ri, r := range fr {
			specs = append(specs, indexLoop(ri == 0, r, edges))
		}
		surf, _ := geom.NewPlane(centroidPts(faces[fi].rings[0]), faces[fi].normal)
		// Provenance (ADR-0043): a surviving face keeps its original identity; fall back to the
		// build-order name only if a loop arrived without one.
		lin := faces[fi].lineage
		if len(lin.Key()) == 0 {
			lin = topo.NewLineage(topo.Tok("delface", "f", fi))
		}
		bld.AddFace(surf, lin, specs...)
	}
	return bld.Build()
}

// buildSharedEdges creates one shared edge per undirected vertex pair (sorted for stable
// lineage).
func buildSharedEdges(bld *topo.Builder, verts []math.Point3, tv []*topo.Vertex, edgeUse map[[2]int]int) map[[2]int]*topo.Edge {
	keys := make([][2]int, 0, len(edgeUse))
	for k := range edgeUse {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})
	edges := make(map[[2]int]*topo.Edge, len(keys))
	for i, k := range keys {
		edges[k] = bld.AddEdge(geom.NewLineSegment(verts[k[0]], verts[k[1]]), tv[k[0]], tv[k[1]], topo.NewLineage(topo.Tok("delface", "e", i)))
	}
	return edges
}

// indexLoop builds a face loop from a ring of welded vertex indices.
func indexLoop(outer bool, ring []int, edges map[[2]int]*topo.Edge) topo.LoopSpec {
	uses := make([]topo.Use, len(ring))
	for i := range ring {
		a, b := ring[i], ring[(i+1)%len(ring)]
		uses[i] = topo.Use{Edge: edges[canon2(a, b)], Reversed: a > b}
	}
	if outer {
		return topo.OuterLoop(uses...)
	}
	return topo.InnerLoop(uses...)
}

// canon2 orders an undirected vertex-index pair.
func canon2(a, b int) [2]int {
	if a < b {
		return [2]int{a, b}
	}
	return [2]int{b, a}
}

// dropRepeats removes consecutive (and wrap-around) duplicate indices — the degenerate edges
// a heal collapses.
func dropRepeats(r []int) []int {
	var out []int
	for _, x := range r {
		if len(out) == 0 || out[len(out)-1] != x {
			out = append(out, x)
		}
	}
	for len(out) > 1 && out[0] == out[len(out)-1] {
		out = out[:len(out)-1]
	}
	return out
}

// pointWelder merges coincident 3D points onto a shared index list, snapping to a
// model-relative weld grid (ADR-0042) the caller derives from the points' size.
type pointWelder struct {
	index  map[[3]int64]int
	points []math.Point3
	grid   float64
}

func newPointWelder(grid float64) *pointWelder {
	return &pointWelder{index: map[[3]int64]int{}, grid: grid}
}

func (w *pointWelder) add(p math.Point3) int {
	k := [3]int64{quantize(p.X, w.grid), quantize(p.Y, w.grid), quantize(p.Z, w.grid)}
	if i, ok := w.index[k]; ok {
		return i
	}
	w.index[k] = len(w.points)
	w.points = append(w.points, p)
	return len(w.points) - 1
}

func (w *pointWelder) weldRing(r []math.Point3) []int {
	out := make([]int, len(r))
	for i, p := range r {
		out[i] = w.add(p)
	}
	return out
}

// centroidPts averages a point set (a point on the face for its plane origin).
func centroidPts(pts []math.Point3) math.Point3 {
	var sx, sy, sz float64
	for _, p := range pts {
		sx, sy, sz = sx+p.X, sy+p.Y, sz+p.Z
	}
	n := float64(len(pts))
	return math.P3(sx/n, sy/n, sz/n)
}
