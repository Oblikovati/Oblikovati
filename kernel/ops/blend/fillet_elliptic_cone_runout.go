// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"
	stdmath "math"
	"slices"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The OPEN-arc rebuild of the EllipticalCylinder∧Cone pinched canal (B4/B8): the rim is a half
// circle whose FAR end is the host-tangency pinch — an existing solid vertex the band tapers
// into (no runout corner needed there) — and whose NEAR end runs out onto the planar side face:
// the band's start cross-section arc lies IN that plane, the two host boundary rulings are
// shortened onto the arc's feet, and the side face gains the added lens between the old corner
// and the arc (DRAWEXE B4: flat face 78125 → 78329.5, lens 204.48 — matched in closed form by
// the W-F derivation, perface_targets.py).

// ellipticConeRunoutBody rebuilds the solid around the open pinched band.
func ellipticConeRunoutBody(body *topo.Body, e *topo.Edge, canal *ellipticConeCanal) (*topo.Body, string) {
	g := newConeCanalRebuild(body, e, canal)
	ro, reason := g.prepareRunout(body)
	if reason != "" {
		return nil, reason
	}
	g.copyEdgesExcept(body)
	for _, f := range body.Faces() {
		if reason := g.copyRunoutFace(f, ro); reason != "" {
			return nil, reason
		}
	}
	g.addBandFace()
	return g.bld.Build(), ""
}

// coneRunout carries the open-end topology: the side face taking the lens, its two shortened
// boundary rulings, and the section arc.
type coneRunout struct {
	sideF                *topo.Face
	wallSeamE, coneSeamE *topo.Edge
	arcE                 *topo.Edge
	arcFwdFromWall       bool // arc edge built wall-foot → cone-foot
}

// prepareRunout resolves the open-end anchors, verifies the runout lands on a planar side face,
// and builds the vertices, rails, shortened rulings and section arc.
func (g *coneCanalRebuild) prepareRunout(body *topo.Body) (*coneRunout, string) {
	openIdx := runoutEndIndex(g.canal)
	voOld, vpOld := runoutEndVertices(g.rim, g.canal)
	g.skipV[voOld] = true
	g.copyVerts(body)
	ro, reason := g.bindRunoutEnd(voOld, openIdx)
	if reason != "" {
		return nil, reason
	}
	return ro, g.bindRunoutRails(ro, openIdx, g.verts[vpOld])
}

// runoutEndIndex returns the station index of the open (non-pinched) end.
func runoutEndIndex(canal *ellipticConeCanal) int {
	last := len(canal.stations) - 1
	if float64(canal.stations[0].wallFoot.DistanceTo(canal.pinch)) < float64(canal.stations[last].wallFoot.DistanceTo(canal.pinch)) {
		return last
	}
	return 0
}

// runoutEndVertices identifies the rim's open-end vertex (removed, replaced by the arc) and its
// pinch vertex (kept — the band tapers into it).
func runoutEndVertices(rim *topo.Edge, canal *ellipticConeCanal) (voOld, vpOld *topo.Vertex) {
	sv, ev := rim.StartVertex(), rim.EndVertex()
	if float64(sv.Point().DistanceTo(canal.pinch)) <= float64(ev.Point().DistanceTo(canal.pinch)) {
		return ev, sv
	}
	return sv, ev
}

// bindRunoutEnd builds the open end: foot vertices on the two shortened host rulings, the
// section arc between them, and the side-face verification.
func (g *coneCanalRebuild) bindRunoutEnd(voOld *topo.Vertex, openIdx int) (*coneRunout, string) {
	wf0 := g.canal.stations[openIdx].wallFoot
	cf0 := g.canal.stations[openIdx].coneFoot
	wallSeamE, _ := wallSeam(g.canal.wallF, g.rim, voOld)
	coneSeamE, _ := wallSeam(g.canal.coneF, g.rim, voOld)
	if wallSeamE == nil || coneSeamE == nil {
		return nil, "elliptic cone canal runout: open end has no host boundary rulings to land on"
	}
	sideF, reason := runoutSideFace(wallSeamE, coneSeamE, g.canal.wallF, g.canal.coneF)
	if reason != "" {
		return nil, reason
	}
	fwdFromWall, ok := resolveSideArcDirection(sideF, wallSeamE, coneSeamE)
	if !ok {
		return nil, "elliptic cone canal runout: open-end rulings are not adjacent on the side face"
	}
	if reason := g.verifyRunoutPlane(sideF, wf0, cf0, openIdx); reason != "" {
		return nil, reason
	}
	ro, reason := g.buildRunoutEnd(voOld, wallSeamE, coneSeamE, sideF, wf0, cf0, openIdx)
	if reason != "" {
		return nil, reason
	}
	ro.arcFwdFromWall = fwdFromWall
	return ro, ""
}

