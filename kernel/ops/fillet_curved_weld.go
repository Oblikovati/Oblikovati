// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// M5 Slice A, Task 5.4 (m5-weld-setback-retrim-derivation.md §B.5): the integration crux. It assembles
// the nine result faces of the axis-aligned curved-arm trihedral fillet — three trimmed analytic arm
// faces (torus/cylinder), the corner sphere's spherical triangle, and the retrimmed host faces — into
// ONE watertight solid, and routes the family through it (greening OCCT blend/simple/B3, corpus 54→55).
//
// It consumes the committed slices: T5.1 solveCurvedCorner (the corner + setback stations), T5.2
// curvedSetbackRail (the arm↔sphere great-circle weld rails), and T5.3 retrimCurvedHost (the circular
// host-retrim). The one guarantee that makes the weld watertight: assembleBody welds faces by shared
// loop POINTS, so every rail an arm and its neighbour share is built by the SAME constructor here and
// in T5.3 (curvedHostArc for torus arcs, cylinderRulingOuter matching T5.3's ruling ends, and one
// farCrossSectionArc reused by the arm face AND the far-runout host) — the two sides land on identical
// endpoints and weld without a crack. Any solve/closure/retrim decline returns the do-no-harm floor.

// weldCurvedArmOrFloor is filletResolvedEdges' curved-arm branch: assemble the trihedral corner into a
// watertight solid, certify it (never gate on IsSolid alone — pair it with Validate.HolesContained so a
// wrong-sign inside-out arm cannot pass), and on ANY decline return the clean do-no-harm floor error
// (never a partial body), carrying the reason so a real reject is diagnosable.
func weldCurvedArmOrFloor(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend) (*topo.Body, error) {
	b, reason := assembleCurvedArmBody(body, fils, blends, ResolutionForBody(body))
	if reason == "" {
		if rep := Validate(b); rep.Valid && rep.HolesContained && b.IsSolid() {
			return b, nil
		}
		reason = "assembled weld did not certify as a valid solid"
	}
	return nil, curvedArmUnweldedError(fils, reason)
}

// assembleCurvedArmBody builds the nine result faces (3 trimmed arms + sphere spherical triangle +
// retrimmed hosts) and welds them via assembleBody. An EMPTY reason means the returned body is the
// watertight weld; a non-empty reason names WHY the weld declined (station gap / host non-tangency /
// Gauss–Bonnet closure / host-retrim decline) and the body is nil — the caller keeps the clean
// do-no-harm floor (never a partial body), threading the reason into curvedArmUnweldedError (the
// T5.1-review decline-reason requirement). Example:
//
//	if b, reason := assembleCurvedArmBody(body, fils, blends, res); reason == "" { /* watertight solid */ }
func assembleCurvedArmBody(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend, res Resolution) (*topo.Body, string) {
	curved := curvedArmsOf(fils)
	if len(curved) == 0 {
		return nil, "no curved arm at this corner (nothing to weld)"
	}
	vid, ok := sharedCornerVertex(curved)
	if !ok {
		return nil, "curved arms do not meet at one shared trihedral vertex"
	}
	arms := cornerArms(fils, vid) // ALL fillets at V — 2 curved arms + the planar Plane∧Plane cyl arm
	if len(arms) < 3 {
		return nil, fmt.Sprintf("trihedral corner needs 3 arms (got %d at vertex %d)", len(arms), vid)
	}
	w, sphere, reason := solveCurvedArmCorner(arms, blends, vid, res)
	if reason != "" {
		return nil, reason
	}
	faces, reason := curvedWeldFaces(body, arms, w, sphere, res)
	if reason != "" {
		return nil, reason
	}
	return assembleBody(faces), ""
}

// curvedArmsOf returns the fils carrying an exact analytic arm surface (the curved Plane∧Cylinder
// edges) — the trigger + shared-vertex anchor. The corner's third arm (a planar Plane∧Plane edge whose
// fillet is an ordinary cylinder) is gathered later by cornerArms, keyed on the shared vertex.
func curvedArmsOf(fils []edgeFillet) []edgeFillet {
	arms := make([]edgeFillet, 0, len(fils))
	for _, ef := range fils {
		if ef.armSurface != nil {
			arms = append(arms, ef)
		}
	}
	return arms
}

// cornerArms gathers every fillet meeting the trihedral vertex vid and normalizes each to carry its arm
// surface in armSurface — the exact torus/cylinder for a curved Plane∧Cylinder edge, or the rolling-ball
// cylinder (ef.cyl) for a planar Plane∧Plane edge (the derivation's "planar cyl arm", §Problem framing).
func cornerArms(fils []edgeFillet, vid uint64) []edgeFillet {
	out := make([]edgeFillet, 0, 3)
	for _, ef := range fils {
		if ef.edge.StartVertex().ID() == vid || ef.edge.EndVertex().ID() == vid {
			out = append(out, armWithSurface(ef, vid))
		}
	}
	return out
}

