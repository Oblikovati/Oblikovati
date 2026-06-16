// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
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
	for _, v := range b.Vertices() {
		g.verts[v] = g.bld.AddVertex(v.Point(), v.Lineage())
	}
	g.addNewEdges()
	for _, e := range b.Edges() {
		if e == af.arcEdge || e == af.ends[0].smoothLine || e == af.ends[1].smoothLine {
			continue // arc removed; smooth lines split
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
	upper  [2]*topo.Edge       // smooth-line upper rimV→vc per end (side ∩ end-cap)
	lower  [2]*topo.Edge       // smooth-line lower vc→bottom per end (side ∩ cylinder)
	reRev  map[*topo.Edge]bool // how a re-trimmed face used each new edge (so the new face uses the opposite)
}

func (g *arcBuild) addNewEdges() {
	af := g.af
	lin := func(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("arcfillet", role, i)) }
	for i := 0; i < 2; i++ {
		g.vc[i] = g.bld.AddVertex(af.ends[i].vc, lin("vc", i))
		g.vt[i] = g.bld.AddVertex(af.ends[i].vt, lin("vt", i))
	}
	bis := bisector(af.ends[0].refDir, af.ends[1].refDir)
	cylArc, _ := geom.Arc3dByThreePoints(af.ends[0].vc, af.torusCenter.TranslateBy(bis.Scale(af.r+af.majorR)), af.ends[1].vc)
	g.cylTan = g.bld.AddEdge(cylArc, g.vc[0], g.vc[1], lin("cyltan", 0))
	capArc, _ := geom.Arc3dByThreePoints(af.ends[0].vt, af.capCenter.TranslateBy(bis.Scale(af.majorR)), af.ends[1].vt)
	g.capTan = g.bld.AddEdge(capArc, g.vt[0], g.vt[1], lin("captan", 0))
	for i := 0; i < 2; i++ {
		u, _ := af.torus.ParamAt(af.ends[i].vc)
		ea, _ := geom.Arc3dByThreePoints(af.ends[i].vc, af.torus.PointAt(u, quarterTube), af.ends[i].vt)
		g.endArc[i] = g.bld.AddEdge(ea, g.vc[i], g.vt[i], lin("endarc", i))
		rimV, bottom := g.verts[af.ends[i].rimV], g.verts[af.ends[i].bottomV]
		g.capLn[i] = g.bld.AddEdge(geom.NewLineSegment(af.ends[i].vt, af.ends[i].rimV.Point()), g.vt[i], rimV, lin("capline", i))
		g.upper[i] = g.bld.AddEdge(geom.NewLineSegment(af.ends[i].rimV.Point(), af.ends[i].vc), rimV, g.vc[i], lin("supper", i))
		g.lower[i] = g.bld.AddEdge(geom.NewLineSegment(af.ends[i].vc, af.ends[i].bottomV.Point()), g.vc[i], bottom, lin("slower", i))
	}
}

// bisector returns the unit direction halfway between two radial directions (the arc midpoint angle).
func bisector(a, b math.UnitVector3) math.Vector3 {
	m, err := math.UnitVector3FromVector(a.AsVector().Add(b.AsVector()))
	if err != nil {
		return a.AsVector()
	}
	return m.AsVector()
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
		// arc on the cap → capLine_A + capTan + capLine_B, walking rimV_0 → vt_0 → vt_1 → rimV_1
		return orientChain(u, []chainEdge{
			{g.capLn[0], g.verts[af.ends[0].rimV], g.vt[0]},
			{g.capTan, g.vt[0], g.vt[1]},
			{g.capLn[1], g.vt[1], g.verts[af.ends[1].rimV]},
		})
	case u.Edge() == af.arcEdge && f == af.cylF:
		// arc on the cylinder → cyl-tangent arc (vc0→vc1); reversed when the use enters from end 1.
		rev := useFromVertex(u).Point().DistanceTo(af.ends[1].rimV.Point()) < weldPointTol
		return []topo.Use{{Edge: g.cylTan, Reversed: rev}}
	case u.Edge() == af.ends[0].smoothLine || u.Edge() == af.ends[1].smoothLine:
		i := 0
		if u.Edge() == af.ends[1].smoothLine {
			i = 1
		}
		if f == af.cylF {
			// cylinder keeps only the lower segment (vc→bottom); reversed when the use enters from bottom.
			rev := useFromVertex(u).Point().DistanceTo(af.ends[i].bottomV.Point()) < weldPointTol
			return []topo.Use{{Edge: g.lower[i], Reversed: rev}}
		}
		return orientChain(u, []chainEdge{{g.upper[i], g.verts[af.ends[i].rimV], g.vc[i]}, {g.lower[i], g.vc[i], g.verts[af.ends[i].bottomV]}})
	}
	return []topo.Use{{Edge: g.edges[u.Edge()], Reversed: u.Reversed()}}
}

// chainEdge is one directed edge of a substitution chain.
type chainEdge struct {
	e        *topo.Edge
	from, to *topo.Vertex
}

// orientChain emits a substitution chain in the direction matching the replaced use. Each chainEdge
// carries the DESIRED traversal (from→to); its reversed flag is whether that opposes the edge's own
// natural start. When the use runs against the chain's overall direction, the whole sequence flips.
func orientChain(u *topo.EdgeUse, chain []chainEdge) []topo.Use {
	out := make([]topo.Use, len(chain))
	forward := useFromVertex(u).Point().DistanceTo(chain[0].from.Point()) < weldPointTol
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
	for i := 0; i < 2; i++ {
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