// runoutSideFace is the one face sharing BOTH open-end rulings that is neither canal host — the
// planar face the runout section imprints.
func runoutSideFace(wallSeamE, coneSeamE *topo.Edge, wallF, coneF *topo.Face) (*topo.Face, string) {
	for _, f := range wallSeamE.Faces() {
		if f == wallF || f == coneF {
			continue
		}
		if slices.Contains(coneSeamE.Faces(), f) {
			return f, ""
		}
	}
	return nil, "elliptic cone canal runout: the two open-end rulings share no side face"
}

// verifyRunoutPlane checks the open-end section arc lies in the side face's plane (feet and arc
// shoulder within the band's geometric slack) — the honesty gate for the imprint.
func (g *coneCanalRebuild) verifyRunoutPlane(sideF *topo.Face, wf0, cf0 math.Point3, openIdx int) string {
	pl, isPlane := sideF.Geometry().(geom.Plane)
	if !isPlane {
		return fmt.Sprintf("elliptic cone canal runout: side face %d is %T, need a plane", sideF.ID(), sideF.Geometry())
	}
	n := pl.Normal().AsUnit()
	c := float64(math.P3(0, 0, 0).VectorTo(pl.Origin).Dot(n.AsVector()))
	slack := ellipticRimEnvelopeCoef * g.res.Weld()
	shoulder := g.canal.loft.Surf.PointAt(0.5, g.canal.loft.VParams[openIdx])
	for _, p := range []math.Point3{wf0, cf0, shoulder} {
		if d := stdmath.Abs(float64(math.P3(0, 0, 0).VectorTo(p).Dot(n.AsVector())) - c); d > slack {
			return fmt.Sprintf("elliptic cone canal runout: section point %v is %g off the side plane (slack %g)", p, d, slack)
		}
	}
	return ""
}

// buildRunoutEnd creates the foot vertices, shortens the two host rulings onto them, and builds
// the section arc edge, wiring the side face's substitutions.
func (g *coneCanalRebuild) buildRunoutEnd(voOld *topo.Vertex, wallSeamE, coneSeamE *topo.Edge, sideF *topo.Face, wf0, cf0 math.Point3, openIdx int) (*coneRunout, string) {
	lin := func(role string) topo.Lineage { return topo.NewLineage(topo.Tok("conecanal", role, 0)) }
	wfV := g.bld.AddVertex(wf0, lin("runoutwf"))
	cfV := g.bld.AddVertex(cf0, lin("runoutcf"))
	if reason := g.shortenRuling(wallSeamE, voOld, wfV, "wall"); reason != "" {
		return nil, reason
	}
	if reason := g.shortenRuling(coneSeamE, voOld, cfV, "cone"); reason != "" {
		return nil, reason
	}
	arcC, err := geom.SurfaceIsoCurve(g.canal.loft.Surf, false, g.canal.loft.VParams[openIdx])
	if err != nil {
		return nil, fmt.Sprintf("elliptic cone canal runout: section arc extraction: %v", err)
	}
	arcE := g.bld.AddEdge(arcC, wfV, cfV, lin("runoutarc"))
	return &coneRunout{sideF: sideF, wallSeamE: wallSeamE, coneSeamE: coneSeamE, arcE: arcE, arcFwdFromWall: true}, ""
}

// shortenRuling verifies the open-end foot sits ON the host boundary ruling and replaces the
// ruling with its retained sub-segment (the foot vertex takes the rim-corner end).
func (g *coneCanalRebuild) shortenRuling(seamE *topo.Edge, voOld *topo.Vertex, footV *topo.Vertex, role string) string {
	ls, isLine := seamE.Geometry().(geom.LineSegment)
	if !isLine {
		return fmt.Sprintf("elliptic cone canal runout: %s boundary edge %d is %T, need a straight ruling", role, seamE.ID(), seamE.Geometry())
	}
	if d := distPointToSegment(footV.Point(), ls); d > ellipticRimEnvelopeCoef*g.res.Weld() {
		return fmt.Sprintf("elliptic cone canal runout: %s foot %v is %g off its boundary ruling", role, footV.Point(), d)
	}
	g.seamRepl[seamE] = g.reaimedSeamEdge(seamE, voOld, seamFarVertex(seamE, voOld), footV)
	return ""
}