// armWithSurface normalizes one corner arm's surface: a planar Plane∧Plane fillet takes its rolling-ball
// cylinder (ef.cyl) as its arm; a torus arm is re-referenced so its angle-zero points at the FAR cut (the
// edge end away from the corner). The torus inherits the host cylinder's arbitrary Ref on import
// (torusArmSurface), but T5.3's curvedHostArc sweeps the assembled span [az0 → station] FROM az0 — so az0
// MUST be the far cut for the host arcs to land on the loop. This re-Ref establishes that invariant
// without touching T5.3 (a torus surface is identical under a Ref rotation — only its u=0 line moves).
func armWithSurface(ef edgeFillet, vid uint64) edgeFillet {
	if ef.armSurface == nil {
		ef.armSurface = ef.cyl // Plane∧Plane fillet: the arm is its constant rolling-ball cylinder
		return ef
	}
	if tor, ok := ef.armSurface.(geom.Torus); ok {
		ef.armSurface = alignTorusRefToFar(tor, farVertexNotVid(ef.edge, vid))
	}
	return ef
}

// farVertexNotVid returns the arm edge's endpoint that is NOT the corner vertex vid — the far cut.
func farVertexNotVid(e *topo.Edge, vid uint64) math.Point3 {
	if e.StartVertex().ID() == vid {
		return e.EndVertex().Point()
	}
	return e.StartVertex().Point()
}

// alignTorusRefToFar rebuilds the torus with its angle-zero reference pointing at far (in the torus
// plane), so the assembled arm spans azimuth [0 → station] from the far cut to the setback. Leaves the
// torus unchanged when far projects onto the axis (degenerate) or the rebuild declines.
func alignTorusRefToFar(tor geom.Torus, far math.Point3) geom.Torus {
	axis := tor.AxisDir.AsVector()
	d := tor.Center.VectorTo(far)
	ref, err := math.UnitVector3FromVector(d.Sub(axis.Scale(d.Dot(axis))))
	if err != nil {
		return tor
	}
	t2, err := geom.NewTorusWithRef(tor.Center, axis, ref.AsVector(), tor.MajorRadius, tor.MinorRadius)
	if err != nil {
		return tor
	}
	return t2
}

// solveCurvedArmCorner reads the corner sphere from blends[vid] and solves the setback corner (T5.1). A
// missing/planar corner blend or a solve decline (station gap / host non-tangency / Gauss–Bonnet
// closure) each returns a diagnostic reason.
func solveCurvedArmCorner(arms []edgeFillet, blends map[uint64]*cornerBlend, vid uint64, res Resolution) (cornerWeld, geom.Sphere, string) {
	cb, ok := blends[vid]
	if !ok {
		return cornerWeld{}, geom.Sphere{}, fmt.Sprintf("no corner sphere solved at the trihedral vertex %d", vid)
	}
	w, ok := solveCurvedCorner(cb.sphere, arms, res)
	if !ok {
		return cornerWeld{}, geom.Sphere{}, "corner solve declined (station gap / host non-tangency / closure failure)"
	}
	return w, cb.sphere, ""
}

// sharedCornerVertex returns the vertex id shared by every arm's edge — the trihedral corner V. A
// well-formed trihedral corner has exactly one such vertex; none (arms fanning to different vertices)
// is declined by the caller.
func sharedCornerVertex(arms []edgeFillet) (uint64, bool) {
	count := map[uint64]int{}
	for _, ef := range arms {
		count[ef.edge.StartVertex().ID()]++
		count[ef.edge.EndVertex().ID()]++
	}
	for id, c := range count {
		if c == len(arms) {
			return id, true
		}
	}
	return 0, false
}

// curvedWeldFaces builds every result face: one trimmed analytic face per arm, the corner sphere's
// spherical triangle, and the retrimmed/far-runout host faces. It computes each arm's shared rail
// bundle ONCE so the arm face and its neighbour host land on byte-identical rails (watertight weld).
func curvedWeldFaces(body *topo.Body, arms []edgeFillet, w cornerWeld, sphere geom.Sphere, res Resolution) ([]filletFace, string) {
	bundles := make([]armRails, len(arms))
	for i := range arms {
		b, ok := armRailBundle(w.arms[i], arms[i], w, res)
		if !ok {
			return nil, fmt.Sprintf("arm rail bundle declined (%T)", arms[i].armSurface)
		}
		bundles[i] = b
	}
	faces := make([]filletFace, 0, len(body.Faces())+len(arms)+1)
	for i := range bundles {
		faces = append(faces, curvedArmTrimmedFace(bundles[i], arms[i]))
	}
	sf, ok := curvedSphereFace(w, sphere)
	if !ok {
		return nil, "corner sphere spherical-triangle face declined"
	}
	hostFaces, reason := curvedHostFaces(body, arms, bundles, w, res)
	if reason != "" {
		return nil, reason
	}
	return append(append(faces, sf), hostFaces...), ""
}

