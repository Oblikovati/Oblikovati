// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// weldGrid is the coincidence grid for welding CSG output vertices (database units).
const weldGrid = 1e-6

// booleanInputQuality is the faceting the BSP-tree CSG meshes its operands at. It keeps the
// display chord tolerance but DISABLES the angular-deflection refinement (a huge angle bound
// → chord-only), because the faceted CSG fallback is numerically fragile: feeding it the
// finer angle-bounded display mesh (e.g. 32-facet small cylinders) makes BSP plane splits
// leave hairline cracks (a few unshared boundary edges). The boolean is a geometry op with
// its own robustness ceiling, separate from how smoothly we DISPLAY analytic surfaces — so it
// pins its input to the proven-robust chord-only resolution. (Analytic faces, including
// modeled threads and drilled holes, still render at full DefaultQuality smoothness.)
func booleanInputQuality() Quality {
	return Quality{ChordTolerance: DefaultQuality().ChordTolerance, AngleTolerance: stdmath.Pi}
}

// Facet converts a body with analytic curved faces (a cylinder, cone, surface of revolution)
// into an all-planar, watertight triangle-cage B-rep — its tessellation welded back into topology.
// The exact planar B-rep boolean hangs on a full periodic curved face, so a curved operand must be
// faceted before ops.Boolean (Oblikovati/Oblikovati#129); this is the general fallback for bodies
// the feature layer's analytic-cylinder re-faceter doesn't special-case. Returns nil when empty.
func Facet(b *topo.Body, feat string) *topo.Body {
	return trianglesToBody(bodyTriangles(b), feat)
}

// bodyTriangles returns a body's tessellation as CSG triangles, each oriented
// outward. The BSP-tree CSG depends on globally consistent outward winding, but
// TessellateBody does not guarantee it for curved faces (a cylinder's side
// triangles can wind either way — the same reason meshGeometryProperties
// re-orients before its volume sum). Trusting the raw winding silently breaks any
// boolean whose minuend is a curved body: the BSP misclassifies inside/outside and
// subtracts nothing (cylinder − tool returned the uncut cylinder). We fix the
// winding here with the per-vertex shading normals, which point outward.
func bodyTriangles(b *topo.Body) []tri {
	mesh, _ := TessellateBody(b, booleanInputQuality())
	var out []tri
	for i := 0; i+2 < len(mesh.Indices); i += 3 {
		ia, ib, ic := mesh.Indices[i], mesh.Indices[i+1], mesh.Indices[i+2]
		a, bb, c := mesh.Positions[ia], mesh.Positions[ib], mesh.Positions[ic]
		if outwardRef(mesh, ia, ib, ic).Dot(a.VectorTo(bb).Cross(a.VectorTo(c))) < 0 {
			bb, c = c, bb // flip to outward winding
		}
		if t, ok := newTri(a, bb, c); ok {
			out = append(out, t)
		}
	}
	return out
}

// trianglesToBody welds CSG output triangles into a watertight B-rep: coincident
// vertices merge, T-junctions are split out so the cage is combinatorially closed, and
// the welded triangle cage becomes a body. Returns nil when the result is empty.
func trianglesToBody(tris []tri, feat string) *topo.Body {
	verts, faces := weldTriangles(tris)
	faces = dedupTriangles(faces)
	if len(faces) == 0 {
		return nil
	}
	faces = removeTJunctions(verts, faces)
	return cageToBody(verts, dropDegenerate(faces), feat)
}

// dedupTriangles cancels coincident triangles produced where coplanar faces of the two
// operands overlap: a triangle and its reverse (opposite orientation) annihilate (an
// internal face); identical-orientation duplicates collapse to one. This keeps each
// surface patch represented exactly once so the weld is 2-manifold.
func dedupTriangles(faces [][3]int) [][3]int {
	type bal struct {
		fwd, rev int
		face     [3]int
	}
	seen := map[[3]int]*bal{}
	order := [][3]int{}
	for _, f := range faces {
		key, reversed := sortedTri(f)
		b := seen[key]
		if b == nil {
			b = &bal{face: f}
			seen[key] = b
			order = append(order, key)
		}
		if reversed {
			b.rev++
		} else {
			b.fwd++
		}
	}
	var out [][3]int
	for _, key := range order {
		b := seen[key]
		if net := b.fwd - b.rev; net != 0 { // surviving orientation wins; equal ⇒ cancelled
			out = append(out, b.face)
		}
	}
	return out
}

