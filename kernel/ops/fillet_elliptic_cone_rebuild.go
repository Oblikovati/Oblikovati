// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The CLOSED-rim rebuild of the EllipticalCylinder∧Cone pinched canal (C3 — the pinch-AT-the-
// host-seam-azimuth sub-case ONLY; see closedEllipticConeSpan in fillet_elliptic_cone_stations.go
// for why the off-seam sub-case, B7/C2, declines instead of routing here): the rim vertex IS the
// pinch, so both host seams keep their full length and the rim edge is simply replaced by the
// band's two rails — single closed edges through the kept vertex, no seam re-aim, no split. The
// open-arc sibling (B4/B8) lives in fillet_elliptic_cone_runout.go.

// ellipticConeCanalBody dispatches the built canal to its rebuild. An empty reason means the
// body is the weld; a non-empty one names the obstruction (never a partial body).
func ellipticConeCanalBody(body *topo.Body, e *topo.Edge, canal *ellipticConeCanal) (*topo.Body, string) {
	if canal.closed {
		return ellipticConeClosedBody(body, e, canal)
	}
	return ellipticConeRunoutBody(body, e, canal)
}

// ellipticConeClosedBody rebuilds the solid around the closed pinched band.
func ellipticConeClosedBody(body *topo.Body, e *topo.Edge, canal *ellipticConeCanal) (*topo.Body, string) {
	g := newConeCanalRebuild(body, e, canal)
	if reason := g.prepareClosed(body); reason != "" {
		return nil, reason
	}
	g.copyEdgesExcept(body)
	for _, f := range body.Faces() {
		if reason := g.copyFaceWithRails(f); reason != "" {
			return nil, reason
		}
	}
	g.addBandFace()
	return g.bld.Build(), ""
}

// coneCanalRebuild is the in-progress rebuild state shared by the closed and runout topologies.
type coneCanalRebuild struct {
	canal *ellipticConeCanal
	rim   *topo.Edge
	res   Resolution
	bld   *topo.Builder
	verts map[*topo.Vertex]*topo.Vertex
	edges map[*topo.Edge]*topo.Edge
	skipV map[*topo.Vertex]bool
	// subst maps an ORIGINAL edge to the use-sequence replacing it on a given face; the rim maps
	// per-face (wall/cone rails), the re-aimed seams map uniformly.
	rimUses  map[*topo.Face][]topo.Use
	seamRepl map[*topo.Edge]*topo.Edge
	bandLoop []topo.Use
}

func newConeCanalRebuild(body *topo.Body, e *topo.Edge, canal *ellipticConeCanal) *coneCanalRebuild {
	return &coneCanalRebuild{
		canal: canal, rim: e, res: ResolutionForBody(body), bld: topo.NewBuilder(body.IsSolid(), body.Lineage()),
		verts: map[*topo.Vertex]*topo.Vertex{}, edges: map[*topo.Edge]*topo.Edge{},
		skipV: map[*topo.Vertex]bool{}, rimUses: map[*topo.Face][]topo.Use{},
		seamRepl: map[*topo.Edge]*topo.Edge{},
	}
}

// prepareClosed builds the vertices and the two whole rails through the kept pinch vertex (the
// rim vertex — closedEllipticConeSpan only ships the pinch-AT-the-seam sub-case, see its doc
// comment).
func (g *coneCanalRebuild) prepareClosed(body *topo.Body) string {
	g.copyVerts(body)
	pinchV := g.verts[g.rim.StartVertex()]
	wallRail, coneRail, reason := g.railCurves()
	if reason != "" {
		return reason
	}
	return g.bindWholeRails(wallRail, coneRail, pinchV)
}

func (g *coneCanalRebuild) copyVerts(body *topo.Body) {
	for _, v := range body.Vertices() {
		if g.skipV[v] {
			continue
		}
		g.verts[v] = g.bld.AddVertex(v.Point(), v.Lineage())
	}
}

