// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Stage 3+4a of the shared corner-weld executor: turn one cornerArmSpec into its contact rails, its far
// termination, and the arm FACES it emits. It CALLS the existing primitives (assignArcFeetToHosts /
// assignFeetToHosts, mixedArmHostRails, armFarRunout, armRunoutRail, farCrossSectionArc) and never
// reimplements them — this stage is what buildMixedArmBundle + mixedArmFace do in the bespoke mixed-corner
// weld, with the per-case orchestration lifted out and every shared boundary handed to the ledger.
//
// An arm is a chain of LINKS. A plain arm has exactly one — the picked edge, its two hosts, its far vertex —
// and everything below reduces to the single-arm case element for element. A rim-CONTINUED arm
// (farRimContinuation) has one link per host-face span it crosses, and each interior boundary is a radius-r
// cross-section STATION shared by the two arm faces meeting there. That station is what splits the band into
// the several faces OCCT emits (design Axis C4): the split is induced by the HOST FACE seam, so it needs no
// separate mechanism — it falls out of the link chain.

// cornerArmLink is one span of an arm: the loop edge it consumes, the two host faces it rides on there, and
// the vertex closing its FAR end.
type cornerArmLink struct {
	edge   *topo.Edge
	hostA  *topo.Face
	hostB  *topo.Face
	farVtx *topo.Vertex
}

// cornerArmStation is one cross-section boundary of the arm: its two contact feet (on the adjoining link's
// hostA / hostB) and the curve joining them. xs is noRail at the NEAR station, whose boundary is the corner
// patch's own rail instead.
type cornerArmStation struct {
	a, b math.Point3
	xs   railID
}

// cornerArmWeldOf runs stage 3 for one arm: near feet → links → stations → rails + far runout → faces+bites.
func cornerArmWeldOf(plan cornerWeldPlan, spec cornerArmSpec, res opstol.Resolution) (cornerArmWeld, string) {
	tol := res.Weld() * plan.radius
	near, reason := cornerNearStation(plan, spec, tol)
	if reason != "" {
		return cornerArmWeld{}, reason
	}
	links, reason := cornerArmLinks(plan, spec, res)
	if reason != "" {
		return cornerArmWeld{}, reason
	}
	stations, reason := cornerArmStations(plan, spec, links, near, tol)
	if reason != "" {
		return cornerArmWeld{}, reason
	}
	rails, run, reason := cornerArmRails(plan, spec, links, stations, res)
	if reason != "" {
		return cornerArmWeld{}, reason
	}
	return cornerArmWeldFrom(plan, spec, links, stations, rails, run)
}

// cornerNearStation resolves the near boundary chain's two outer endpoints and assigns them to the arm's two
// hosts. A1 keeps the byte-identical assignArcFeetToHosts call the built welds make; A2 uses its
// general-curve sibling assignFeetToHosts.
func cornerNearStation(plan cornerWeldPlan, spec cornerArmSpec, tol float64) (cornerArmStation, string) {
	if spec.nearKind == armTerminatesAtArc {
		a, b, ok := assignArcFeetToHosts(spec.ef, spec.nearArc, tol)
		if !ok {
			return cornerArmStation{}, fmt.Sprintf("corner-arc endpoints do not land on the two hosts %T/%T",
				spec.ef.a.Geometry(), spec.ef.b.Geometry())
		}
		return cornerArmStation{a: a, b: b, xs: noRail}, ""
	}
	p0, p1, ok := plan.ledger.ends(spec.near)
	if !ok {
		return cornerArmStation{}, "near boundary chain is empty (no rail registered)"
	}
	a, b, ok := assignFeetToHosts(spec.ef, p0, p1, tol)
	if !ok {
		return cornerArmStation{}, fmt.Sprintf("near rail ends do not land on the two hosts %T/%T",
			spec.ef.a.Geometry(), spec.ef.b.Geometry())
	}
	return cornerArmStation{a: a, b: b, xs: noRail}, ""
}

