// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
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
	if padHoldsMaterialOutside(body) {
		// The cage is consistently wound but inside-out; reversing both loops flips every face's
		// derived sense together (the manual analog of sweptSolid's reverseFaces volume flip).
		return buildWrappedPad(reversedPts(innerLoop), reversedPts(outerLoop), innerCap, outerCap, feat)
	}
	return body, nil
}

// padHoldsMaterialOutside reports whether the cage's own faces bound a NEGATIVE volume, which says the
// material is on the outside and the whole thing is inside-out.
//
// It integrates the ANALYTIC faces, not a mesh. The senses under test are a modelling attribute, and a
// tessellation is derived from the B-rep rather than an authority over it — and here the mesh could not
// answer the question at all, because a mesh takes each face's orientation from its LOOP and the stored
// senses are exactly what is in doubt. This is OrientClosedSolid's shape: decide from the geometry, and
// when the geometry cannot say, leave the body alone rather than guess.
func padHoldsMaterialOutside(b *topo.Body) bool {
	for _, shell := range b.Shells() {
		if v, ok := query.AnalyticShellVolume(shell); ok && v < 0 {
			return true
		}
	}
	return false
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
	ie, oe, re := padEdges(bld, innerLoop, outerLoop, iv, ov, innerCap, outerCap, lin)
	addPadCaps(bld, innerCap, outerCap, ie, oe, innerLoop, outerLoop, lin)
	addPadWalls(bld, innerLoop, outerLoop, iv, ov, ie, oe, re, lin)
	return bld.Build(), nil
}

// padEdges creates the inner-loop, outer-loop and radial edges of the pad (n of each).
func padEdges(bld *topo.Builder, innerLoop, outerLoop []math.Point3, iv, ov []*topo.Vertex,
	innerCap, outerCap geom.Surface, lin func(string, int) topo.Lineage) (ie, oe, re []*topo.Edge) {
	n := len(innerLoop)
	res := geom.ResolutionForPoints(append(append([]math.Point3{}, innerLoop...), outerLoop...))
	ie, oe, re = make([]*topo.Edge, n), make([]*topo.Edge, n), make([]*topo.Edge, n)
	for i := range n {
		j := (i + 1) % n
		ie[i] = bld.AddEdge(capSectionCurve(innerCap, innerLoop, outerLoop, i, false, res), iv[i], iv[j], lin("ie", i))
		oe[i] = bld.AddEdge(capSectionCurve(outerCap, innerLoop, outerLoop, i, true, res), ov[i], ov[j], lin("oe", i))
		re[i] = bld.AddEdge(geom.NewLineSegment(innerLoop[i], outerLoop[i]), iv[i], ov[i], lin("re", i))
	}
	return ie, oe, re
}

// capSectionCurve is the curve one loop edge follows: the section of its own WALL's plane with the cap
// the edge lies on, restricted to the arc between the two wrapped points. Both points lie on that
// plane and on that cap by construction, so the section passes through them exactly — and the edge
// then lies on BOTH faces it bounds.
//
// A straight segment between the same two points does not lie on the cap at all; it is a chord
// through the solid, so the cap face it bounded was not valid geometry however small the sag. On a
// 1 cm glyph edge at radius 15 the chord left the cone by 8.3e-3 cm (Oblikovati/Oblikovati#3503).
//
// It falls back to the chord when the pair carries no analytic section — the intersector defers a
// cone's parabolic boundary — so the pad still builds, with that face no better than it was.
func capSectionCurve(capSurf geom.Surface, innerLoop, outerLoop []math.Point3, i int, outer bool,
	res geom.Resolution) geom.Curve3 {
	a, b := innerLoop[i], innerLoop[(i+1)%len(innerLoop)]
	if outer {
		a, b = outerLoop[i], outerLoop[(i+1)%len(outerLoop)]
	}
	chord := geom.NewLineSegment(a, b)
	wall, ok := padWallPlane(innerLoop, outerLoop, i, outer)
	if !ok {
		return chord
	}
	sec, handled := geom.IntersectSurfacesAnalytic(wall, capSurf, res)
	if !handled {
		return chord
	}
	for _, c := range sec {
		if arc, got := geom.ConicArcBetween(c, a, b, a.Midpoint(b)); got && arcRunsBetween(arc, a, b, res) {
			return arc
		}
	}
	return chord
}

// padWallPlane is the plane the segment's wall lies in on one side: the quad's own when its four
// corners are coplanar, otherwise the triangle carrying that side's loop edge — A = (i0, o0, o1) for
// the outer, B = (i0, o1, i1) for the inner. It names the same planes addPadWalls builds the faces on,
// which is what makes a loop edge cut from it lie on both faces it bounds.
func padWallPlane(innerLoop, outerLoop []math.Point3, i int, outer bool) (geom.Plane, bool) {
	n := len(innerLoop)
	j := (i + 1) % n
	i0, i1, o0, o1 := innerLoop[i], innerLoop[j], outerLoop[i], outerLoop[j]
	if quadPlanar(i0, i1, o1, o0) {
		pl, err := geom.PlaneByThreePoints(i0, i1, o0)
		return pl, err == nil
	}
	if outer {
		pl, err := geom.PlaneByThreePoints(i0, o1, o0)
		return pl, err == nil
	}
	pl, err := geom.PlaneByThreePoints(i0, i1, o1)
	return pl, err == nil
}

// arcRunsBetween verifies the section actually passes THROUGH both endpoints. ConicParamAt projects
// onto the conic, it does not test membership, so a section curve that misses the pair would otherwise
// be accepted and store an edge whose curve never reaches its own vertices. The section and the
// wrapped point are independent computations, so the comparison is a Sew().
func arcRunsBetween(arc geom.Curve3, a, b math.Point3, res geom.Resolution) bool {
	tol := math.Scalar(res.Sew())
	return arc.PointAt(0).DistanceTo(a) <= tol && arc.PointAt(1).DistanceTo(b) <= tol
}

