// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// rebuildWithRimFillet rebuilds the body with the rim rounded: every vertex/edge/face is copied
// (the TransformBody pattern) EXCEPT the rim circle and the rim vertex (removed) and the wall seam
// (re-aimed to the receded cyl-tangent vertex). It then inserts the cyl-tangent + cap-tangent circles,
// the torus seam, re-trims the cylinder wall and cap onto them, and adds the torus band.
func rebuildWithRimFillet(b *topo.Body, rf *rimFillet) (*topo.Body, error) {
	return rebuildRim(b, rf, false)
}

// rebuildWithConcaveRimFillet is the CONCAVE dual, shared by two callers whose solvers both flag
// rimFillet.concave=true: the sphere/cone-host cove band (S2/S5, fillet_curved_closed_rim_concave.go —
// the rebuild recedes the curved host UP its bump and GROWS the plate hole out to the wider plane-contact
// circle) and the cylinder-host bore-lip mirror (K1, fillet_rim_concave.go's rimWithCapOrientation — the
// plate hole is ALREADY the wider R+r circle, so only the winding matters here). Two things differ from
// the convex path, both gated on concave=true so the convex J1 band stays byte-identical: the re-aimed
// curved-host seam stays on a SPHERE host as a meridian ARC (a cone/cylinder seam — K1 included — is
// still the straight ruling), and the torus band winds the OTHER way so the added/receded material keeps
// the signed volume positive.
func rebuildWithConcaveRimFillet(b *topo.Body, rf *rimFillet) (*topo.Body, error) {
	return rebuildRim(b, rf, true)
}

// rebuildRim is the shared rim-rebuild driver; concave selects the reversed seam/winding variants shared
// by the S2/S5 sphere/cone cove band and the K1-style cylinder bore-lip mirror.
func rebuildRim(b *topo.Body, rf *rimFillet, concave bool) (*topo.Body, error) {
	g := &rimBuild{
		rf: rf, concave: concave, bld: topo.NewBuilder(b.IsSolid(), b.Lineage()),
		verts: map[*topo.Vertex]*topo.Vertex{}, edges: map[*topo.Edge]*topo.Edge{},
		capRimReversed: capRimEdgeReversed(rf.cap, rf.rimEdge),
	}
	g.copyVerts(b)
	g.addRimVerts()
	g.copyEdges(b)
	g.addRimEdges()
	for _, f := range b.Faces() {
		g.copyFace(f)
	}
	g.addBandFace()
	return g.bld.Build(), nil
}

// rimBuild carries the in-progress rebuild: the old→new vertex/edge maps plus the new rim entities.
type rimBuild struct {
	rf      *rimFillet
	concave bool // S2/S5: sphere meridian-arc seam + reversed cove-band winding
	bld     *topo.Builder
	verts   map[*topo.Vertex]*topo.Vertex
	edges   map[*topo.Edge]*topo.Edge
	vc      *topo.Vertex // cyl-tangent seam vertex (replaces the rim vertex on the wall)
	vt      *topo.Vertex // cap-tangent seam vertex
	cylE    *topo.Edge   // cyl-tangent circle
	capE    *topo.Edge   // cap-tangent circle
	seamE   *topo.Edge   // torus seam arc vc→vt
	wallE   *topo.Edge   // re-aimed wall seam bottom→vc
	// capRimReversed is the ORIGINAL rim edge's Reversed flag as used by the cap face, captured before the
	// rebuild starts (see rimReplacementUse / addBandFace, #2006). It is the one piece of source-import
	// direction the rebuild keeps verbatim.
	capRimReversed bool
}

func (g *rimBuild) copyVerts(b *topo.Body) {
	for _, v := range b.Vertices() {
		if v == g.rf.rimV {
			continue // the rim vertex is replaced by the two tangent-circle seam vertices
		}
		g.verts[v] = g.bld.AddVertex(v.Point(), v.Lineage())
	}
}

func (g *rimBuild) addRimVerts() {
	g.vc = g.bld.AddVertex(g.rf.cylTan.PointAt(0), topo.NewLineage(topo.Tok("rimfillet", "vc", 0)))
	g.vt = g.bld.AddVertex(g.rf.capTan.PointAt(0), topo.NewLineage(topo.Tok("rimfillet", "vt", 0)))
}

func (g *rimBuild) copyEdges(b *topo.Body) {
	for _, e := range b.Edges() {
		if e == g.rf.rimEdge || e == g.rf.seamEdge {
			continue // rim removed; wall seam re-aimed (added in addRimEdges)
		}
		g.edges[e] = g.bld.AddEdge(e.Geometry(), g.verts[e.StartVertex()], g.verts[e.EndVertex()], e.Lineage())
	}
}