// seamFarVertex is the ruling's vertex away from the removed rim corner.
func seamFarVertex(seamE *topo.Edge, voOld *topo.Vertex) *topo.Vertex {
	if seamE.StartVertex() == voOld {
		return seamE.EndVertex()
	}
	return seamE.StartVertex()
}

// distPointToSegment is the distance from p to the closed segment.
func distPointToSegment(p math.Point3, ls geom.LineSegment) float64 {
	lo, hi := ls.Domain()
	a, b := ls.PointAt(lo), ls.PointAt(hi)
	ab := a.VectorTo(b)
	t := a.VectorTo(p).Dot(ab) / ab.Dot(ab)
	t = stdmath.Max(0, stdmath.Min(1, t))
	return float64(p.DistanceTo(a.TranslateBy(ab.Scale(t))))
}

// bindRunoutRails builds the two open rails (open-end foot → pinch vertex) and the band loop.
func (g *coneCanalRebuild) bindRunoutRails(ro *coneRunout, openIdx int, pinchV *topo.Vertex) string {
	wallRail, coneRail, reason := g.railCurves()
	if reason != "" {
		return reason
	}
	wfV, cfV := ro.arcE.StartVertex(), ro.arcE.EndVertex()
	lin := func(role string) topo.Lineage { return topo.NewLineage(topo.Tok("conecanal", role, 0)) }
	wE := g.addRailEdge(wallRail, openIdx, wfV, pinchV, lin("wallrail"))
	cE := g.addRailEdge(coneRail, openIdx, cfV, pinchV, lin("conerail"))
	wRev, r1 := g.hostWalkReversed(g.canal.wallF, wallRail)
	cRev, r2 := g.hostWalkReversed(g.canal.coneF, coneRail)
	if r1 != "" || r2 != "" {
		return r1 + r2
	}
	g.rimUses[g.canal.wallF] = []topo.Use{{Edge: wE, Reversed: wRev}}
	g.rimUses[g.canal.coneF] = []topo.Use{{Edge: cE, Reversed: cRev}}
	return g.assembleRunoutBandLoop(ro, wE, cE, wRev, cRev)
}

// addRailEdge builds a rail edge with its open-end vertex first when the loft walked from the
// open end (openIdx 0), or the pinch vertex first otherwise.
func (g *coneCanalRebuild) addRailEdge(rail geom.BSplineCurve, openIdx int, footV, pinchV *topo.Vertex, lin topo.Lineage) *topo.Edge {
	if openIdx == 0 {
		return g.bld.AddEdge(rail, footV, pinchV, lin)
	}
	return g.bld.AddEdge(rail, pinchV, footV, lin)
}

// assembleRunoutBandLoop chains the band's three boundary uses (arc + two rails, each
// antiparallel to its host face's use) into a contiguous cycle. The side face's arc direction
// was resolved from its ORIGINAL loop order (resolveSideArcDirection), so the band mirrors it.
func (g *coneCanalRebuild) assembleRunoutBandLoop(ro *coneRunout, wE, cE *topo.Edge, wRev, cRev bool) string {
	uses := []topo.Use{{Edge: ro.arcE, Reversed: ro.arcFwdFromWall}, {Edge: wE, Reversed: !wRev}, {Edge: cE, Reversed: !cRev}}
	chained, ok := chainUses(uses)
	if !ok {
		return "elliptic cone canal runout: band boundary does not chain into one cycle"
	}
	g.bandLoop = chained
	return ""
}

// resolveSideArcDirection reads the side face's ORIGINAL loop: when its walk passes the removed
// corner from the wall ruling onto the cone ruling, the inserted arc runs wall-foot → cone-foot
// forward there (and the band takes the mirror). ok=false when the two rulings are not adjacent
// in any side-face loop — the corner is not the simple two-ruling junction this vein models.
func resolveSideArcDirection(sideF *topo.Face, wallSeamE, coneSeamE *topo.Edge) (fwdFromWall, ok bool) {
	for _, l := range sideF.Loops() {
		src := l.EdgeUses()
		for i, u := range src {
			next := src[(i+1)%len(src)]
			if u.Edge() == wallSeamE && next.Edge() == coneSeamE {
				return true, true
			}
			if u.Edge() == coneSeamE && next.Edge() == wallSeamE {
				return false, true
			}
		}
	}
	return false, false
}