// cornerArmLinks builds the arm's link chain: one link for a plain arm, or the tangent-continuous rim chain
// for a farRimContinuation arm.
func cornerArmLinks(plan cornerWeldPlan, spec cornerArmSpec, res opstol.Resolution) ([]cornerArmLink, string) {
	first := cornerArmLink{
		edge: spec.ef.edge, hostA: spec.ef.a, hostB: spec.ef.b,
		farVtx: farEndVertex(spec.ef.edge, plan.vertex),
	}
	if spec.far != farRimContinuation {
		return []cornerArmLink{first}, ""
	}
	return rimContinuationLinks(first, spec.surface, plan, res)
}

// cornerArmStations returns the near station plus every INTERIOR seam station; the last station is fixed by
// the far runout in cornerArmWeldFrom.
func cornerArmStations(plan cornerWeldPlan, spec cornerArmSpec, links []cornerArmLink, near cornerArmStation, tol float64) ([]cornerArmStation, string) {
	out := make([]cornerArmStation, len(links)+1)
	out[0] = near
	for i := 0; i+1 < len(links); i++ {
		st, reason := cornerSeamStation(plan, spec, links[i], links[i+1], i, tol)
		if reason != "" {
			return nil, reason
		}
		out[i+1] = st
	}
	return out, ""
}

// cornerSeamStation is the cross-section station where a rim-continued arm crosses from one host-face span to
// the next: the rolling ball's centre at the seam vertex, its two contact feet on the OUTGOING link's hosts,
// and the radius-r cross-section arc between them — the same primitive a plane cap uses, registered once so
// the two arm faces meeting there read one curve object.
func cornerSeamStation(plan cornerWeldPlan, spec cornerArmSpec, in, out cornerArmLink, idx int, tol float64) (cornerArmStation, string) {
	ball, ok := armBallCenter(spec.surface, in.farVtx.Point())
	if !ok {
		return cornerArmStation{}, fmt.Sprintf("rim seam %d: arm spine undefined at vertex %d", idx, in.farVtx.ID())
	}
	footA, okA := armRunoutFoot(out.hostA, ball, plan.radius, tol)
	footB, okB := armRunoutFoot(out.hostB, ball, plan.radius, tol)
	if !okA || !okB {
		return cornerArmStation{}, fmt.Sprintf("rim seam %d: ball at vertex %d not tangent to a host (a=%v b=%v)",
			idx, in.farVtx.ID(), okA, okB)
	}
	xs, ok := farCrossSectionArc(spec.surface, plan.radius, footA, footB)
	if !ok {
		return cornerArmStation{}, fmt.Sprintf("rim seam %d: cross-section arc failed on feet %v→%v", idx, footA, footB)
	}
	return cornerArmStation{a: footA, b: footB, xs: plan.ledger.add(fmt.Sprintf("%s/seam[%d]", spec.role, idx), xs)}, ""
}

// cornerArmRailChain is the per-link contact rails, index-aligned with the links, each oriented FAR→NEAR.
type cornerArmRailChain struct {
	a, b []railID
}

// cornerArmRails builds every link's two contact rails plus the far termination. The LAST link runs through
// the shared far-runout engine (armFarRunout), which owns the perpendicular-vs-oblique dispatch and may move
// that link's rail ends onto the authoritative feet; interior links land between two known stations.
func cornerArmRails(plan cornerWeldPlan, spec cornerArmSpec, links []cornerArmLink, stations []cornerArmStation, res opstol.Resolution) (cornerArmRailChain, armRunout, string) {
	last := len(links) - 1
	chain := cornerArmRailChain{a: make([]railID, len(links)), b: make([]railID, len(links))}
	railA, railB, run, reason := cornerArmFarRails(plan, spec, links, stations[last], res)
	if reason != "" {
		return chain, run, reason
	}
	chain.a[last] = plan.ledger.add(fmt.Sprintf("%s/hostA[%d]", spec.role, last), railA)
	chain.b[last] = plan.ledger.add(fmt.Sprintf("%s/hostB[%d]", spec.role, last), railB)
	for i := range last {
		if reason := cornerArmLinkRails(plan, spec, links[i], stations[i+1], stations[i], chain, i, res); reason != "" {
			return chain, run, reason
		}
	}
	return chain, run, ""
}

