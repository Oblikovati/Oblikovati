// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// rebuildWithArcFillet rebuilds the body with the arc rounded into a torus + two flat setback
// end-caps. Every vertex/edge/face is copied (the TransformBody pattern), then the cap, cylinder, and
// the two side planes are re-trimmed onto the new tangent arcs, each smooth tangent line is split at
// its cyl-tangent point, and the torus + two end-cap faces are added.
func rebuildWithArcFillet(b *topo.Body, af *arcFillet) (*topo.Body, error) {
	g := &arcBuild{af: af, bld: topo.NewBuilder(b.IsSolid(), b.Lineage()),
		verts: map[*topo.Vertex]*topo.Vertex{}, edges: map[*topo.Edge]*topo.Edge{}, reRev: map[*topo.Edge]bool{}}
	g.resolveEndCapMerges() // an end cap coplanar with its side face is absorbed, not emitted
	for _, v := range b.Vertices() {
		if g.supersededRimVertex(v) {
			continue // the merge replaces this corner by the cap-tangent point; no edge reaches it
		}
		g.verts[v] = g.bld.AddVertex(v.Point(), v.Lineage())
	}
	g.addNewEdges()
	for _, e := range b.Edges() {
		if g.replacedEdge(e) {
			continue // arc removed; smooth lines split; a merged end's cap∩side edge re-ended
		}
		g.edges[e] = g.bld.AddEdge(e.Geometry(), g.verts[e.StartVertex()], g.verts[e.EndVertex()], e.Lineage())
	}
	for _, f := range b.Faces() {
		g.copyFace(f)
	}
	g.addTorusAndCaps()
	return g.bld.Build(), nil
}

type arcBuild struct {
	af     *arcFillet
	bld    *topo.Builder
	verts  map[*topo.Vertex]*topo.Vertex
	edges  map[*topo.Edge]*topo.Edge
	vc, vt [2]*topo.Vertex
	cylTan *topo.Edge          // cyl-tangent arc vc_0→vc_1
	capTan *topo.Edge          // cap-tangent arc vt_0→vt_1
	endArc [2]*topo.Edge       // tube cross-section vc→vt per end
	capLn  [2]*topo.Edge       // cap line vt→rimV per end
	upper  [2]*topo.Edge       // smooth-line upper rimV→vc per end (side ∩ end-cap); nil when merged
	lower  [2]*topo.Edge       // smooth-line lower vc→bottom per end (side ∩ cylinder)
	reRev  map[*topo.Edge]bool // how a re-trimmed face used each new edge (so the new face uses the opposite)
	// The setback end-cap merge (fillet_arc_endcap.go): when the cap's radial plane IS the side face's
	// plane, the side face absorbs the cap, the rim vertex it was drawn to is superseded, and that end's
	// cap∩side edge is re-ended on the cap-tangent point.
	merged   [2]bool
	capSide  [2]*topo.Edge // the original cap∩side edge at rimV (merged ends only)
	capShort [2]*topo.Edge // the same line, re-ended on vt (merged ends only)
}

func (g *arcBuild) addNewEdges() {
	af := g.af
	lin := func(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("arcfillet", role, i)) }
	for i := range 2 {
		g.vc[i] = g.bld.AddVertex(af.ends[i].vc, lin("vc", i))
		g.vt[i] = g.bld.AddVertex(af.ends[i].vt, lin("vt", i))
	}
	// Both tangent arcs are on-band circles, so each is pinned by its own two terminations' azimuths —
	// NOT by the two ends' radials, which a run-out end no longer shares between the two contacts.
	cylMid := af.torus.PointAt((af.ends[0].uCyl+af.ends[1].uCyl)/2, af.vCyl)
	cylArc, _ := geom.Arc3dByThreePoints(af.ends[0].vc, cylMid, af.ends[1].vc)
	g.cylTan = g.bld.AddEdge(cylArc, g.vc[0], g.vc[1], lin("cyltan", 0))
	capMid := af.torus.PointAt((af.ends[0].uCap+af.ends[1].uCap)/2, capTube)
	capArc, _ := geom.Arc3dByThreePoints(af.ends[0].vt, capMid, af.ends[1].vt)
	g.capTan = g.bld.AddEdge(capArc, g.vt[0], g.vt[1], lin("captan", 0))
	for i := range 2 {
		g.addEndEdges(i, lin)
	}
}

