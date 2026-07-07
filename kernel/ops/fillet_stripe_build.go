// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// rebuildWithStripe reconstructs the body with the tangent stripe rounded in: the shared face and the
// walls are re-trimmed onto the blend's contact curves, the vertical smooth edge below each junction
// is split at the tube depth (its upper part consumed), and one blend face per segment is added,
// consecutive faces sharing a junction section circle. Mirrors the topological half of OCCT
// ChFi3d_Builder — the geometry came from the blend engine (fillet_stripe.go). See ADR-0050 P4b.
func rebuildWithStripe(b *topo.Body, st *tangentStripe) (*topo.Body, error) {
	g := &stripeBuild{
		st: st, bld: topo.NewBuilder(b.IsSolid(), b.Lineage()),
		verts: map[*topo.Vertex]*topo.Vertex{}, edges: map[*topo.Edge]*topo.Edge{},
		reRev: map[*topo.Edge]bool{}, chainIdx: map[*topo.Edge]int{}, downIdx: map[*topo.Edge]int{},
		junIdx: map[*topo.Vertex]int{},
	}
	g.index()
	for _, v := range b.Vertices() {
		if _, gone := g.junIdx[v]; !gone {
			g.verts[v] = g.bld.AddVertex(v.Point(), v.Lineage())
		}
	}
	if err := g.addNewVertsAndEdges(); err != nil {
		return nil, err
	}
	for _, e := range b.Edges() {
		if _, chain := g.chainIdx[e]; chain {
			continue
		}
		if _, down := g.downIdx[e]; down {
			continue
		}
		g.edges[e] = g.bld.AddEdge(e.Geometry(), g.verts[e.StartVertex()], g.verts[e.EndVertex()], e.Lineage())
	}
	for _, f := range b.Faces() {
		g.copyFace(f)
	}
	g.addBlendFaces()
	return g.bld.Build(), nil
}

type stripeBuild struct {
	st       *tangentStripe
	bld      *topo.Builder
	verts    map[*topo.Vertex]*topo.Vertex
	edges    map[*topo.Edge]*topo.Edge
	reRev    map[*topo.Edge]bool // how a re-trimmed face used each new stripe edge (blend uses the opposite)
	chainIdx map[*topo.Edge]int  // chain edge → segment index
	downIdx  map[*topo.Edge]int  // vertical smooth edge → junction index
	junIdx   map[*topo.Vertex]int
	vS1, vW  []*topo.Vertex // per junction: shared-side foot, wall-side foot
	topE     []*topo.Edge   // per segment: shared-face contact
	wallE    []*topo.Edge   // per segment: wall contact
	section  []*topo.Edge   // per junction: tube section circle vS1[j]→vW[j]
	lowerE   []*topo.Edge   // per junction: surviving lower part of the split vertical edge
}

// index builds the lookup maps from the solved stripe: chain edges → segment, down edges → junction,
// junction vertices → index.
func (g *stripeBuild) index() {
	for i, e := range g.st.edges {
		g.chainIdx[e] = i
	}
	for j, d := range g.st.down {
		g.downIdx[d] = j
		g.junIdx[g.st.junction[j]] = j
	}
}

// addNewVertsAndEdges creates the stripe's new vertices (the shared-side and wall-side section feet at
// every junction) and its new edges (the two contacts per segment, the section circle and the lower
// split-edge remnant per junction).
func (g *stripeBuild) addNewVertsAndEdges() error {
	n := len(g.st.segs)
	lin := func(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("stripe", role, i)) }
	g.vS1, g.vW = make([]*topo.Vertex, n), make([]*topo.Vertex, n)
	for j := 0; j < n; j++ {
		g.vS1[j] = g.bld.AddVertex(g.st.segs[j].topA, lin("s1", j))
		g.vW[j] = g.bld.AddVertex(g.st.segs[j].wallA, lin("w", j))
	}
	g.topE, g.wallE = make([]*topo.Edge, n), make([]*topo.Edge, n)
	for i := 0; i < n; i++ {
		k := (i + 1) % n
		g.topE[i] = g.bld.AddEdge(g.st.segs[i].topContact, g.vS1[i], g.vS1[k], lin("top", i))
		g.wallE[i] = g.bld.AddEdge(g.st.segs[i].wallContact, g.vW[i], g.vW[k], lin("wall", i))
	}
	return g.addSectionsAndRemnants(lin)
}

// addSectionsAndRemnants adds, per junction, the tube section circle two blend faces share and the
// surviving lower part of the split vertical smooth edge.
func (g *stripeBuild) addSectionsAndRemnants(lin func(string, int) topo.Lineage) error {
	n := len(g.st.segs)
	g.section, g.lowerE = make([]*topo.Edge, n), make([]*topo.Edge, n)
	for j := 0; j < n; j++ {
		arc, err := geom.Arc3dByThreePoints(g.st.segs[j].topA, g.st.apex[j], g.st.segs[j].wallA)
		if err != nil {
			return fmt.Errorf("fillet: cannot build the section circle at stripe junction %d: %w", j, err)
		}
		g.section[j] = g.bld.AddEdge(arc, g.vS1[j], g.vW[j], lin("sec", j))
		bottom := otherVertex(g.st.down[j], g.st.junction[j])
		g.lowerE[j] = g.bld.AddEdge(geom.NewLineSegment(g.st.segs[j].wallA, bottom.Point()),
			g.vW[j], g.verts[bottom], lin("lower", j))
	}
	return nil
}

