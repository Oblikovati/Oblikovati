// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// wrappedPadSolid builds a wrapped emboss pad as a watertight solid whose INNER and OUTER faces are
// the genuine curved surfaces the profile was wrapped onto (innerCap/outerCap — cylinders at each
// radius), not a flat cap. A single planar face over a wrapped (curved) loop is a non-coplanar face on
// a plane surface — an invalid B-rep the mesh oracle tolerates but the analytic point classifier
// correctly rejects (exact-containment-oracle-batch). The side walls between corresponding inner/outer
// loop points stay planar (a warped segment is split into two triangles), exactly the sweptSolid
// faceting. The two loops carry the SAME point count and correspondence (same sketch point per index).
func wrappedPadSolid(innerLoop, outerLoop []math.Point3, innerCap, outerCap geom.Surface, feat string) (*topo.Body, error) {
	body, err := buildWrappedPad(innerLoop, outerLoop, innerCap, outerCap, feat)
	if err != nil {
		return nil, err
	}
	if query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume < 0 {
		// The cage is consistently wound but inside-out; reversing both loops flips every face's
		// normal together (the manual analog of sweptSolid's reverseFaces volume flip).
		return buildWrappedPad(reversedPts(innerLoop), reversedPts(outerLoop), innerCap, outerCap, feat)
	}
	return body, nil
}

// buildWrappedPad wires the cage: inner/outer loop vertices, the circumferential and radial edges, the
// two curved caps and the planar side walls. Every edge is used by exactly two faces in opposite
// directions, so the cage is manifold; the inner cap is reversed so its material faces the outer cap.
func buildWrappedPad(innerLoop, outerLoop []math.Point3, innerCap, outerCap geom.Surface, feat string) (*topo.Body, error) {
	n := len(innerLoop)
	lin := func(kind string, i int) topo.Lineage { return topo.NewLineage(topo.Tok(feat, kind, i)) }
	bld := topo.NewBuilder(true, lin("body", 0))
	iv, ov := make([]*topo.Vertex, n), make([]*topo.Vertex, n)
	for i := range n {
		iv[i] = bld.AddVertex(innerLoop[i], lin("iv", i))
		ov[i] = bld.AddVertex(outerLoop[i], lin("ov", i))
	}
	ie, oe, re := padEdges(bld, innerLoop, outerLoop, iv, ov, lin)
	addPadCaps(bld, innerCap, outerCap, ie, oe, lin)
	addPadWalls(bld, innerLoop, outerLoop, iv, ov, ie, oe, re, lin)
	return bld.Build(), nil
}

// padEdges creates the inner-loop, outer-loop and radial edges of the pad (n of each).
func padEdges(bld *topo.Builder, innerLoop, outerLoop []math.Point3, iv, ov []*topo.Vertex,
	lin func(string, int) topo.Lineage) (ie, oe, re []*topo.Edge) {
	n := len(innerLoop)
	ie, oe, re = make([]*topo.Edge, n), make([]*topo.Edge, n), make([]*topo.Edge, n)
	for i := range n {
		j := (i + 1) % n
		ie[i] = bld.AddEdge(geom.NewLineSegment(innerLoop[i], innerLoop[j]), iv[i], iv[j], lin("ie", i))
		oe[i] = bld.AddEdge(geom.NewLineSegment(outerLoop[i], outerLoop[j]), ov[i], ov[j], lin("oe", i))
		re[i] = bld.AddEdge(geom.NewLineSegment(innerLoop[i], outerLoop[i]), iv[i], ov[i], lin("re", i))
	}
	return ie, oe, re
}

// addPadCaps adds the two curved cap faces. The inner cap walks its loop forward (and is reversed, so
// its material-outward normal points away from the outer cap); the outer cap walks the opposite way so
// each shared loop edge is used once in each direction.
func addPadCaps(bld *topo.Builder, innerCap, outerCap geom.Surface, ie, oe []*topo.Edge,
	lin func(string, int) topo.Lineage) {
	n := len(ie)
	inner := make([]topo.Use, n)
	outer := make([]topo.Use, n)
	for i := range n {
		inner[i] = topo.Fwd(ie[i])
		outer[i] = topo.Rev(oe[n-1-i])
	}
	bld.AddReversedFace(innerCap, lin("icap", 0), topo.OuterLoop(inner...))
	bld.AddFace(outerCap, lin("ocap", 0), topo.OuterLoop(outer...))
}

