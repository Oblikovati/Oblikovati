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
// in T5.3 (curvedHostArc for torus arcs, armRulingEnd for the cylinder ruling ends — the weld side
// calls it through cylinderRulingOuterOnHost so both sides land identically by construction — and one
// farCrossSectionArc reused by the arm face AND the far-runout host) — the two sides land on identical
// endpoints and weld without a crack. Any solve/closure/retrim decline returns the do-no-harm floor.

// weldCurvedArmOrFloor is filletResolvedEdges' curved-arm branch: assemble the trihedral corner into a
// watertight solid, certify it (never gate on IsSolid alone — pair it with Validate.HolesContained so a
// wrong-sign inside-out arm cannot pass), and on ANY decline return the clean do-no-harm floor error
// (never a partial body), carrying the reason so a real reject is diagnosable.
func weldCurvedArmOrFloor(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter) (*topo.Body, error) {
	b, reason := assembleCurvedArmBody(body, fils, blends, miters, ResolutionForBody(body))
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
func assembleCurvedArmBody(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter, res Resolution) (*topo.Body, string) {
	curved := curvedArmsOf(fils)
	if len(curved) == 0 {
		return nil, "no curved arm at this corner (nothing to weld)"
	}
	if ef, ok := ellipticClosedRimCanalArm(fils); ok {
		return ellipticClosedRimCanalBody(body, ef) // J6/J8: one CLOSED elliptic rim → non-analytic canal band
	}
	if m := curvedMiterOf(fils, miters); m != nil {
		return curvedMiterBody(body, m, res) // families B/C: 2-arm curved miter (torus + cylinder) mutual trim
	}
	if isSingleArmRunout(fils) {
		return singleArmRunoutBody(body, fils[0], res) // one curved arm, two plane-capped ends: corner-free both-ends weld
	}
	if isConvexClosedRimArm(fils) {
		return convexClosedRimBandBody(body, fils[0], res) // one convex CLOSED cone/cyl rim → full torus band (J1)
	}
	if isConcaveClosedRimArm(fils) {
		return concaveClosedRimBandBody(body, fils[0], res) // one concave CLOSED sphere/cone cap rim → cove band (S2/S5)
	}
	vid, ok := sharedCornerVertex(curved)
	if !ok {
		return nil, "curved arms do not meet at one shared trihedral vertex"
	}
	return trihedralCornerBody(body, fils, blends, vid, res)
}

// trihedralCornerBody welds the 3-arm trihedral corner at shared vertex vid — the single-ball path (with
// its canal sibling) that predates the single-arm runout dispatch. Split out of assembleCurvedArmBody so
// the runout dispatch fits without pushing the router over funlen; the body below is byte-identical.
func trihedralCornerBody(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend, vid uint64, res Resolution) (*topo.Body, string) {
	arms := cornerArms(fils, vid) // ALL fillets at V — 2 curved arms + the planar Plane∧Plane cyl arm
	if len(arms) < 3 {
		return nil, fmt.Sprintf("trihedral corner needs 3 arms (got %d at vertex %d)", len(arms), vid)
	}
	if reason := declineUnconsumedPicks(fils, arms, vid); reason != "" {
		return nil, reason // a pick outside the welded corner would be left unrounded — decline, don't ship a partial
	}
	if b, reason, took := curvedMixedCornerBody(body, arms, res); took {
		return b, reason // M8-class mixed-sense curved-host 2r-torus corner (the sphere single-ball path below is UNTOUCHED)
	}
	if b, reason, took := cornerWeldLayerBody(body, arms, res); took {
		return b, reason // the general corner-weld layer (N4 class today); the sphere path below is UNTOUCHED
	}
	if b, reason, took := canalArmBody(body, arms, blends, vid, res); took {
		return b, reason // ADR-C4-1: tangent-degenerate valence-4 corner → sibling canal weld (single-ball path below is UNTOUCHED)
	}
	w, sphere, reason := solveCurvedArmCorner(arms, blends, vid, res)
	if reason != "" {
		return nil, reason
	}
	faces, reason := curvedWeldFaces(body, arms, w, sphere, res)
	if reason != "" {
		return nil, reason
	}
	return assembleCornerBlendBody(body, faces), ""
}

// assembleCornerBlendBody welds the corner faces, then corrects the one thing orientFilletShell cannot
// pin: the corner blend sphere patch's ABSOLUTE winding, which the sphere-patch mesher reads to pick which
// of the two regions it bounds. orientFilletShell fixes only RELATIVE windings and pins the shell's global
// sense to an arbitrary arm seed, so a cone-host corner blend can land inverted and mesh the COMPLEMENT
// (D1: 1016.7 vs the exact 238.5). When the original body has no host sphere and the built patch meshes >
// a hemisphere, the whole shell is UNIFORMLY reversed and re-welded — a uniform flip is invisible to every
// mesher except the sphere patch (whose region it inverts). It fires ONLY on the genuinely-inverted case
// (never B3 or the 60 greens, whose sub-hemisphere caps mesh correctly), so byte-identity holds.
func assembleCornerBlendBody(originalBody *topo.Body, faces []filletFace) *topo.Body {
	built := assembleBody(faces)
	if !cornerBlendMeshesComplement(originalBody, built) {
		return built
	}
	flipped := make([]filletFace, len(faces))
	for i, f := range faces {
		flipped[i] = reverseFilletFace(f)
	}
	return assembleBody(flipped)
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

// declineUnconsumedPicks returns a non-empty do-no-harm reason when the welded trihedral corner does not
// consume EVERY pick. Slice A welds exactly the arms meeting the shared vertex vid (arms ⊆ fils, gathered
// by cornerArms), so any pick not at that vertex — an extra curved arm at a different vertex, or an
// unrelated planar edge — is never rounded, yet the corner weld alone can still close into a body that
// passes Validate and IsSolid and would be RETURNED with that pick left unrounded (a partial/wrong solid).
// Declining here routes the whole op to the clean floor (curvedArmUnweldedError) instead. Empty means
// every pick is consumed by the corner. (M5 Slice A whole-branch review, I-2.)
func declineUnconsumedPicks(fils, arms []edgeFillet, vid uint64) string {
	if len(arms) == len(fils) {
		return ""
	}
	return fmt.Sprintf("curved weld consumes only the %d arms at trihedral vertex %d, but %d edges were picked: %d unconsumed pick(s) would be left unrounded (Slice A welds one trihedral corner)",
		len(arms), vid, len(fils), len(fils)-len(arms))
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
	picks := filletedEdgeSet(arms)
	for i := range arms {
		b, reason := armRailBundle(w.arms[i], arms[i], w, picks, res)
		if reason != "" {
			return nil, fmt.Sprintf("arm rail bundle declined (%T): %s", arms[i].armSurface, reason)
		}
		bundles[i] = b
	}
	faces := make([]filletFace, 0, len(body.Faces())+len(arms)+1)
	for i := range bundles {
		faces = append(faces, curvedArmTrimmedFace(bundles[i], arms[i]))
	}
	sf, ok := curvedCornerFace(w, sphere, arms, res)
	if !ok {
		return nil, "corner sphere spherical-triangle face declined"
	}
	hostFaces, reason := curvedHostFaces(body, arms, bundles, w, res)
	if reason != "" {
		return nil, reason
	}
	all := append(append(faces, sf), hostFaces...)
	return orientForSphereHost(body, all, hostFaces), "" // orientForSphereHost lives in fillet_curved_sphere_orient.go
}

// armRails is one arm's four boundary edges as a closed loop (host rail on ef.a, setback great-arc,
// host rail on ef.b reversed, far runout trim), plus that far trim alone (outer0→outer1) — the rail
// shared with the far-runout host, kept separate so both faces reuse the identical curve — and the
// runout object (regime + capping identity) the far-runout host bite router reads (FR3).
//
// hostA/hostB are the SAME (possibly oblique-re-terminated) host contact rails, un-reversed and oriented
// outer→tHost — hostA on ef.a, hostB on ef.b. They are kept as their own fields (not re-extracted from
// segs) so the CORNER-HOST retrim (fillet_curved_retrim.go, FR4) consumes the identical curve OBJECT the
// arm face carries, welding the two sides by construction: for a perpendicular runout hostA/hostB are the
// untouched full curvedHostArc arcs (existing greens byte-identical), for an oblique runout their outer
// ends are the closed-form feet ON the host loop (D5/E4). Re-extracting from segs would double-reverse
// segs[2] and re-derive its arc — a congruent but NOT bit-identical curve, breaking the weld identity.
type armRails struct {
	segs   []endSeg
	far    endSeg
	runout armRunout
	hostA  endSeg // ef.a host contact rail, oriented outer→tHost (== segs[0])
	hostB  endSeg // ef.b host contact rail, oriented outer→tHost (segs[2] is this, reversed)
	// armSurface overrides ef.armSurface for the trimmed arm FACE when non-nil — the cone-ruling canal arm
	// (CN4b-2) re-lofts over the CORNER span [x_f,far, x_f,C] (NOT the edge span), so the setback boundary is
	// the exact v-edge of the surface rather than a curve interior to it (which meshes the sliver past the
	// corner, D1 +119). nil for every torus/cylinder arm (its analytic surface already stops at the corner).
	armSurface geom.Surface
}

// armRailBundle assembles one arm's four boundary rails (§B.5). t0/t1 are the arm's two host-tangent
// points (the setback rail endpoints); the far terminus runs THROUGH the general far-runout engine
// (armFarRunout, FR3) — perpendicular caps take the existing farCrossSectionArc verbatim (byte-identity
// by call-graph, ADR-2) and pass the host rails through untouched; an oblique cap builds the analytic
// section trim (intersectArmCapping) and RE-TERMINATES the two host rails on the feet (ADR-4). Returns a
// non-empty reason (the exact obstruction) on any host-rail / setback / far-runout decline — do-no-harm.
func armRailBundle(set armSetback, ef edgeFillet, w cornerWeld, filletedEdges map[uint64]bool, res Resolution) (armRails, string) {
	if set.canalSpine != nil {
		return canalArmRailBundle(set, ef, w, filletedEdges, res) // CN4b-2: cone-ruling canal arm (spring rails + oblique/snout cap)
	}
	t0 := endpointOf(w.center, w.radius, set.railDir0) // host-tangent point on ef.a
	t1 := endpointOf(w.center, w.radius, set.railDir1) // host-tangent point on ef.b
	h0, ok0 := armHostContactRail(ef.a, set, t0, w, res)
	h1, ok1 := armHostContactRail(ef.b, set, t1, w, res)
	setback, ok2 := setbackEndSeg(w, set, t0, t1)
	if !ok0 || !ok1 || !ok2 {
		return armRails{}, "host contact rail or setback rail could not be built"
	}
	h0, h1, run, ok3, reason := armFarRunout(ef, w, h0, h1, filletedEdges, res)
	if !ok3 {
		return armRails{}, reason
	}
	return closeArmRails(h0, h1, setback, run), ""
}

// closeArmRails assembles one arm's four-rail boundary loop from its (possibly re-terminated) host rails,
// the setback great-arc, and the far runout trim reversed (outer1→outer0, closing to h0.from).
// reverseEndSegs reverses ANY curve — an Arc3d by its three points (a perpendicular cross-section arc,
// byte-identical to the pre-FR3 loop) or an analytic section curve via ReverseCurve3 (an oblique spiric/
// ellipse trim) — so both regimes close the loop the same way.
func closeArmRails(h0, h1, setback endSeg, run armRunout) armRails {
	segs := []endSeg{h0, setback}
	segs = append(segs, reverseEndSegs([]endSeg{h1})...)       // t1 → outer1
	segs = append(segs, reverseEndSegs([]endSeg{run.trim})...) // outer1 → outer0 (closes to h0.from)
	return armRails{segs: segs, far: run.trim, runout: run, hostA: h0, hostB: h1}
}

// filletedEdgeSet is the set of picked (filleted) edge ids at the welded corner — the arm edges gathered
// by cornerArms. The far-runout admission gate reads it to decline when a SECOND picked edge ends at an
// arm's far vertex (fillet-fillet interference, out of scope; architecture Q5).
func filletedEdgeSet(arms []edgeFillet) map[uint64]bool {
	set := make(map[uint64]bool, len(arms))
	for _, ef := range arms {
		set[ef.edge.ID()] = true
	}
	return set
}

// armHostContactRail builds one arm's contact rail on host, oriented outer→tHost: a torus arm carves a
// circular arc (curvedHostArc — the SAME constructor T5.3's retrim uses, so the two sides weld), a
// cylinder arm a straight ruling whose outer end is armRulingEnd (the SAME landing the retrim uses).
// Declines when neither applies.
func armHostContactRail(host *topo.Face, set armSetback, tHost math.Point3, w cornerWeld, res Resolution) (endSeg, bool) {
	tol := res.Weld() * w.radius
	switch s := set.arm.(type) {
	case geom.Torus:
		arc, ok := curvedHostArc(host.Geometry(), s, w, res)
		if !ok || float64(arc.PointAt(1).DistanceTo(tHost)) > tol {
			return endSeg{}, false // no torus rail on this host, or it misses the tangent point
		}
		return endSeg{from: arc.PointAt(0), to: tHost, curve: arc, mid: arc.PointAt(0.5), arc: true}, true
	case geom.Cylinder:
		outer, ok := cylinderRulingOuterOnHost(host, s, set, tHost, w, tol)
		if !ok {
			return endSeg{}, false
		}
		return endSeg{from: outer, to: tHost}, true
	}
	return endSeg{}, false
}

// cylinderRulingOuterOnHost is the SINGLE source of truth for a cylinder arm ruling's outer end on a
// host: it calls the SAME armRulingEnd (N2) the host retrim uses, with the host's OWN original loop and
// bitten vertex AND the arm's setback (its far-vertex authority), so the arm face and the retrimmed host
// land on a byte-identical point and weld without a crack. This replaced the former closed-form
// cylinderRulingOuter, which slid tHost to the arm FAR VERTEX's axial station: that equalled the host
// loop's extreme only when the far vertex happened to sit on the host rim (true for B3, a coincidence —
// NOT the identity the old docstring claimed). Routing both sides through one helper makes the coupling
// correct by construction, or declines to the floor.
func cylinderRulingOuterOnHost(host *topo.Face, arm geom.Cylinder, set armSetback, tHost math.Point3, w cornerWeld, tol float64) (math.Point3, bool) {
	segs := originalHostSegs(host)
	if len(segs) == 0 {
		return math.Point3{}, false // a host with no readable outer loop cannot anchor a ruling
	}
	v := bittenVertex(segs, w.center)
	return armRulingEnd(host, arm, set, tHost, v, segs, tol)
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
	surface := ef.armSurface
	if rails.armSurface != nil {
		surface = rails.armSurface // CN4b-2: the canal arm re-lofted over the corner span [x_f,far, x_f,C]
	}
	return filletFace{
		surface: surface,
		loops:   []filletLoop{loopFromSegs(rails.segs)},
		parent:  filletEdgeProvenance(ef.edge),
	}
}

// curvedCornerFace emits the corner patch: the SURFACE is validated through the RailLoop engine
// (extractCurvedCorner→resolveBlend — analyticSphere wins the octant, coons4 the degenerate 4-sided
// fill), while the octant's boundary LOOP stays the legacy chainSetbackArcs so B3 is byte-for-byte
// (ADR-2 Step 1 strangler; sphere loop-collapse is a gated follow-up). Falls back to curvedSphereFace
// wholesale on any engine decline (do-no-harm). The Kind-gate is load-bearing: only BlendKindSphere
// (via the legacy loop), BlendKindCoons4, and the M6' BlendKindCanal are admitted; any other tier
// (e.g. tri3) is NOT a valid curved corner and falls back rather than shipping a wrong 3-sided corner.
func curvedCornerFace(w cornerWeld, sphere geom.Sphere, arms []edgeFillet, res Resolution) (filletFace, bool) {
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok {
		return curvedSphereFace(w, sphere) // extractor declined — legacy octant path
	}
	patch, ok := resolveBlend(loop, res)
	if !ok {
		return curvedSphereFace(w, sphere)
	}
	switch patch.Kind {
	case BlendKindSphere:
		return curvedSphereFace(w, sphere) // octant: engine-validated surface == sphere, KEEP legacy loop (ADR-2 Step 1)
	case BlendKindCoons4:
		return patchToFilletFace(patch, topo.Lineage{}), true // degenerate 4-sided fill: take the engine's loops
	case BlendKindCanal:
		return patchToFilletFace(patch, topo.Lineage{}), true // M6' rolling-ball canal: same handoff as coons4 (surface + received-rail loops)
	default:
		return curvedSphereFace(w, sphere) // any other tier (e.g. tri3) is NOT a valid curved corner — do-no-harm
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