// armRails is one arm's four boundary edges as a closed loop (host rail on ef.a, setback great-arc,
// host rail on ef.b reversed, far cross-section arc), plus that far arc alone (outer0→outer1) — the
// rail shared with the far-runout host, kept separate so both faces reuse the identical curve.
type armRails struct {
	segs []endSeg
	far  endSeg
}

// armRailBundle assembles one arm's four boundary rails (§B.5). t0/t1 are the arm's two host-tangent
// points (the setback rail endpoints); the far cross-section arc joins the two host rails' outer ends
// on the arm's terminal radius-r circle. Declines when a host rail or the setback rail cannot be built.
func armRailBundle(set armSetback, ef edgeFillet, w cornerWeld, res Resolution) (armRails, bool) {
	t0 := endpointOf(w.center, w.radius, set.railDir0) // host-tangent point on ef.a
	t1 := endpointOf(w.center, w.radius, set.railDir1) // host-tangent point on ef.b
	far := armFarVertex(ef, w.center)
	h0, ok0 := armHostContactRail(ef.a, set, t0, far, w, res)
	h1, ok1 := armHostContactRail(ef.b, set, t1, far, w, res)
	setback, ok2 := setbackEndSeg(w, set, t0, t1)
	farArc, ok3 := farCrossSectionArc(set.arm, w.radius, h0.from, h1.from)
	if !ok0 || !ok1 || !ok2 || !ok3 {
		return armRails{}, false
	}
	segs := []endSeg{h0, setback}
	segs = append(segs, reverseEndSegs([]endSeg{h1})...)     // t1 → outer1
	segs = append(segs, reverseEndSegs([]endSeg{farArc})...) // outer1 → outer0 (closes to h0.from)
	return armRails{segs: segs, far: farArc}, true
}

// armFarVertex is the arm edge's far end — the endpoint AWAY from the trihedral corner (the corner end
// sits within ~r√2 of the ball centre C, the far end is the arm's runout station). It anchors a
// cylinder arm's straight ruling and its far cross-section circle.
func armFarVertex(ef edgeFillet, c math.Point3) math.Point3 {
	s, e := ef.edge.StartVertex().Point(), ef.edge.EndVertex().Point()
	if s.DistanceTo(c) >= e.DistanceTo(c) {
		return s
	}
	return e
}

// armHostContactRail builds one arm's contact rail on host, oriented outer→tHost: a torus arm carves a
// circular arc (curvedHostArc — the SAME constructor T5.3's retrim uses, so the two sides weld), a
// cylinder arm a straight ruling from the far runout to tHost. Declines when neither applies.
func armHostContactRail(host *topo.Face, set armSetback, tHost, far math.Point3, w cornerWeld, res Resolution) (endSeg, bool) {
	tol := res.Weld() * w.radius
	switch s := set.arm.(type) {
	case geom.Torus:
		arc, ok := curvedHostArc(host.Geometry(), s, w, res)
		if !ok || float64(arc.PointAt(1).DistanceTo(tHost)) > tol {
			return endSeg{}, false // no torus rail on this host, or it misses the tangent point
		}
		return endSeg{from: arc.PointAt(0), to: tHost, curve: arc, mid: arc.PointAt(0.5), arc: true}, true
	case geom.Cylinder:
		return endSeg{from: cylinderRulingOuter(s, tHost, far), to: tHost}, true
	}
	return endSeg{}, false
}

// cylinderRulingOuter is the far end of a cylinder arm's straight ruling on a host: tHost slid along the
// arm axis to the far runout's axial station. This reproduces T5.3's rulingOuterEnd landing (wall axial
// extreme / planar loop exit) in closed form, so the arm face and the retrimmed host weld on it.
func cylinderRulingOuter(cyl geom.Cylinder, tHost, far math.Point3) math.Point3 {
	axis := cyl.AxisDir.AsVector()
	return tHost.TranslateBy(axis.Scale(tHost.VectorTo(far).Dot(axis)))
}

