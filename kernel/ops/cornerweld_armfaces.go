// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/topo"
)

// Stage 4b of the shared corner-weld executor: the arm FACES and the per-host / per-cap bites one arm
// contributes. Each face is the four-sided ring [hostA rail (far→near), the link's near boundary (hostA foot →
// hostB foot), hostB rail reversed, the link's far boundary reversed]. That is the shape mixedArmFace emits,
// generalized so a rim-continued arm's interior seam cross-section is the far boundary of one face and the
// near boundary of the next — which is how the band splits into the faces OCCT emits. Every side comes from
// the ledger by handle, so both faces of every boundary read ONE curve object.

// cornerArmWeldFrom assembles one arm's emitted faces plus its host and cap bites, once its links, stations,
// rails and far runout are known. It also fixes the LAST station (the far runout's feet + trim).
func cornerArmWeldFrom(plan cornerWeldPlan, spec cornerArmSpec, links []cornerArmLink, stations []cornerArmStation, rails cornerArmRailChain, run armRunout) (cornerArmWeld, string) {
	last := len(links) - 1
	farTrim := plan.ledger.add(spec.role+"/far", run.trim)
	stations[last+1] = cornerArmStation{a: run.feet[0], b: run.feet[1], xs: farTrim}
	nearDir, ok := cornerNearDir(plan, spec, stations[0])
	if !ok {
		return cornerArmWeld{}, "near boundary chain does not run between the two host feet"
	}
	faces := make([]cornerFaceRing, 0, len(links))
	for i := range links {
		faces = append(faces, cornerArmLinkFace(spec, rails, stations, i, nearDir))
	}
	hosts, reason := cornerArmHostBites(links, rails)
	if reason != "" {
		return cornerArmWeld{}, reason
	}
	cap := cornerCapBite{face: run.capping, trim: farTrim, far: links[last].farVtx.Point(), sense: spec.sense}
	return cornerArmWeld{faces: faces, hosts: hosts, cap: cap}, ""
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

// cornerArmLinkFace emits link i's face: its hostA rail (far→near), its near boundary (the corner patch's
// rail for link 0, the preceding seam cross-section otherwise), its hostB rail reversed, and its far boundary
// (the next seam cross-section, or the far runout trim) reversed.
func cornerArmLinkFace(spec cornerArmSpec, rails cornerArmRailChain, stations []cornerArmStation, i int, nearDir railSense) cornerFaceRing {
	sides := []railRef{{id: rails.a[i], dir: railForward}}
	if i == 0 {
		sides = append(sides, cornerNearRefs(spec.near, nearDir)...)
	} else {
		sides = append(sides, railRef{id: stations[i].xs, dir: railForward})
	}
	sides = append(sides, railRef{id: rails.b[i], dir: railReversed}, railRef{id: stations[i+1].xs, dir: railReversed})
	return cornerFaceRing{surface: spec.surface, sides: sides, parent: filletEdgeProvenance(spec.ef.edge)}
}

// cornerNearRefs claims the near boundary chain in the direction that runs hostA foot → hostB foot.
func cornerNearRefs(near []railID, dir railSense) []railRef {
	if dir == railForward {
		return forwardRefs(near)
	}
	return reversedRefs(near)
}

// cornerArmHostBites groups the arm's per-link contact rails by HOST FACE, in FAR→NEAR chain order (so a host
// crossed by several links — N4's shared top plane, which the rim continuation runs right across — receives
// ONE contiguous chain replacing ALL the loop edges the arm consumed there). Declines when a link rides on
// one face twice, which no retrim can re-clip.
func cornerArmHostBites(links []cornerArmLink, rails cornerArmRailChain) ([]cornerArmHostBite, string) {
	acc := newHostBiteAccumulator()
	for i := len(links) - 1; i >= 0; i-- {
		if links[i].hostA == links[i].hostB {
			return nil, fmt.Sprintf("link %d rides on face %d twice", i, links[i].hostA.ID())
		}
		acc.add(links[i].hostA, rails.a[i], links[i].edge)
		acc.add(links[i].hostB, rails.b[i], links[i].edge)
	}
	return acc.bites, ""
}

// hostBiteAccumulator groups rails by host face while preserving first-seen face order and per-face chain
// order — a plain map would randomise both, and the retrim consumes the chain in order.
type hostBiteAccumulator struct {
	bites []cornerArmHostBite
	index map[*topo.Face]int
}

func newHostBiteAccumulator() *hostBiteAccumulator {
	return &hostBiteAccumulator{index: map[*topo.Face]int{}}
}

// add appends one link's rail (and the loop edge it replaces) to that host face's bite.
func (a *hostBiteAccumulator) add(face *topo.Face, rail railID, edge *topo.Edge) {
	k, seen := a.index[face]
	if !seen {
		a.index[face] = len(a.bites)
		a.bites = append(a.bites, cornerArmHostBite{face: face, rails: []railID{rail}, consumed: []*topo.Edge{edge}})
		return
	}
	a.bites[k].rails = append(a.bites[k].rails, rail)
	a.bites[k].consumed = append(a.bites[k].consumed, edge)
}