// addEndEdges builds one end's new edges: the torus cross-section arc and the smooth line's lower
// piece always, plus EITHER the setback triangle's two straight sides (an unmerged end) OR the re-ended
// cap∩side edge (a merged end, whose rim vertex is superseded and so has no edge drawn to it).
func (g *arcBuild) addEndEdges(i int, lin func(string, int) topo.Lineage) {
	af := g.af
	g.endArc[i] = g.bld.AddEdge(g.terminalCurve(i), g.vc[i], g.vt[i], lin("endarc", i))
	g.lower[i] = g.bld.AddEdge(geom.NewLineSegment(af.ends[i].vc, af.ends[i].bottomV.Point()),
		g.vc[i], g.verts[af.ends[i].bottomV], lin("slower", i))
	if g.absorbed(i) {
		g.addMergedCapSide(i, lin("capside", i))
		return
	}
	rimV := g.verts[af.ends[i].rimV]
	g.capLn[i] = g.bld.AddEdge(geom.NewLineSegment(af.ends[i].vt, af.ends[i].rimV.Point()), g.vt[i], rimV, lin("capline", i))
	g.upper[i] = g.bld.AddEdge(geom.NewLineSegment(af.ends[i].rimV.Point(), af.ends[i].vc), rimV, g.vc[i], lin("supper", i))
}

// terminalCurve is end i's band boundary from the cyl-tangent contact to the cap-tangent contact: the
// SPIRIC section when the band runs out on the side plane, otherwise the tube cross-section arc in this
// end's own radial plane (through the tube midpoint between the two contacts).
func (g *arcBuild) terminalCurve(i int) geom.Curve3 {
	af := g.af
	if sec := af.ends[i].runout; sec != nil {
		return *sec
	}
	arc, _ := geom.Arc3dByThreePoints(af.ends[i].vc, af.torus.PointAt(af.ends[i].uCyl, (af.vCyl+capTube)/2), af.ends[i].vt)
	return arc
}