// cornerArmFarRails lands the LAST link's contact rails from its near station out to its far vertex, then
// terminates it through armFarRunout. The synthesized cornerWeld centre is that link's OWN near vertex (the
// corner point for a single-link arm, the preceding seam vertex for a continued one) so farEndVertex and
// farVertexNotVid2 resolve the intended end — the pattern singleRunoutTrims already uses.
func cornerArmFarRails(plan cornerWeldPlan, spec cornerArmSpec, links []cornerArmLink, near cornerArmStation, res opstol.Resolution) (endSeg, endSeg, armRunout, string) {
	last := len(links) - 1
	ef := spec.ef
	ef.edge, ef.a, ef.b = links[last].edge, links[last].hostA, links[last].hostB
	origin := plan.vertex
	if last > 0 {
		origin = links[last-1].farVtx.Point()
	}
	farA, farB, reason := mixedArmFarFeet(ef, spec.surface, origin, plan.radius, res)
	if reason != "" {
		return endSeg{}, endSeg{}, armRunout{}, reason
	}
	railA, okA := cornerLinkContactRail(ef.a, links[last].edge, spec.surface, farA, near.a, res)
	railB, okB := cornerLinkContactRail(ef.b, links[last].edge, spec.surface, farB, near.b, res)
	if !okA || !okB {
		return endSeg{}, endSeg{}, armRunout{}, fmt.Sprintf("a host contact rail could not be built (a=%v b=%v)", okA, okB)
	}
	railA, railB, run, ok, reason := armFarRunout(ef, cornerWeld{center: origin, radius: plan.radius}, railA, railB, plan.filleted, res)
	if !ok {
		return endSeg{}, endSeg{}, run, "far runout: " + reason
	}
	return railA, railB, run, ""
}

// cornerArmLinkRails lands one INTERIOR link's contact rails between its far and near stations, on that
// link's own two host faces.
func cornerArmLinkRails(plan cornerWeldPlan, spec cornerArmSpec, link cornerArmLink, far, near cornerArmStation, chain cornerArmRailChain, i int, res opstol.Resolution) string {
	railA, okA := cornerLinkContactRail(link.hostA, link.edge, spec.surface, far.a, near.a, res)
	railB, okB := cornerLinkContactRail(link.hostB, link.edge, spec.surface, far.b, near.b, res)
	if !okA || !okB {
		return fmt.Sprintf("rim link %d: a host contact rail could not be built (a=%v b=%v)", i, okA, okB)
	}
	chain.a[i] = plan.ledger.add(fmt.Sprintf("%s/hostA[%d]", spec.role, i), railA)
	chain.b[i] = plan.ledger.add(fmt.Sprintf("%s/hostB[%d]", spec.role, i), railB)
	return ""
}

// assignFeetToHosts returns a near boundary's endpoint on ef.a (nearA) and on ef.b (nearB) — the
// general-curve sibling of assignArcFeetToHosts (which is Arc3d-specific), for an A2 arm whose near
// boundary is a fitted rail rather than a radius-r arc. ok=false when an endpoint lands on neither host.
func assignFeetToHosts(ef edgeFillet, p0, p1 math.Point3, tol float64) (math.Point3, math.Point3, bool) {
	if onHostSurface(ef.a.Geometry(), p0, tol) && onHostSurface(ef.b.Geometry(), p1, tol) {
		return p0, p1, true
	}
	if onHostSurface(ef.a.Geometry(), p1, tol) && onHostSurface(ef.b.Geometry(), p0, tol) {
		return p1, p0, true
	}
	return math.Point3{}, math.Point3{}, false
}