func (g *rimBuild) addRimEdges() {
	lin := func(role string) topo.Lineage { return topo.NewLineage(topo.Tok("rimfillet", role, 0)) }
	g.cylE = g.bld.AddEdge(g.rf.cylTan, g.vc, g.vc, lin("cyltan"))
	g.capE = g.bld.AddEdge(g.rf.capTan, g.vt, g.vt, lin("captan"))
	seam, _ := geom.Arc3dByThreePoints(g.rf.cylTan.PointAt(0), g.rf.seamMid, g.rf.capTan.PointAt(0))
	g.seamE = g.bld.AddEdge(seam, g.vc, g.vt, lin("seam"))
	bottom := g.verts[g.rf.bottomV]
	g.wallE = g.bld.AddEdge(g.wallSeamCurve(bottom.Point(), g.vc.Point()), bottom, g.vc, lin("wallseam"))
}

// wallSeamCurve is the re-aimed curved-host seam from the host's far vertex (bottom) to the receded
// contact vertex vc. On a cone/cylinder host it is the straight ruling (byte-identical to the convex J1
// / lone-rim path). On a concave SPHERE host it is the meridian ARC on the sphere (the host boundary must
// stay on the sphere, so a line — which cuts through the sphere interior — is wrong): the great-circle
// sub-arc through the on-sphere midpoint of bottom and vc.
func (g *rimBuild) wallSeamCurve(bottom, vc math.Point3) geom.Curve3 {
	sph, isSphere := g.rf.cyl.Geometry().(geom.Sphere)
	if !g.concave || !isSphere {
		return geom.NewLineSegment(bottom, vc)
	}
	db, e1 := math.UnitVector3FromVector(sph.Center.VectorTo(bottom))
	dc, e2 := math.UnitVector3FromVector(sph.Center.VectorTo(vc))
	if e1 != nil || e2 != nil {
		return geom.NewLineSegment(bottom, vc)
	}
	mid, err := math.UnitVector3FromVector(db.AsVector().Add(dc.AsVector()))
	if err != nil {
		return geom.NewLineSegment(bottom, vc) // antipodal degeneracy — fall back to the ruling
	}
	onSphere := sph.Center.TranslateBy(mid.AsVector().Scale(sph.Radius))
	arc, err := geom.Arc3dByThreePoints(bottom, onSphere, vc)
	if err != nil {
		return geom.NewLineSegment(bottom, vc)
	}
	return arc
}

// quarterTube is v=π/4 — the tube midpoint between the cyl-tangent contact (v=0) and the cap-tangent
// contact (v=π/2) of a convex rim, used as the seam arc's on-arc point.
const quarterTube = 0.7853981633974483

// threeQuarterTube is v=3π/4 — the tube midpoint between the cap-tangent contact (v=π/2, unchanged) and
// the CONCAVE bore-lip's cyl-tangent contact (v=π, the mirror of the convex v=0 equator: Torus.PointAt's
// Major+Minor·cos v term reaches the wall radius R at v=π when major=R+r), used as the concave rim's
// seam arc on-arc point (rimWithCapOrientation, fillet_rim_concave.go).
const threeQuarterTube = 2.356194490192345

// copyFace copies one face, re-aiming the cylinder wall and the cap onto the new circles and leaving
// every other face untouched.
func (g *rimBuild) copyFace(f *topo.Face) {
	specs := g.loopSpecsWithRim(f)
	if f.Reversed() {
		g.bld.AddReversedFace(f.Geometry(), f.Lineage(), specs...)
		return
	}
	g.bld.AddFace(f.Geometry(), f.Lineage(), specs...)
}

// loopSpecsWithRim rebuilds a face's loops against the new edges, substituting the rim circle (→ the
// cyl-tangent circle on the wall, the cap-tangent circle on the cap) and the wall seam (→ re-aimed).
func (g *rimBuild) loopSpecsWithRim(f *topo.Face) []topo.LoopSpec {
	specs := make([]topo.LoopSpec, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		uses := make([]topo.Use, 0, len(l.EdgeUses()))
		for _, u := range l.EdgeUses() {
			uses = append(uses, g.mapUse(f, u))
		}
		if l.IsOuter() {
			specs = append(specs, topo.OuterLoop(uses...))
		} else {
			specs = append(specs, topo.InnerLoop(uses...))
		}
	}
	return specs
}

// mapUse maps one edge use onto the rebuilt edges, swapping the rim and the wall seam.
func (g *rimBuild) mapUse(f *topo.Face, u *topo.EdgeUse) topo.Use {
	switch u.Edge() {
	case g.rf.rimEdge:
		return g.rimReplacementUse(f)
	case g.rf.seamEdge:
		return topo.Use{Edge: g.wallE, Reversed: u.Reversed()}
	}
	return topo.Use{Edge: g.edges[u.Edge()], Reversed: u.Reversed()}
}

