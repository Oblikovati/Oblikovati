// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved-face miter body weld (families B/C). Two equal-radius rolling-ball arms — one exact torus,
// one exact cylinder — sharing a face MUTUALLY TRIM along the torus∩cylinder seam (fillet_miter_curved.go).
// This assembles the watertight solid: the two trimmed arm faces (each bounded by its two host contact
// rails, the shared seam, and its far cross-section trim), the receded shared face (bitten by BOTH arms'
// shared rails meeting at the seam top), the receded outer faces and far caps, and every untouched face
// carried through. Any decline returns a do-no-harm reason (the caller keeps the clean floor, never a
// partial body) — so a far end that caps against a curved face (P5's cylinder caps, an oblique/canal
// runout out of this analytic scope) floors honestly here.

// curvedMiterOf returns the curved-contact miter corner (families B/C) whose two arms are exactly the
// filleted edges, or nil when this corner set is not a 2-arm curved miter (the caller keeps the
// trihedral / single-arm paths). It fires only when there are exactly two picked edges and a solved
// curved miter joins them at their shared vertex.
func curvedMiterOf(fils []edgeFillet, miters map[uint64]*cornerMiter) *cornerMiter {
	if len(fils) != 2 {
		return nil
	}
	for _, m := range miters {
		if m == nil || m.curved == nil {
			continue
		}
		if miterJoinsEdges(m, fils[0].edge, fils[1].edge) {
			return m
		}
	}
	return nil
}

// miterJoinsEdges reports whether the curved miter's two arm edges are exactly e0 and e1.
func miterJoinsEdges(m *cornerMiter, e0, e1 *topo.Edge) bool {
	a, b := m.curved.torEdge, m.curved.cylEdge
	return (a == e0 && b == e1) || (a == e1 && b == e0)
}

// miterArmSide is one curved-miter arm's assembled boundary data: its exact surface, the outer host,
// the two host contact rails (shared-face rail farShared→sTop, outer-face rail farOuter→sBot), and the
// far cross-section runout (trim + capping face) terminating the non-miter end.
type miterArmSide struct {
	surface geom.Surface
	edge    *topo.Edge
	outer   *topo.Face
	railSh  endSeg // shared-face contact rail: farShared → sTop
	railOut endSeg // outer-face contact rail: farOuter → sBot
	run     armRunout
}

// curvedMiterBody welds the 2-arm curved miter into a watertight solid, or returns a do-no-harm reason.
// It builds each arm's rails + far runout, then the arm faces, the twice-bitten shared face, and the
// receded outer/cap hosts, and assembles + certifies via the caller.
func curvedMiterBody(body *topo.Body, m *cornerMiter, res Resolution) (*topo.Body, string) {
	c := m.curved
	r := c.arms.tor.MinorRadius
	torSide, reason := buildMiterArmSide(m, c.arms.tor, c.torEdge, r, res)
	if reason != "" {
		return nil, reason
	}
	cylSide, reason := buildMiterArmSide(m, c.arms.cyl, c.cylEdge, r, res)
	if reason != "" {
		return nil, reason
	}
	faces, reason := curvedMiterFaces(body, m, torSide, cylSide, r, res)
	if reason != "" {
		return nil, reason
	}
	return assembleBody(faces), ""
}

// buildMiterArmSide solves one arm's two host contact rails and its far cross-section runout: the far
// feet from the arm ball centre at the non-miter vertex, the shared/outer rails from those feet to the
// seam endpoints (sTop/sBot), and the far termination through the general far-runout engine (which
// floors honestly on an oblique/curved cap — P5's cylinder-capped far ends). reason names any decline.
func buildMiterArmSide(m *cornerMiter, arm geom.Surface, edge *topo.Edge, r float64, res Resolution) (miterArmSide, string) {
	outer := otherFace(edge, m.shared)
	if outer == nil {
		return miterArmSide{}, fmt.Sprintf("curved miter: edge %d has no outer face opposite the shared face", edge.ID())
	}
	tol := res.Weld() * r
	far := farVertexNotVid(edge, m.vertex.ID())
	ball, ok := armBallCenter(arm, far)
	if !ok {
		return miterArmSide{}, fmt.Sprintf("curved miter: arm spine undefined at far vertex of edge %d", edge.ID())
	}
	farShared, okS := armRunoutFoot(m.shared, ball, r, tol)
	farOuter, okO := armRunoutFoot(outer, ball, r, tol)
	if !okS || !okO {
		return miterArmSide{}, fmt.Sprintf("curved miter: far foot not tangent on a host of edge %d (shared=%v outer=%v)", edge.ID(), okS, okO)
	}
	return assembleMiterArmSide(m, arm, edge, outer, farShared, farOuter, r, res)
}

