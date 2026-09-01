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
	if b, reason, took := independentClosedRimsBody(body, fils); took {
		return b, reason // B3: ≥2 disjoint CLOSED Cylinder∧Plane rims → sequential single-rim bands (no corner exists)
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
	return assembleCornerBlendBody(faces), ""
}

// assembleCornerBlendBody welds the corner faces. It used to follow up with a whole-shell uniform
// flip when the assembled corner-blend sphere patch meshed the COMPLEMENT of its region — a repair
// for orientFilletShell pinning the shell's absolute sense to an arbitrary arm seed. That fixup is
// GONE (M48/C3, Oblikovati/Oblikovati#3432): the sphere-patch mesher now asks the face which region
// it owns (interiorAxis → brep.PointInFaceTrim), and measured on the corner bodies the fixup was
// built for — D1 10078.811 against OCCT 10078.800, with C2/C6/B3/C8 equally on target — it no longer
// fires anywhere in the fillet and parity corpora. Deleting it also removes the last modelling
// decision in this path that read a tessellation.
func assembleCornerBlendBody(faces []filletFace) *topo.Body {
	return assembleBody(faces)
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