// capRimEdgeReversed finds the ORIGINAL rim edge's use on the cap face and returns its Reversed flag —
// captured once, before the rebuild, so addBandFace can mirror it (see rimReplacementUse). rimFaces
// already guarantees rim borders cap directly, so the search always finds a hit.
func capRimEdgeReversed(cap *topo.Face, rim *topo.Edge) bool {
	for _, l := range cap.Loops() {
		for _, u := range l.EdgeUses() {
			if u.Edge() == rim {
				return u.Reversed()
			}
		}
	}
	return false
}

// rimReplacementUse returns the correctly-oriented use of the rim edge's replacement — capE on the cap
// face, cylE on the cylinder wall — for face f. capE/cylE are FRESH edges solveRim builds with their own
// fixed parametrization (Center/RefDir=ref/Radius, framed off the picked rim vertex), unrelated to the
// original STEP-imported rim curve's own direction.
//
// The two replacements behave differently, because their LOOP ROLE differs in how it can vary:
//   - cylE always replaces the rim on the cylinder wall's own OUTER loop (a wall's top rim is never a
//     hole in some larger face) — the cyl-side Reversed is a fixed function of concave/convex, mirroring
//     whatever addBandFace hard-codes for cylE, so Validate's 2-incidence rule (opposite Reversed on a
//     manifold edge's two uses) holds by construction.
//   - capE's cap-side ROLE does vary: on a lone cylinder cap (I9) the rim is the cap's OUTER boundary; on
//     a boss-root rim (R8/W6/W8/W9, #2006) the "cap" is a bigger plate and the rim bounds a HOLE loop
//     instead. A valid B-rep's hole and outer loops are, by construction, wound OPPOSITE to each other in
//     the same absolute sense, and the ORIGINAL rim edge's Reversed flag already encodes that correct
//     hole-vs-outer sense (forcing it to a role-blind constant, as cylE does, satisfies Validate but winds
//     the cap's hole the wrong way and mistessellates its area — confirmed against the corpus's expected
//     area, not just Validate). So capE keeps the cap face's original Reversed verbatim (capRimReversed),
//     and it is addBandFace's OWN capE winding that adapts to mirror it, not the other way around.
func (g *rimBuild) rimReplacementUse(f *topo.Face) topo.Use {
	if f == g.rf.cap {
		return topo.Use{Edge: g.capE, Reversed: g.capRimReversed}
	}
	return topo.Use{Edge: g.cylE, Reversed: !g.concave} // mirrors addBandFace's Reversed=concave on cylE
}

// addBandFace adds the fillet band (a torus tube on every analytic rim, a canal BSpline on the elliptic
// rim): seam up the tube, around the cap-tangent circle (opposite the
// cap), seam down, around the cyl-tangent circle (opposite the wall) — the SolidCylinderFilletedTop
// pattern, so each circle is shared with its neighbour in the opposite orientation. The CONCAVE cove
// band (S2/S5) reverses that loop: the added material is on the far side of the tube, so the band's
// outward normal — and thus its winding — flips, keeping the signed volume positive.
//
// capE's use here is NOT a fixed Fwd/Rev constant like cylE's: it is pinned to the MIRROR of
// capRimReversed, the cap face's own (role-preserving, see rimReplacementUse) use of capE, so the two
// faces sharing capE always end up antiparallel regardless of whether the cap's rim is an outer boundary
// (I9) or a hole (R8/W6/W8/W9, #2006) — the one degree of freedom the source topology actually varies.
func (g *rimBuild) addBandFace() {
	// The "torus" token is KEPT DELIBERATELY even though the band may now be a canal BSpline (the elliptic
	// rim, fillet_elliptic_rim_canal.go). It is a topological-naming token, not a description: it feeds the
	// face's reference key (ADR-0043), so renaming it to something honest like "band" would perturb every
	// rim-fillet refkey and break every rim fingerprint pin, for zero geometric gain. If a future slice
	// ever does need to rename it, that is a refkey-migration change with pin re-capture, not a cleanup.
	lin := topo.NewLineage(topo.Tok("rimfillet", "torus", 0))
	capUse := topo.Use{Edge: g.capE, Reversed: !g.capRimReversed}
	if g.concave {
		g.bld.AddFace(g.rf.band, lin,
			topo.OuterLoop(topo.Rev(g.cylE), topo.Fwd(g.seamE), capUse, topo.Rev(g.seamE)))
		return
	}
	g.bld.AddFace(g.rf.band, lin,
		topo.OuterLoop(topo.Fwd(g.seamE), capUse, topo.Rev(g.seamE), topo.Fwd(g.cylE)))
}