// sortedTri returns a triangle's canonical (ascending) vertex key and whether the given
// winding runs opposite the canonical cyclic order (i.e. it is the reversed face).
func sortedTri(f [3]int) ([3]int, bool) {
	a, b, c := f[0], f[1], f[2]
	even := (a < b && b < c) || (b < c && c < a) || (c < a && a < b) // cyclic-sorted ⇒ same orientation
	key := f
	if key[1] < key[0] {
		key[0], key[1] = key[1], key[0]
	}
	if key[2] < key[1] {
		key[1], key[2] = key[2], key[1]
	}
	if key[1] < key[0] {
		key[0], key[1] = key[1], key[0]
	}
	return key, !even
}

// weldTriangles merges coincident triangle corners onto a shared vertex list, dropping
// triangles that collapse to a degenerate (a repeated corner).
func weldTriangles(tris []tri) ([]math.Point3, [][3]int) {
	index := map[[3]int64]int{}
	var verts []math.Point3
	weld := func(p math.Point3) int {
		k := [3]int64{quantize(p.X), quantize(p.Y), quantize(p.Z)}
		if i, ok := index[k]; ok {
			return i
		}
		index[k] = len(verts)
		verts = append(verts, p)
		return len(verts) - 1
	}
	var faces [][3]int
	for _, t := range tris {
		a, b, c := weld(t.a), weld(t.b), weld(t.c)
		if a != b && b != c && a != c {
			faces = append(faces, [3]int{a, b, c})
		}
	}
	return verts, faces
}

func quantize(v float64) int64 { return int64(stdmath.Round(v / weldGrid)) }

// removeTJunctions repeatedly splits any triangle edge that another vertex lands on, so
// every undirected edge ends up shared by exactly two triangles (the prerequisite for a
// closed solid). Capped to avoid pathological loops.
func removeTJunctions(verts []math.Point3, faces [][3]int) [][3]int {
	for pass := 0; pass < 64; pass++ {
		split := false
		var next [][3]int
		for _, f := range faces {
			if rep, ok := splitFaceAtTJunction(verts, f); ok {
				next = append(next, rep...)
				split = true
			} else {
				next = append(next, f)
			}
		}
		faces = next
		if !split {
			break
		}
	}
	return faces
}

// splitFaceAtTJunction finds a vertex lying on one of the triangle's edges and splits
// the triangle across it into two triangles, reporting whether a split happened.
func splitFaceAtTJunction(verts []math.Point3, f [3]int) ([][3]int, bool) {
	for e := 0; e < 3; e++ {
		p, q := f[e], f[(e+1)%3]
		apex := f[(e+2)%3]
		for ci := range verts {
			if ci == p || ci == q || ci == apex {
				continue
			}
			if onSegment(verts[ci], verts[p], verts[q]) {
				return [][3]int{{p, ci, apex}, {ci, q, apex}}, true
			}
		}
	}
	return nil, false
}

// onSegment reports whether p lies on the interior of segment a→b: within onLineTol of
// the line and strictly between the endpoints (more than onLineTol from each). Endpoint
// exclusion is by absolute distance, not a parameter fraction, so a vertex near a long
// edge's end is still recognized as a T-junction.
const onLineTol = 1e-6

func onSegment(p, a, b math.Point3) bool {
	ab := a.VectorTo(b)
	lenSq := ab.LengthSquared()
	if lenSq == 0 {
		return false
	}
	t := a.VectorTo(p).Dot(ab) / lenSq
	if t <= 0 || t >= 1 {
		return false
	}
	foot := a.TranslateBy(ab.Scale(t))
	if foot.DistanceTo(p) >= onLineTol {
		return false
	}
	return p.DistanceTo(a) > onLineTol && p.DistanceTo(b) > onLineTol
}