// copyFace copies a face, re-trimming the shared face (chain edge → shared contact) and the walls
// (chain edge → wall contact, vertical edge → lower remnant); others copied verbatim.
func (g *stripeBuild) copyFace(f *topo.Face) {
	specs := make([]topo.LoopSpec, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		uses := make([]topo.Use, 0, len(l.EdgeUses()))
		for _, u := range l.EdgeUses() {
			mu := g.mapUse(f, u)
			g.reRev[mu.Edge] = mu.Reversed
			uses = append(uses, mu)
		}
		if l.IsOuter() {
			specs = append(specs, topo.OuterLoop(uses...))
		} else {
			specs = append(specs, topo.InnerLoop(uses...))
		}
	}
	if f.Reversed() {
		g.bld.AddReversedFace(f.Geometry(), f.Lineage(), specs...)
		return
	}
	g.bld.AddFace(f.Geometry(), f.Lineage(), specs...)
}

// mapUse substitutes a removed edge's use with its stripe replacement, orienting the new edge so it
// runs the same way the original use did (matched by the use's from-vertex, mapped onto the new feet).
func (g *stripeBuild) mapUse(f *topo.Face, u *topo.EdgeUse) topo.Use {
	from := useFromVertex(u)
	if i, ok := g.chainIdx[u.Edge()]; ok {
		if f == g.st.shared {
			return dirUse(g.topE[i], g.vS1[g.junIdx[from]])
		}
		return dirUse(g.wallE[i], g.vW[g.junIdx[from]])
	}
	if j, ok := g.downIdx[u.Edge()]; ok {
		fromV := g.verts[from]
		if from == g.st.junction[j] {
			fromV = g.vW[j]
		}
		return dirUse(g.lowerE[j], fromV)
	}
	return topo.Use{Edge: g.edges[u.Edge()], Reversed: u.Reversed()}
}

// dirUse returns the use of edge e that starts at from (reversed when from is e's end vertex).
func dirUse(e *topo.Edge, from *topo.Vertex) topo.Use {
	return topo.Use{Edge: e, Reversed: e.StartVertex() != from}
}

// addBlendFaces adds one blend face per segment: the quad topE[i] · section[i+1] · wallE[i] · section[i],
// wound so the loop normal matches the blend surface's outward radial (a convex fillet's surface faces
// AWAY from the material, toward the rounded-off corner). A purely topological anchor is not enough —
// it keeps every edge used twice but can still leave the surface geometrically inside-out, which the
// mass-properties integral then reads as material on the wrong side of the tube.
func (g *stripeBuild) addBlendFaces() {
	n := len(g.st.segs)
	lin := func(i int) topo.Lineage { return topo.NewLineage(topo.Tok("stripe", "blend", i)) }
	for i := 0; i < n; i++ {
		k := (i + 1) % n
		ring := []math.Point3{g.vS1[i].Point(), g.vS1[k].Point(), g.vW[k].Point(), g.vW[i].Point()}
		loop := []topo.Use{
			dirUse(g.topE[i], g.vS1[i]),
			dirUse(g.section[k], g.vS1[k]),
			dirUse(g.wallE[i], g.vW[k]),
			dirUse(g.section[i], g.vW[i]),
		}
		if blendRingFlipped(g.st.segs[i].surf, ring) {
			loop = reverseLoop(loop)
		}
		g.bld.AddFace(g.st.segs[i].surf, lin(i), topo.OuterLoop(loop...))
	}
}

// blendRingFlipped reports whether the quad ring winds against the blend surface's outward normal —
// the triangle from three ring corners has a normal opposing surf.NormalAt at the ring centroid.
func blendRingFlipped(surf geom.Surface, ring []math.Point3) bool {
	nrm := ring[0].VectorTo(ring[1]).Cross(ring[0].VectorTo(ring[2]))
	u, v := surf.ParamAt(centroidPts(ring))
	return float64(nrm.Dot(surf.NormalAt(u, v))) < 0
}

// filletTangentStripe rounds a closed tangent chain (mixed straight/arc segments sharing one face) as
// one continuous blend stripe — the #1797 top-perimeter case the per-edge miter path could not build.
func filletTangentStripe(body *topo.Body, edges []*topo.Edge, closed bool, r float64) (*topo.Body, error) {
	st, err := solveTangentStripe(body, edges, closed, r)
	if err != nil {
		return nil, err
	}
	res, err := rebuildWithStripe(body, st)
	if err != nil {
		return nil, err
	}
	if rep := Validate(res); !rep.Valid || !res.IsSolid() {
		return nil, fmt.Errorf("fillet: tangent-stripe result is not a valid solid %v", rep.Issues)
	}
	return res, nil
}