// addPadWalls adds one side-wall face per loop segment, planar when its four corners are coplanar or
// two triangles when the segment warps — sharing ie[i]/oe[i] with the caps (opposite direction) and
// the radial edges with the neighbouring walls.
func addPadWalls(bld *topo.Builder, innerLoop, outerLoop []math.Point3, iv, ov []*topo.Vertex,
	ie, oe, re []*topo.Edge, lin func(string, int) topo.Lineage) {
	n := len(ie)
	for i := range n {
		j := (i + 1) % n
		if quadPlanar(innerLoop[i], innerLoop[j], outerLoop[j], outerLoop[i]) {
			pl, err := geom.PlaneByThreePoints(innerLoop[i], innerLoop[j], outerLoop[i])
			if err == nil {
				bld.AddFace(pl, lin("wall", i), topo.OuterLoop(topo.Fwd(re[i]), topo.Fwd(oe[i]), topo.Rev(re[j]), topo.Rev(ie[i])))
				continue
			}
		}
		addWarpedWall(bld, innerLoop, outerLoop, iv, ov, ie, oe, re, i, j, lin)
	}
}

// addWarpedWall splits a non-planar wall segment into two exact-planar triangles across an internal
// diagonal iv[i]→ov[j], reusing the SAME four boundary edges (re[i], oe[i], re[j], ie[i]) the planar
// quad would, in the same directions — so the cage stays manifold whether a segment is planar or not.
// Triangle A = (iv[i], ov[i], ov[j]); triangle B = (iv[i], ov[j], iv[j]); the diagonal is used once
// in each direction.
//
// Each triangle's PLANE is named in the reverse of the order its own loop walks, because that is the
// sense the rest of the cage carries: the planar-quad wall above is named (i0, i1, o0) against a loop
// running i0→o0→o1→i1, and the two caps are wired to match. Naming a triangle in loop order instead —
// the reading that looks right in isolation — gave the split faces a material side OPPOSITE every
// other face of the pad. Nothing downstream saw it: Validate checks loop TRAVERSAL, which stayed
// manifold, and the mesh reads orientation off the loop. What it broke was the outward vector area,
// which a closed shell owes the divergence theorem: the pad missed closing by 7.1e-3 relative and
// integrated to 0.5438 cm³ where 0.3735 is right, so every analytic mass property of a wrapped emboss
// fell back to the mesh (Oblikovati/Oblikovati#3503; the missing invariant is #3504).
func addWarpedWall(bld *topo.Builder, innerLoop, outerLoop []math.Point3, iv, ov []*topo.Vertex,
	ie, oe, re []*topo.Edge, i, j int, lin func(string, int) topo.Lineage) {
	diag := bld.AddEdge(geom.NewLineSegment(innerLoop[i], outerLoop[j]), iv[i], ov[j], lin("wdiag", i))
	if tA, err := geom.PlaneByThreePoints(innerLoop[i], outerLoop[j], outerLoop[i]); err == nil {
		bld.AddFace(tA, lin("wallA", i), topo.OuterLoop(topo.Fwd(re[i]), topo.Fwd(oe[i]), topo.Rev(diag)))
	}
	if tB, err := geom.PlaneByThreePoints(innerLoop[i], innerLoop[j], outerLoop[j]); err == nil {
		bld.AddFace(tB, lin("wallB", i), topo.OuterLoop(topo.Fwd(diag), topo.Rev(re[j]), topo.Rev(ie[i])))
	}
}

// reversedPts returns pts in reverse order (used to flip the cage inside-out).
func reversedPts(pts []math.Point3) []math.Point3 {
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}
