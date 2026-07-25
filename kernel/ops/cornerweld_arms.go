// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Stage 3+4a of the shared corner-weld executor: turn one cornerArmSpec into its two contact rails, its far
// termination, and the arm FACE it emits. It CALLS the existing primitives (assignArcFeetToHosts /
// assignFeetToHosts, mixedArmHostRails, armFarRunout) and never reimplements them — this stage is exactly
// what buildMixedArmBundle + mixedArmFace do in the two bespoke mixed-corner welds, with the per-case
// orchestration lifted out and every shared boundary handed to the ledger.

// cornerArmLink is the span of an arm between the corner site and its far termination: the loop edge it
// consumes, the two host faces it rides on, and the vertex closing its far end.
type cornerArmLink struct {
	edge   *topo.Edge
	hostA  *topo.Face
	hostB  *topo.Face
	farVtx *topo.Vertex
}

// cornerArmStation is one cross-section boundary of the arm — its two contact feet (on hostA / hostB) and
// the curve joining them. xs is noRail at the NEAR station, whose boundary is the corner patch's rail.
type cornerArmStation struct {
	a, b math.Point3
	xs   railID
}

// cornerArmWeldOf runs stage 3 for one arm: near feet → link → contact rails + far runout → face + bites.
func cornerArmWeldOf(plan cornerWeldPlan, spec cornerArmSpec, res Resolution) (cornerArmWeld, string) {
	tol := res.Weld() * plan.radius
	near, reason := cornerNearStation(plan, spec, tol)
	if reason != "" {
		return cornerArmWeld{}, reason
	}
	link := cornerArmLink{
		edge: spec.ef.edge, hostA: spec.ef.a, hostB: spec.ef.b,
		farVtx: farEndVertex(spec.ef.edge, plan.vertex),
	}
	railA, railB, run, reason := cornerArmFarRails(plan, spec, link, near, res)
	if reason != "" {
		return cornerArmWeld{}, reason
	}
	rails := cornerArmRailChain{
		a: plan.ledger.add(spec.role+"/hostA", railA),
		b: plan.ledger.add(spec.role+"/hostB", railB),
	}
	return cornerArmWeldFrom(plan, spec, link, near, rails, run)
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

// cornerArmRailChain is one arm's two contact-rail handles, each oriented FAR→NEAR on its host.
type cornerArmRailChain struct {
	a, b railID
}

// cornerArmFarRails lands the arm's two contact rails from its near station out to its far vertex, then runs
// the far end through the shared far-runout engine (armFarRunout), which owns the perpendicular-vs-oblique
// dispatch and may move the rails' outer ends onto the authoritative feet.
func cornerArmFarRails(plan cornerWeldPlan, spec cornerArmSpec, link cornerArmLink, near cornerArmStation, res Resolution) (endSeg, endSeg, armRunout, string) {
	ef := spec.ef
	ef.edge, ef.a, ef.b = link.edge, link.hostA, link.hostB
	railA, railB, reason := mixedArmHostRails(ef, spec.surface, near.a, near.b, plan.vertex, plan.radius, res)
	if reason != "" {
		return endSeg{}, endSeg{}, armRunout{}, reason
	}
	w := cornerWeld{center: plan.vertex, radius: plan.radius}
	railA, railB, run, ok, reason := armFarRunout(ef, w, railA, railB, plan.filleted, res)
	if !ok {
		return endSeg{}, endSeg{}, run, "far runout: " + reason
	}
	return railA, railB, run, ""
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