// setbackEndSeg wraps the arm↔sphere great-circle weld rail (T5.2) as an endSeg oriented t0→t1.
func setbackEndSeg(w cornerWeld, set armSetback, t0, t1 math.Point3) (endSeg, bool) {
	rail, ok := curvedSetbackRail(w, set)
	if !ok {
		return endSeg{}, false
	}
	return endSeg{from: t0, to: t1, curve: rail, mid: rail.PointAt(0.5), arc: true}, true
}

// farCrossSectionArc is the arm's terminal cross-section arc (radius r about the spine point at the far
// runout) joining the two host rails' outer ends — the torus tube quarter at the y=0 cut, or the
// through-cylinder foot arc on the cap it exits (§B.5). Shared verbatim with the far-runout host face.
func farCrossSectionArc(arm geom.Surface, r float64, outer0, outer1 math.Point3) (endSeg, bool) {
	spine, ok := armBallCenter(arm, outer0)
	if !ok {
		return endSeg{}, false
	}
	mid := arcMidBetween(spine, r, outer0, outer1)
	arc, err := geom.Arc3dByThreePoints(outer0, mid, outer1)
	if err != nil {
		return endSeg{}, false
	}
	return endSeg{from: outer0, to: outer1, curve: arc, mid: mid, arc: true}, true
}

// curvedArmTrimmedFace emits one trimmed arm face: the analytic arm surface bounded by its four rails
// (two host contact rails + the setback great-arc + the far cross-section arc). Its parent lineage is
// the generating filleted edge (ADR-0043, via filletEdgeProvenance — the same helper the planar cyl
// faces use) so the blend's edges/faces inherit a stable topological name that survives an upstream
// edit, rather than a build-order name that renumbers.
func curvedArmTrimmedFace(rails armRails, ef edgeFillet) filletFace {
	return filletFace{
		surface: ef.armSurface,
		loops:   []filletLoop{loopFromSegs(rails.segs)},
		parent:  filletEdgeProvenance(ef.edge),
	}
}

// curvedSphereFace builds the corner sphere's spherical-triangle patch: the surface is the solved corner
// sphere, the loop is the three setback great-arcs (each shared with an arm) chained into a closed ring.
// It carries NO parent lineage by design: the corner sphere is generated by the trihedral VERTEX (where
// the three filleted edges meet), not by any single edge, so no filletEdgeProvenance edge-name applies —
// this mirrors the planar corner patch (spherePatchFace), which likewise emits no single-edge parent.
func curvedSphereFace(w cornerWeld, sphere geom.Sphere) (filletFace, bool) {
	segs, ok := chainSetbackArcs(w)
	if !ok {
		return filletFace{}, false
	}
	return filletFace{surface: sphere, loops: []filletLoop{loopFromSegs(segs)}}, true
}

// chainSetbackArcs collects each arm's setback great-arc and chains them head-to-tail into the closed
// spherical triangle (the three rails share the host-tangent points pairwise, so they close into a ring).
func chainSetbackArcs(w cornerWeld) ([]endSeg, bool) {
	arcs := make([]endSeg, 0, len(w.arms))
	for _, a := range w.arms {
		rail, ok := curvedSetbackRail(w, a)
		if !ok {
			return nil, false
		}
		arcs = append(arcs, endSeg{from: rail.PointAt(0), to: rail.PointAt(1), curve: rail, mid: rail.PointAt(0.5), arc: true})
	}
	return chainEndSegs(arcs, railGreatCircleTol*w.radius)
}

// chainEndSegs orders a set of endSegs head-to-tail into a closed ring, reversing a seg when it is met
// from its `to` end. Returns false if the chain cannot close (a rail endpoint has no continuation).
func chainEndSegs(segs []endSeg, tol float64) ([]endSeg, bool) {
	used := make([]bool, len(segs))
	out := make([]endSeg, 0, len(segs))
	cur := segs[0].from
	for range segs {
		next, ok := takeEndSegFrom(segs, used, cur, tol)
		if !ok {
			return nil, false
		}
		out = append(out, next)
		cur = next.to
	}
	if float64(cur.DistanceTo(out[0].from)) > tol {
		return nil, false // chain did not return to its start — not a closed loop
	}
	return out, true
}

// takeEndSegFrom returns the first unused seg touching cur, oriented to start at cur (reversing it when
// cur is its `to` end), and marks it used.
func takeEndSegFrom(segs []endSeg, used []bool, cur math.Point3, tol float64) (endSeg, bool) {
	for i := range segs {
		if used[i] {
			continue
		}
		if float64(segs[i].from.DistanceTo(cur)) <= tol {
			used[i] = true
			return segs[i], true
		}
		if float64(segs[i].to.DistanceTo(cur)) <= tol {
			used[i] = true
			return reverseEndSegs([]endSeg{segs[i]})[0], true
		}
	}
	return endSeg{}, false
}
