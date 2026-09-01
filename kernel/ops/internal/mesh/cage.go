// SPDX-License-Identifier: GPL-2.0-only

package mesh

import (
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Assembling a triangle CAGE into a B-rep body. It lived in csg_body.go because the BSP
// boolean was its first caller, but the convex hull and the mesh-to-B-rep bridge build
// bodies the same way — so it belongs beside the mesh types, not in the operation layer.

// CageToBody assembles a B-rep from a welded triangle cage: one shared vertex per cage
// vertex, one shared edge per undirected edge (sorted for stable lineage), a planar face
// per triangle. The body is a solid when every edge is shared by exactly two triangles.
func CageToBody(verts []math.Point3, faces [][3]int, feat string) *topo.Body {
	if len(faces) == 0 {
		return nil
	}
	uses := EdgeUseCounts(faces)
	bld := topo.NewBuilder(IsClosedCage(uses), topo.NewLineage(topo.Tok(feat, "body", 0)))
	tv := make([]*topo.Vertex, len(verts))
	for i, p := range verts {
		tv[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	edges := buildCageEdges(bld, verts, tv, uses, feat)
	for fi, f := range faces {
		bld.AddFace(TrianglePlane(verts, f), topo.NewLineage(topo.Tok(feat, "face", fi)), triangleLoop(f, edges))
	}
	return bld.Build()
}

// EdgeUseCounts counts how many triangles use each undirected edge.
func EdgeUseCounts(faces [][3]int) map[[2]int]int {
	uses := map[[2]int]int{}
	for _, f := range faces {
		for i := range 3 {
			uses[EdgeKeyOf(f[i], f[(i+1)%3])]++
		}
	}
	return uses
}

func IsClosedCage(uses map[[2]int]int) bool {
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

// TrianglePlane fits the plane through a triangle's centroid with its (winding) normal,
// falling back to +Z for a degenerate triangle.
func TrianglePlane(verts []math.Point3, f [3]int) geom.Surface {
	a, b, c := verts[f[0]], verts[f[1]], verts[f[2]]
	centroid := math.P3((a.X+b.X+c.X)/3, (a.Y+b.Y+c.Y)/3, (a.Z+b.Z+c.Z)/3)
	p, err := geom.NewPlane(centroid, a.VectorTo(b).Cross(a.VectorTo(c)))
	if err != nil {
		p, _ = geom.NewPlane(centroid, math.V3(0, 0, 1))
	}
	return p
}

// triangleLoop builds a triangle's outer loop, marking a use reversed when its directed
// edge runs against the canonical (min,max) stored edge.
func triangleLoop(f [3]int, edges map[[2]int]*topo.Edge) topo.LoopSpec {
	uses := make([]topo.Use, 3)
	for i := range 3 {
		a, b := f[i], f[(i+1)%3]
		uses[i] = topo.Use{Edge: edges[EdgeKeyOf(a, b)], Reversed: a > b}
	}
	return topo.OuterLoop(uses...)
}

func EdgeKeyOf(a, b int) [2]int {
	if a < b {
		return [2]int{a, b}
	}
	return [2]int{b, a}
}

// OnSegment reports whether p lies on the interior of segment a→b: within lineTol of
// the line and strictly between the endpoints (more than lineTol from each). Endpoint
// exclusion is by absolute distance, not a parameter fraction, so a vertex near a long
// edge's end is still recognized as a T-junction. lineTol is the model-relative on-line
// resolution the caller derives (ADR-0042), not a fixed constant.
func OnSegment(p, a, b math.Point3, lineTol float64) bool {
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
	if foot.DistanceTo(p) >= lineTol {
		return false
	}
	return p.DistanceTo(a) > lineTol && p.DistanceTo(b) > lineTol
}