// useStart / useEnd are a use's traversal endpoints.
func canalUseStart(u topo.Use) *topo.Vertex {
	if u.Reversed {
		return u.Edge.EndVertex()
	}
	return u.Edge.StartVertex()
}
func canalUseEnd(u topo.Use) *topo.Vertex {
	if u.Reversed {
		return u.Edge.StartVertex()
	}
	return u.Edge.EndVertex()
}

// chainUses orders uses into one contiguous cycle, flipping uses when needed only between
// distinct vertices (uses touching the same vertex twice keep their given direction).
func chainUses(uses []topo.Use) ([]topo.Use, bool) {
	if len(uses) == 0 {
		return nil, false
	}
	out := []topo.Use{uses[0]}
	rest := append([]topo.Use{}, uses[1:]...)
	for len(rest) > 0 {
		at := canalUseEnd(out[len(out)-1])
		idx := indexOfUseStarting(rest, at)
		if idx < 0 {
			return nil, false
		}
		out = append(out, rest[idx])
		rest = append(rest[:idx], rest[idx+1:]...)
	}
	return out, canalUseEnd(out[len(out)-1]) == canalUseStart(out[0])
}

// indexOfUseStarting finds a use starting at v.
func indexOfUseStarting(uses []topo.Use, v *topo.Vertex) int {
	for i, u := range uses {
		if canalUseStart(u) == v {
			return i
		}
	}
	return -1
}

// copyRunoutFace copies one face, substituting the canal edges and inserting the section arc
// into the side face's loop at the removed corner.
func (g *coneCanalRebuild) copyRunoutFace(f *topo.Face, ro *coneRunout) string {
	if f != ro.sideF {
		return g.copyFaceWithRails(f)
	}
	specs := make([]topo.LoopSpec, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		uses, reason := g.sideFaceLoopUses(l, ro)
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

// sideFaceLoopUses maps the side face's loop, inserting the arc use between the two shortened
// rulings where the removed corner sat.
func (g *coneCanalRebuild) sideFaceLoopUses(l *topo.Loop, ro *coneRunout) ([]topo.Use, string) {
	src := l.EdgeUses()
	uses := make([]topo.Use, 0, len(src)+1)
	for i, u := range src {
		mapped, reason := g.mappedSideUse(u)
		if reason != "" {
			return nil, reason
		}
		uses = append(uses, mapped)
		next := src[(i+1)%len(src)]
		if isSeamPair(u.Edge(), next.Edge(), ro) {
			uses = append(uses, sideArcUse(u.Edge(), ro))
		}
	}
	return uses, ""
}

// mappedSideUse maps one side-face use onto the rebuilt edges (the rim never borders the side
// face, so only seam replacements and plain copies occur).
func (g *coneCanalRebuild) mappedSideUse(u *topo.EdgeUse) (topo.Use, string) {
	if repl := g.seamRepl[u.Edge()]; repl != nil {
		return topo.Use{Edge: repl, Reversed: u.Reversed()}, ""
	}
	if g.edges[u.Edge()] == nil {
		return topo.Use{}, fmt.Sprintf("elliptic cone canal runout: side face use of un-copied edge %d", u.Edge().ID())
	}
	return topo.Use{Edge: g.edges[u.Edge()], Reversed: u.Reversed()}, ""
}

// isSeamPair reports whether consecutive loop edges are exactly the two shortened rulings (the
// corner the arc replaces sits between them).
func isSeamPair(a, b *topo.Edge, ro *coneRunout) bool {
	return (a == ro.wallSeamE && b == ro.coneSeamE) || (a == ro.coneSeamE && b == ro.wallSeamE)
}

// sideArcUse is the arc's use in the side-face loop: forward (wall-foot → cone-foot) when the
// loop arrives on the wall-side ruling, reversed otherwise.
func sideArcUse(arrivedOn *topo.Edge, ro *coneRunout) topo.Use {
	if arrivedOn == ro.wallSeamE {
		return topo.Use{Edge: ro.arcE}
	}
	return topo.Use{Edge: ro.arcE, Reversed: true}
}