// cornerLinkContactRail builds one link's host contact rail. It takes armRunoutRail's rail whenever the span
// admits a three-point fit; a span of a HALF TURN or more does NOT — its chord passes through the contact
// circle's centre, so arcMidBetween's on-circle midpoint degenerates to the centre itself and the three
// points are collinear. Such a span is instead built as the contact-circle sub-arc carrying the link edge's
// OWN signed sweep, the construction reflexContactRail uses for a reflex pick. N4's second rim link is
// exactly 180°, which is precisely the case the three-point fit cannot express.
func cornerLinkContactRail(host *topo.Face, picked *topo.Edge, arm geom.Surface, foot0, foot1 math.Point3, res opstol.Resolution) (endSeg, bool) {
	if seg, ok := armRunoutRail(host, picked, arm, foot0, foot1, res); ok {
		return seg, true
	}
	return sweptContactRail(host, picked, arm, foot0, foot1, res)
}

// sweptContactRail builds a torus arm's host contact rail as the arc of the host contact circle from foot0 to
// foot1 carrying the link edge's own signed sweep — exact for a half-turn or larger span, where the
// three-point fit has no answer. The rail's half is fixed by its MIDPOINT, not by its endpoints: at exactly
// 180° both halves share the same endpoints, so the discriminator is the ball's contact foot at the link
// edge's own midpoint, which lies on the correct half by construction. ok=false for a non-torus arm, a
// non-arc edge, a host with no contact circle, or when neither sweep sign reproduces that midpoint.
func sweptContactRail(host *topo.Face, picked *topo.Edge, arm geom.Surface, foot0, foot1 math.Point3, res opstol.Resolution) (endSeg, bool) {
	tor, okT := arm.(geom.Torus)
	arc, okA := picked.Geometry().(geom.Arc3d)
	if !okT || !okA {
		return endSeg{}, false
	}
	center, radius, ok := torusContactCircle(host.Geometry(), tor, res)
	if !ok {
		return endSeg{}, false
	}
	footMid, ok := linkMidContactFoot(host, picked, arm, res)
	if !ok {
		return endSeg{}, false
	}
	// Scale the tol on the ROLLING-BALL radius, as every other tol in the weld layer does
	// (res.Weld()*plan.radius). Scaling it on the CONTACT-CIRCLE radius instead — 15 or 20 here against a
	// fillet r of 5 — would make the one check that discriminates the rail's half 3–4× looser than the
	// checks around it, which is backwards for the layer's single wrong-half guard.
	tol := tangencyTol(tor.MinorRadius, res)
	for _, sweep := range [2]float64{arc.SweepAngle, -arc.SweepAngle} {
		parent, err := geom.NewArc3d(center, arc.Normal.AsVector(), center.VectorTo(foot0), radius, 0, sweep)
		if err != nil || float64(parent.PointAt(1).DistanceTo(foot1)) > tol {
			continue
		}
		if mid := parent.PointAt(0.5); float64(mid.DistanceTo(footMid)) <= tol {
			return endSeg{from: foot0, to: foot1, curve: parent, mid: mid, arc: true}, true
		}
	}
	return endSeg{}, false
}

// linkMidContactFoot is the arm ball's contact foot on host at the link edge's own midpoint — the witness
// that fixes which half of the contact circle the rail runs along. Its tangency tol scales on the
// rolling-ball radius, matching the rest of the weld layer (see sweptContactRail).
func linkMidContactFoot(host *topo.Face, picked *topo.Edge, arm geom.Surface, res opstol.Resolution) (math.Point3, bool) {
	ball, ok := armBallCenter(arm, picked.Geometry().PointAt(0.5))
	if !ok {
		return math.Point3{}, false
	}
	r, ok := armTubeRadius(arm)
	if !ok {
		return math.Point3{}, false
	}
	return armRunoutFoot(host, ball, r, tangencyTol(r, res))
}