// railCurves extracts the two rails as the loft's own u-isocurves (face boundary and face
// surface agree exactly by construction).
func (g *coneCanalRebuild) railCurves() (geom.BSplineCurve, geom.BSplineCurve, string) {
	w, err := geom.SurfaceIsoCurve(g.canal.loft.Surf, true, 0)
	if err != nil {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, fmt.Sprintf("elliptic cone canal: wall rail extraction: %v", err)
	}
	c, err := geom.SurfaceIsoCurve(g.canal.loft.Surf, true, 1)
	if err != nil {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, fmt.Sprintf("elliptic cone canal: cone rail extraction: %v", err)
	}
	wb, okW := w.(geom.BSplineCurve)
	cb, okC := c.(geom.BSplineCurve)
	if !okW || !okC {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, fmt.Sprintf("elliptic cone canal: rails are %T/%T, need BSplineCurve", w, c)
	}
	return wb, cb, ""
}

// bindWholeRails (C3): one closed edge per rail through the kept pinch vertex.
func (g *coneCanalRebuild) bindWholeRails(wallRail, coneRail geom.BSplineCurve, pinchV *topo.Vertex) string {
	lin := func(role string) topo.Lineage { return topo.NewLineage(topo.Tok("conecanal", role, 0)) }
	wE := g.bld.AddEdge(wallRail, pinchV, pinchV, lin("wallrail"))
	cE := g.bld.AddEdge(coneRail, pinchV, pinchV, lin("conerail"))
	wRev, reason := g.hostWalkReversed(g.canal.wallF, wallRail)
	if reason != "" {
		return reason
	}
	cRev, reason := g.hostWalkReversed(g.canal.coneF, coneRail)
	if reason != "" {
		return reason
	}
	g.rimUses[g.canal.wallF] = []topo.Use{{Edge: wE, Reversed: wRev}}
	g.rimUses[g.canal.coneF] = []topo.Use{{Edge: cE, Reversed: cRev}}
	g.bandLoop = []topo.Use{{Edge: wE, Reversed: !wRev}, {Edge: cE, Reversed: !cRev}}
	return ""
}

// hostWalkReversed reports whether the host face must traverse the rail curve REVERSED to keep
// the rotational sense its loop walked the rim with (the rail is a near-parallel offset of the
// rim, so a tangent dot decides).
func (g *coneCanalRebuild) hostWalkReversed(hostF *topo.Face, rail geom.BSplineCurve) (bool, string) {
	use, ok := edgeUseOn(hostF, g.rim)
	if !ok {
		return false, fmt.Sprintf("elliptic cone canal: face %d does not use the rim edge", hostF.ID())
	}
	lo, hi := rail.Domain()
	mid := 0.5 * (lo + hi)
	railTan := rail.PointAt(mid).VectorTo(rail.PointAt(mid + (hi-lo)*1e-4))
	rimWalk := rimWalkDirNear(g.rim, rail.PointAt(mid), use.Reversed())
	d := railTan.Dot(rimWalk)
	if d == 0 {
		return false, "elliptic cone canal: rail/rim tangent alignment is degenerate"
	}
	return d < 0, ""
}

// edgeUseOn finds a face's (first) use of an edge.
func edgeUseOn(f *topo.Face, e *topo.Edge) (*topo.EdgeUse, bool) {
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			if u.Edge() == e {
				return u, true
			}
		}
	}
	return nil, false
}

// rimWalkDirNear is the direction the face's loop walks the rim, evaluated at the rim point
// nearest q (dense parameter sampling — the rim is a conic arc, its tangent varies slowly).
func rimWalkDirNear(rim *topo.Edge, q math.Point3, reversed bool) math.Vector3 {
	c := rim.Geometry()
	lo, hi := c.Domain()
	bestT, bestD := lo, stdmath.Inf(1)
	for k := 0; k <= 256; k++ {
		t := lo + (hi-lo)*float64(k)/256
		if d := float64(c.PointAt(t).DistanceTo(q)); d < bestD {
			bestT, bestD = t, d
		}
	}
	tan := c.PointAt(bestT).VectorTo(c.PointAt(bestT + (hi-lo)*1e-6))
	if reversed {
		return tan.Scale(-1)
	}
	return tan
}