// assembleMiterArmSide builds the two contact rails (far foot → seam endpoint) and the far runout for
// one arm, given its far feet. Split from buildMiterArmSide to stay within funlen.
func assembleMiterArmSide(m *cornerMiter, arm geom.Surface, edge *topo.Edge, outer *topo.Face, farShared, farOuter math.Point3, r float64, res Resolution) (miterArmSide, string) {
	sTop, sBot := m.seam[0], m.seam[len(m.seam)-1]
	railSh, okA := armRunoutRail(m.shared, edge, arm, farShared, sTop, res)
	railOut, okB := armRunoutRail(outer, edge, arm, farOuter, sBot, res)
	if !okA || !okB {
		return miterArmSide{}, fmt.Sprintf("curved miter: a host contact rail could not be built on edge %d (shared=%v outer=%v)", edge.ID(), okA, okB)
	}
	ef := edgeFillet{a: m.shared, b: outer, edge: edge, armSurface: arm}
	filed := map[uint64]bool{m.curved.torEdge.ID(): true, m.curved.cylEdge.ID(): true}
	_, _, run, ok, reason := armFarRunout(ef, cornerWeld{center: m.vertex.Point(), radius: r}, railSh, railOut, filed, res)
	if !ok {
		return miterArmSide{}, fmt.Sprintf("curved miter: far runout declined on edge %d: %s", edge.ID(), reason)
	}
	return miterArmSide{surface: arm, edge: edge, outer: outer, railSh: railSh, railOut: railOut, run: run}, ""
}

// curvedMiterFaces builds every result face: the two trimmed arm faces, the twice-bitten shared face,
// each arm's receded outer host and far cap, and every untouched face carried through verbatim.
func curvedMiterFaces(body *topo.Body, m *cornerMiter, tor, cyl miterArmSide, r float64, res Resolution) ([]filletFace, string) {
	tol := res.Weld() * r
	shared, ok := sharedMiterRetrim(m.shared, tor.railSh, cyl.railSh, m.vertex.Point(), tol)
	if !ok {
		return nil, "curved miter: shared-face two-rail retrim declined"
	}
	bites := miterHostBites(m, tor, cyl)
	out := []filletFace{miterArmFace(tor, m.seam), miterArmFace(cyl, m.seam), shared}
	for _, f := range body.Faces() {
		if f == m.shared {
			continue // replaced by the twice-bitten retrim
		}
		bite, bitten := bites[f]
		if !bitten {
			out = append(out, passthroughFace(f))
			continue
		}
		ff, ok := singleRunoutHostFace(f, bite.seg, bite.avoid, tol)
		if !ok {
			return nil, fmt.Sprintf("curved miter: host %T retrim declined", f.Geometry())
		}
		out = append(out, ff)
	}
	return out, ""
}

// miterHostBite is one bitten host's recede rail and the removed-span anchor vertex.
type miterHostBite struct {
	seg   endSeg
	avoid math.Point3
}

// miterHostBites maps each arm's outer host and far cap to its recede bite (the contact rail on an
// outer face, the far cross-section trim on a cap) plus the vertex the removed span carries.
func miterHostBites(m *cornerMiter, tor, cyl miterArmSide) map[*topo.Face]miterHostBite {
	v := m.vertex.Point()
	bites := map[*topo.Face]miterHostBite{}
	for _, s := range []miterArmSide{tor, cyl} {
		far := farVertexNotVid(s.edge, m.vertex.ID())
		bites[s.outer] = miterHostBite{seg: reverseEndSegs([]endSeg{s.railOut})[0], avoid: v} // outer host recedes along sBot→farOuter
		if s.run.capping != nil {
			bites[s.run.capping] = miterHostBite{seg: s.run.trim, avoid: far}
		}
	}
	return bites
}

// miterArmFace emits one trimmed arm face: the shared-face rail (farShared→sTop), the seam
// (sTop→sBot, the chord list SHARED with the other arm so the weld is watertight), the outer-face
// rail reversed (sBot→farOuter), and the far cross-section trim reversed (farOuter→farShared).
func miterArmFace(side miterArmSide, seam []math.Point3) filletFace {
	segs := []endSeg{side.railSh}
	segs = append(segs, seamEndSegs(seam)...)
	segs = append(segs, reverseEndSegs([]endSeg{side.railOut})...)
	segs = append(segs, reverseEndSegs([]endSeg{side.run.trim})...)
	return filletFace{surface: side.surface, loops: []filletLoop{loopFromSegs(segs)}, parent: filletEdgeProvenance(side.edge)}
}

// seamEndSegs turns the seam polyline into straight chord segments sTop→sBot.
func seamEndSegs(seam []math.Point3) []endSeg {
	segs := make([]endSeg, 0, len(seam)-1)
	for i := 0; i+1 < len(seam); i++ {
		segs = append(segs, endSeg{from: seam[i], to: seam[i+1]})
	}
	return segs
}

// sharedMiterRetrim recedes the shared face: it removes the corner span carrying the miter vertex v and
// splices in BOTH arms' shared-face contact rails, which meet at the seam top sTop — so the shared face
// recedes to the two rails joined at their common tangent point. railTor/railCyl run farShared→sTop.
func sharedMiterRetrim(shared *topo.Face, railTor, railCyl endSeg, v math.Point3, tol float64) (filletFace, bool) {
	bitten := hostBittenLoop(shared, v, tol)
	outer := outerHostLoop(shared)
	if bitten == nil || outer == nil {
		return filletFace{}, false
	}
	bite := append([]endSeg{railTor}, reverseEndSegs([]endSeg{railCyl})...) // farShared_tor → sTop → farShared_cyl
	far, ok := farPathSegs(segsFromLoop(bitten), railCyl.from, railTor.from, v, tol)
	if !ok {
		return filletFace{}, false
	}
	retrim := loopFromSegs(append(bite, far...))
	loops := hostLoopsWithRetrim(shared, bitten, outer, retrim)
	return filletFace{surface: shared.Geometry(), loops: loops, parent: shared.Lineage()}, true
}