// copyFace copies a face, re-trimming the cap, cylinder, and side planes; others copied verbatim.
func (g *arcBuild) copyFace(f *topo.Face) {
	specs := make([]topo.LoopSpec, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		var uses []topo.Use
		for _, u := range l.EdgeUses() {
			mapped := g.mapUse(f, u)
			for _, mu := range mapped {
				g.reRev[mu.Edge] = mu.Reversed // record how the re-trimmed face used each new edge
			}
			uses = append(uses, mapped...)
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

// mapUse maps one edge use onto the rebuilt edges, substituting the arc and smooth lines per face.
func (g *arcBuild) mapUse(f *topo.Face, u *topo.EdgeUse) []topo.Use {
	af := g.af
	switch {
	case u.Edge() == af.arcEdge && f == af.capF:
		// arc on the cap → (capLine_A) + capTan + (capLine_B), walking end 0 → end 1; a MERGED end
		// contributes no cap line, because its cap∩side edge already reaches the cap-tangent point.
		return orientChain(vertsCoincide(useFromVertex(u).Point(), af.ends[0].rimV.Point()), g.arcChainOnCap())
	case u.Edge() == af.arcEdge && f == af.cylF:
		// arc on the cylinder → cyl-tangent arc (vc0→vc1); reversed when the use enters from end 1.
		rev := vertsCoincide(useFromVertex(u).Point(), af.ends[1].rimV.Point())
		return []topo.Use{{Edge: g.cylTan, Reversed: rev}}
	case u.Edge() == af.ends[0].smoothLine || u.Edge() == af.ends[1].smoothLine:
		return g.mapSmoothLine(f, u)
	}
	if i, ok := g.capSideIndex(u.Edge()); ok {
		return []topo.Use{{Edge: g.capShort[i], Reversed: u.Reversed()}} // same line, re-ended on vt
	}
	return []topo.Use{{Edge: g.edges[u.Edge()], Reversed: u.Reversed()}}
}

// mapSmoothLine maps a use of one end's cyl∩side smooth line: the cylinder keeps only the piece below
// the cyl-tangent point, while the side face takes the chain that walks away from the cap — the setback
// triangle's upper side when the end kept its own cap face, the band's terminal cross-section ARC when
// the side face absorbed that cap.
func (g *arcBuild) mapSmoothLine(f *topo.Face, u *topo.EdgeUse) []topo.Use {
	af := g.af
	i := 0
	if u.Edge() == af.ends[1].smoothLine {
		i = 1
	}
	if f == af.cylF {
		// cylinder keeps only the lower segment (vc→bottom); reversed when the use enters from bottom.
		rev := vertsCoincide(useFromVertex(u).Point(), af.ends[i].bottomV.Point())
		return []topo.Use{{Edge: g.lower[i], Reversed: rev}}
	}
	return orientChain(vertsCoincide(useFromVertex(u).Point(), af.ends[i].rimV.Point()), g.smoothChainOnSide(i))
}

// vertsCoincide reports whether two fillet vertices are the same point. The tolerance is model-relative
// (ADR-0042, #1399): for distinct vertices it scales with their separation (so the test stays correct
// at any scale), and for a coincident pair it floors to the base weld — no cm-anchored constant.
func vertsCoincide(a, b math.Point3) bool {
	return a.DistanceTo(b) < tol.ForPoints([]math.Point3{a, b}).Weld()
}

// chainEdge is one directed edge of a substitution chain.
type chainEdge struct {
	e        *topo.Edge
	from, to *topo.Vertex
}

// orientChain emits a substitution chain in the direction matching the replaced use. Each chainEdge
// carries the DESIRED traversal (from→to); its reversed flag is whether that opposes the edge's own
// natural start. When the use runs against the chain's overall direction (forward=false) the whole
// sequence flips. forward is decided by the CALLER against the ORIGINAL vertex the chain replaces, not
// against chain[0]: a merged end's chain no longer starts on the vertex the use does (it starts on the
// cap-tangent point that superseded it), so reading the direction off chain[0] would invert it.
func orientChain(forward bool, chain []chainEdge) []topo.Use {
	out := make([]topo.Use, len(chain))
	for i, c := range chain {
		rev := c.e.StartVertex() != c.from // the edge's natural start differs from the desired start
		if forward {
			out[i] = topo.Use{Edge: c.e, Reversed: rev}
		} else {
			out[len(chain)-1-i] = topo.Use{Edge: c.e, Reversed: !rev}
		}
	}
	return out
}

// addTorusAndCaps adds the torus band over the arc and the two flat setback end-cap triangles, each
// loop oriented opposite to the re-trimmed face that shares its anchor edge so every edge is used in
// both senses (a valid manifold).
func (g *arcBuild) addTorusAndCaps() {
	lin := func(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("arcfillet", role, i)) }
	// Torus patch cycle vc0→vc1→vt1→vt0→vc0; anchor cylTan opposite the cylinder (already built).
	torus := []topo.Use{topo.Fwd(g.cylTan), topo.Fwd(g.endArc[1]), topo.Rev(g.capTan), topo.Rev(g.endArc[0])}
	if torus[0].Reversed != !g.reRev[g.cylTan] {
		torus = reverseLoop(torus)
	}
	g.bld.AddFace(g.af.torus, lin("torus", 0), topo.OuterLoop(torus...))
	torusEndRev := map[*topo.Edge]bool{}
	for _, u := range torus {
		torusEndRev[u.Edge] = u.Reversed
	}
	for i := range 2 {
		if g.absorbed(i) {
			continue // the side face absorbed this end: it is not a face of its own (fillet_arc_endcap.go)
		}
		// End-cap cycle vc→vt→rimV→vc; anchor endArc opposite the torus (now the surrounding faces —
		// torus on endArc, cap on capLn, side on upper — are all built, so one orientation satisfies all).
		ecLoop := []topo.Use{topo.Fwd(g.endArc[i]), topo.Fwd(g.capLn[i]), topo.Fwd(g.upper[i])}
		if ecLoop[0].Reversed != !torusEndRev[g.endArc[i]] {
			ecLoop = reverseLoop(ecLoop)
		}
		plane, _ := geom.NewPlane(g.af.ends[i].rimV.Point(), endCapNormal(g.af, i))
		g.bld.AddFace(plane, lin("endcap", i), topo.OuterLoop(ecLoop...))
	}
}

// reverseLoop walks a loop the opposite way: reverse the use order and invert each reversed flag.
func reverseLoop(uses []topo.Use) []topo.Use {
	out := make([]topo.Use, len(uses))
	for i, u := range uses {
		out[len(uses)-1-i] = topo.Use{Edge: u.Edge, Reversed: !u.Reversed}
	}
	return out
}

// endCapNormal returns the setback end-cap's plane normal — the radial plane (axis × refDir), signed
// to point AWAY from the arc (away from the other end's radial direction); face winding is handled by
// the loop orientation.
func endCapNormal(af *arcFillet, i int) math.Vector3 {
	n := af.axisN.AsVector().Cross(af.ends[i].refDir.AsVector())
	if n.Dot(af.ends[1-i].refDir.AsVector()) > 0 {
		n = n.Scale(-1) // point away from the arc (the other end lies on the +n side otherwise)
	}
	return n
}