// reaimedSeamEdge rebuilds the seam ruling with the rim-vertex end replaced by the junction
// vertex, preserving the original start/end roles so loop use flags carry over verbatim.
func (g *coneCanalRebuild) reaimedSeamEdge(seamE *topo.Edge, rimV, farV *topo.Vertex, seamV *topo.Vertex) *topo.Edge {
	far := g.verts[farV]
	if seamE.StartVertex() == rimV {
		return g.bld.AddEdge(geom.NewLineSegment(seamV.Point(), far.Point()), seamV, far, seamE.Lineage())
	}
	return g.bld.AddEdge(geom.NewLineSegment(far.Point(), seamV.Point()), far, seamV, seamE.Lineage())
}

// copyEdgesExcept copies every edge except the rim and the substituted seams.
func (g *coneCanalRebuild) copyEdgesExcept(body *topo.Body) {
	for _, e := range body.Edges() {
		if e == g.rim {
			continue
		}
		if _, replaced := g.seamRepl[e]; replaced {
			continue
		}
		g.edges[e] = g.bld.AddEdge(e.Geometry(), g.verts[e.StartVertex()], g.verts[e.EndVertex()], e.Lineage())
	}
}

// copyFaceWithRails copies one face, substituting the rim edge by that face's rail uses and any
// re-aimed seam by its replacement.
func (g *coneCanalRebuild) copyFaceWithRails(f *topo.Face) string {
	specs := make([]topo.LoopSpec, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		uses, reason := g.mappedLoopUses(f, l)
		if reason != "" {
			return reason
		}
		if l.IsOuter() {
			specs = append(specs, topo.OuterLoop(uses...))
		} else {
			specs = append(specs, topo.InnerLoop(uses...))
		}
	}
	if f.Reversed() {
		g.bld.AddReversedFace(f.Geometry(), f.Lineage(), specs...)
	} else {
		g.bld.AddFace(f.Geometry(), f.Lineage(), specs...)
	}
	return ""
}

// mappedLoopUses maps one loop's uses onto the rebuilt edges.
func (g *coneCanalRebuild) mappedLoopUses(f *topo.Face, l *topo.Loop) ([]topo.Use, string) {
	uses := make([]topo.Use, 0, len(l.EdgeUses())+2)
	for _, u := range l.EdgeUses() {
		switch {
		case u.Edge() == g.rim:
			rail, ok := g.rimUses[f]
			if !ok {
				return nil, fmt.Sprintf("elliptic cone canal: face %d uses the rim but is not a canal host", f.ID())
			}
			uses = append(uses, rail...)
		case g.seamRepl[u.Edge()] != nil:
			uses = append(uses, topo.Use{Edge: g.seamRepl[u.Edge()], Reversed: u.Reversed()})
		default:
			uses = append(uses, topo.Use{Edge: g.edges[u.Edge()], Reversed: u.Reversed()})
		}
	}
	return uses, ""
}

// addBandFace adds the pinched canal band. The loop was assembled rail-by-rail (each circuit
// closes through the pinch vertex), antiparallel to the host uses — the manifold rule that makes
// the shell orientable; the surface's own normal was already turned outward by the walk-direction
// probe (ellipticConeBandOutward).
func (g *coneCanalRebuild) addBandFace() {
	lin := topo.NewLineage(topo.Tok("conecanal", "band", 0))
	g.bld.AddFace(g.canal.loft.Surf, lin, topo.OuterLoop(g.bandLoop...))
}