// addPadCaps adds the two curved cap faces. The inner cap walks its loop forward, the outer the
// opposite way, so each shared loop edge is used once in each direction; each is then minted with the
// sense its OWN loop implies, never with a fixed one.
func addPadCaps(bld *topo.Builder, innerCap, outerCap geom.Surface, ie, oe []*topo.Edge,
	innerLoop, outerLoop []math.Point3, lin func(string, int) topo.Lineage) {
	n := len(ie)
	inner := make([]topo.Use, n)
	outer := make([]topo.Use, n)
	fwd, rev := make([]math.Point3, n), make([]math.Point3, n)
	for i := range n {
		inner[i], fwd[i] = topo.Fwd(ie[i]), innerLoop[i]
		outer[i], rev[i] = topo.Rev(oe[n-1-i]), outerLoop[(n-i)%n]
	}
	addPadFace(bld, innerCap, capWindsClockwise(innerCap, fwd), lin("icap", 0), inner)
	addPadFace(bld, outerCap, capWindsClockwise(outerCap, rev), lin("ocap", 0), outer)
}

// addPadFace mints one face forward or reversed, so every caller states the sense as a value rather
// than by picking a different builder call.
func addPadFace(bld *topo.Builder, surf geom.Surface, reversed bool, lineage topo.Lineage, loop []topo.Use) {
	if reversed {
		bld.AddReversedFace(surf, lineage, topo.OuterLoop(loop...))
		return
	}
	bld.AddFace(surf, lineage, topo.OuterLoop(loop...))
}

// capWindsClockwise reports whether the loop runs CLOCKWISE in the cap surface's own (u, v) — which is
// to say the material-outward normal is the opposite of S_u×S_v, so the face must be minted reversed.
//
// This is the invariant OCCT keeps by construction: a face is FORWARD exactly when its outer wire runs
// counter-clockwise in the surface's parameter space (ShapeAnalysis::TotCross2D, and the shoelace this
// mirrors). Here it was a CONVENTION instead — the inner cap always reversed, the outer always forward
// — and the convention disagreed with the geometry. Every face of the cage was minted with a sense
// opposite its own winding, which is self-consistent as a shell and so passed unnoticed: the boolean
// then rebuilt the pad's WALLS in the arrangement's own (correct) convention while passing the cap
// through untouched, leaving exactly one face out of step in the result and its outward vector area
// 3.0e-4 short of closing (Oblikovati/Oblikovati#3504).
//
// u is unwrapped as the loop is walked, so a patch straddling the surface's seam still gives one
// continuous contour rather than a shoelace torn across the branch cut.
func capWindsClockwise(surf geom.Surface, loop []math.Point3) bool {
	if len(loop) < 3 {
		return false
	}
	uPeriod := surfaceUPeriod(surf)
	u0, v0 := surf.ParamAt(loop[0])
	area, pu, pv := 0.0, u0, v0
	for _, p := range loop[1:] {
		u, v := surf.ParamAt(p)
		u = nearestBranch(u, pu, uPeriod)
		area += pu*v - u*pv
		pu, pv = u, v
	}
	return area+pu*v0-u0*pv < 0
}

// nearestBranch moves u by whole periods onto the branch nearest prev, so a loop crossing the seam
// keeps travelling instead of jumping a period.
func nearestBranch(u, prev, period float64) float64 {
	if period <= 0 {
		return u
	}
	return u + period*stdmath.Round((prev-u)/period)
}

// surfaceUPeriod is the surface's period in u, read from its own domain: an angular parameter reports
// [0, 2π], and anything else reports 0 for "does not wrap". Only u is asked because a pad's caps are
// the wrap surfaces — a cylinder or a cone — whose v runs along the axis or the slant and never wraps.
func surfaceUPeriod(surf geom.Surface) float64 {
	lo, hi := surf.UDomain()
	if stdmath.Abs(hi-lo-2*stdmath.Pi) < uPeriodWeld {
		return 2 * stdmath.Pi
	}
	return 0
}

// uPeriodWeld is how close a domain span must be to a full turn to count as an angular parameter. It
// compares two ANGLES on a fixed constant, so it carries no model scale.
const uPeriodWeld = 1e-9 // tol:angular — a parameter domain either is a full turn or is not

// addPadWalls adds one side-wall face per loop segment, planar when its four corners are coplanar or
// two triangles when the segment warps — sharing ie[i]/oe[i] with the caps (opposite direction) and
// the radial edges with the neighbouring walls.
func addPadWalls(bld *topo.Builder, innerLoop, outerLoop []math.Point3, iv, ov []*topo.Vertex,
	ie, oe, re []*topo.Edge, lin func(string, int) topo.Lineage) {
	n := len(ie)
	for i := range n {
		j := (i + 1) % n
		if quadPlanar(innerLoop[i], innerLoop[j], outerLoop[j], outerLoop[i]) {
			pl, err := geom.PlaneByThreePoints(innerLoop[i], outerLoop[i], outerLoop[j])
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
	if tA, err := geom.PlaneByThreePoints(innerLoop[i], outerLoop[i], outerLoop[j]); err == nil {
		bld.AddFace(tA, lin("wallA", i), topo.OuterLoop(topo.Fwd(re[i]), topo.Fwd(oe[i]), topo.Rev(diag)))
	}
	if tB, err := geom.PlaneByThreePoints(innerLoop[i], outerLoop[j], innerLoop[j]); err == nil {
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
