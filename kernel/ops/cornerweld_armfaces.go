// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/topo"
)

// Stage 4b of the shared corner-weld executor: the arm FACE and the per-host / per-cap bites one arm
// contributes. The face is the four-sided ring [hostA rail (far→near), near boundary (hostA foot → hostB
// foot), hostB rail reversed, far trim reversed] — the shape mixedArmFace emits, with every side taken from
// the ledger by handle so the arm and its neighbour (patch, host retrim, cap) read ONE curve object.

// cornerArmWeldFrom assembles one arm's emitted face plus its host and cap bites, once its link, near
// station, contact rails and far runout are known.
func cornerArmWeldFrom(plan cornerWeldPlan, spec cornerArmSpec, link cornerArmLink, near cornerArmStation, rails cornerArmRailChain, run armRunout) (cornerArmWeld, string) {
	farTrim := plan.ledger.add(spec.role+"/far", run.trim)
	nearDir, ok := cornerNearDir(plan, spec, near)
	if !ok {
		return cornerArmWeld{}, "near boundary chain does not run between the two host feet"
	}
	sides := []railRef{{id: rails.a, dir: railForward}}
	sides = append(sides, cornerNearRefs(spec.near, nearDir)...)
	sides = append(sides, railRef{id: rails.b, dir: railReversed}, railRef{id: farTrim, dir: railReversed})
	face := cornerFaceRing{surface: spec.surface, sides: sides, parent: filletEdgeProvenance(spec.ef.edge)}
	hosts := []cornerArmHostBite{
		{face: link.hostA, rails: []railID{rails.a}, consumed: []*topo.Edge{link.edge}},
		{face: link.hostB, rails: []railID{rails.b}, consumed: []*topo.Edge{link.edge}},
	}
	cap := cornerCapBite{face: run.capping, trim: farTrim, far: link.farVtx.Point(), sense: spec.sense}
	return cornerArmWeld{faces: []cornerFaceRing{face}, hosts: hosts, cap: cap}, ""
}

// cornerNearDir reports which direction the plan's near boundary chain must be read so it runs from the
// hostA foot to the hostB foot (the arm face's loop order). ok=false when it does not span the two feet —
// an inconsistent plan, floored rather than welded backwards.
func cornerNearDir(plan cornerWeldPlan, spec cornerArmSpec, near cornerArmStation) (railSense, bool) {
	p0, p1, ok := plan.ledger.ends(spec.near)
	if !ok {
		return railForward, false
	}
	tol := railGreatCircleTol * plan.radius
	if float64(p0.DistanceTo(near.a)) <= tol && float64(p1.DistanceTo(near.b)) <= tol {
		return railForward, true
	}
	if float64(p1.DistanceTo(near.a)) <= tol && float64(p0.DistanceTo(near.b)) <= tol {
		return railReversed, true
	}
	return railForward, false
}

// cornerNearRefs claims the near boundary chain in the direction that runs hostA foot → hostB foot.
func cornerNearRefs(near []railID, dir railSense) []railRef {
	if dir == railForward {
		return forwardRefs(near)
	}
	return reversedRefs(near)
}