// dropDegenerate removes triangles with a repeated corner.
func dropDegenerate(faces [][3]int) [][3]int {
	out := faces[:0]
	for _, f := range faces {
		if f[0] != f[1] && f[1] != f[2] && f[0] != f[2] {
			out = append(out, f)
		}
	}
	return out
}

// cageToBody assembles a B-rep from a welded triangle cage: one shared vertex per cage
// vertex, one shared edge per undirected edge (sorted for stable lineage), a planar face
// per triangle. The body is a solid when every edge is shared by exactly two triangles.
func cageToBody(verts []math.Point3, faces [][3]int, feat string) *topo.Body {
	if len(faces) == 0 {
		return nil
	}
	uses := edgeUseCounts(faces)
	bld := topo.NewBuilder(isClosedCage(uses), topo.NewLineage(topo.Tok(feat, "body", 0)))
	tv := make([]*topo.Vertex, len(verts))
	for i, p := range verts {
		tv[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	edges := buildCageEdges(bld, verts, tv, uses, feat)
	for fi, f := range faces {
		bld.AddFace(trianglePlane(verts, f), topo.NewLineage(topo.Tok(feat, "face", fi)), triangleLoop(f, edges))
	}
	return bld.Build()
}

func edgeKeyOf(a, b int) [2]int {
	if a < b {
		return [2]int{a, b}
	}
	return [2]int{b, a}
}

// edgeUseCounts counts how many triangles use each undirected edge.
func edgeUseCounts(faces [][3]int) map[[2]int]int {
	uses := map[[2]int]int{}
	for _, f := range faces {
		for i := 0; i < 3; i++ {
			uses[edgeKeyOf(f[i], f[(i+1)%3])]++
		}
	}
	return uses
}

func isClosedCage(uses map[[2]int]int) bool {
	for _, c := range uses {
		if c != 2 {
			return false
		}
	}
	return len(uses) > 0
}

// buildCageEdges creates one shared topo edge per undirected edge, in sorted-key order
// so the synthesized lineage (and reference keys) is stable.
func buildCageEdges(bld *topo.Builder, verts []math.Point3, tv []*topo.Vertex, uses map[[2]int]int, feat string) map[[2]int]*topo.Edge {
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
	edges := make(map[[2]int]*topo.Edge, len(keys))
	for i, k := range keys {
		seg := geom.NewLineSegment(verts[k[0]], verts[k[1]])
		edges[k] = bld.AddEdge(seg, tv[k[0]], tv[k[1]], topo.NewLineage(topo.Tok(feat, "edge", i)))
	}
	return edges
}

// triangleLoop builds a triangle's outer loop, marking a use reversed when its directed
// edge runs against the canonical (min,max) stored edge.
func triangleLoop(f [3]int, edges map[[2]int]*topo.Edge) topo.LoopSpec {
	uses := make([]topo.Use, 3)
	for i := 0; i < 3; i++ {
		a, b := f[i], f[(i+1)%3]
		uses[i] = topo.Use{Edge: edges[edgeKeyOf(a, b)], Reversed: a > b}
	}
	return topo.OuterLoop(uses...)
}

// trianglePlane fits the plane through a triangle's centroid with its (winding) normal,
// falling back to +Z for a degenerate triangle.
func trianglePlane(verts []math.Point3, f [3]int) geom.Surface {
	a, b, c := verts[f[0]], verts[f[1]], verts[f[2]]
	centroid := math.P3((a.X+b.X+c.X)/3, (a.Y+b.Y+c.Y)/3, (a.Z+b.Z+c.Z)/3)
	p, err := geom.NewPlane(centroid, a.VectorTo(b).Cross(a.VectorTo(c)))
	if err != nil {
		p, _ = geom.NewPlane(centroid, math.V3(0, 0, 1))
	}
	return p
}
