// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// weldCornerPlan — THE shared corner-weld executor (corner-weld-layer-design.md ADR-1). It runs the six
// stages every bespoke weld already runs, but ONCE, driven by a declarative plan:
//
//	1 classify      the builder's job (it handed in the plan)
//	2 solve corner  the builder's job (patch + rails are already in the ledger)
//	3 per-arm bundle    cornerArmWelds — contact rails + far termination per arm
//	4 emit faces        the arm faces + the patch face, every side read from the ledger by handle
//	5 retrim hosts      cornerPlanHostFaces — corner hosts, single-arm hosts, far caps, passthrough
//	6 certify+assemble  certifyTwoIncident THEN assembleBody
//
// Any decline returns (nil, reason) — the do-no-harm floor. Never a partial body, never a widened
// tolerance (invariant #3).

// railRef is one face's claim on a ledger rail, in the direction that face traverses it.
type railRef struct {
	id  railID
	dir railSense
}

// cornerFaceRing is one emitted face: a surface plus an ordered ring of rail claims. The ring's segs are
// resolved from the ledger at emission time, so two faces sharing a boundary get the SAME curve object.
type cornerFaceRing struct {
	surface geom.Surface
	sides   []railRef
	parent  topo.Lineage
}

// cornerArmHostBite is one arm's contact-rail chain on ONE host face, oriented FAR→NEAR, together with the
// host loop edges the chain replaces (the picked edge, plus any edge a rim continuation ran through).
type cornerArmHostBite struct {
	face     *topo.Face
	rails    []railID
	consumed []*topo.Edge
}

// cornerCapBite is one arm's far-termination bite on its capping face: the runout trim (shared with the
// arm face by handle), the far vertex point the bite replaces, and whether the cap recedes or grows.
type cornerCapBite struct {
	face  *topo.Face
	trim  railID
	far   math.Point3
	sense retrimSense
}

// cornerArmWeld is one arm's stage-3 bundle: the faces it emits (more than one when a rim continuation
// crosses a host-face seam), its per-host bites, and its far cap bite.
type cornerArmWeld struct {
	faces []cornerFaceRing
	hosts []cornerArmHostBite
	cap   cornerCapBite
}

// weldCornerPlan turns one corner site's plan into a watertight body, or declines with a named reason.
// Example:
//
//	plan, took, why := builder.Plan(body, arms, res)
//	if took && why == "" { welded, why := weldCornerPlan(body, plan, res) }
func weldCornerPlan(body *topo.Body, plan cornerWeldPlan, res Resolution) (*topo.Body, string) {
	welds, reason := cornerArmWelds(plan, res)
	if reason != "" {
		return nil, reason
	}
	faces, reason := cornerPlanBlendFaces(plan, welds)
	if reason != "" {
		return nil, reason
	}
	hostFaces, reason := cornerPlanHostFaces(body, plan, welds, res)
	if reason != "" {
		return nil, reason
	}
	if why := plan.ledger.certifyTwoIncident(); why != "" {
		return nil, why // a crack, named at the PLAN level instead of a downstream Closed=false
	}
	return assembleBody(append(faces, hostFaces...)), ""
}

// cornerArmWelds runs stage 3 for every arm in plan order. A decline carries the arm's role label so the
// floor reads like the bespoke welds' messages.
func cornerArmWelds(plan cornerWeldPlan, res Resolution) ([]cornerArmWeld, string) {
	out := make([]cornerArmWeld, 0, len(plan.arms))
	for _, spec := range plan.arms {
		w, reason := cornerArmWeldOf(plan, spec, res)
		if reason != "" {
			return nil, spec.role + " arm: " + reason
		}
		out = append(out, w)
	}
	return out, ""
}

// cornerPlanBlendFaces runs stage 4: the arm faces plus the corner patch, each side resolved from the
// ledger. The patch is emitted as SINGLE curve-segs (never a sampled polyline), so the tessellator reads
// each side identically on both faces. Each emitted face claims its sides under its OWN ordinal, so the
// ledger's certificate counts distinct faces rather than raw references.
func cornerPlanBlendFaces(plan cornerWeldPlan, welds []cornerArmWeld) ([]filletFace, string) {
	rings := make([]cornerFaceRing, 0, len(welds)+1)
	for _, w := range welds {
		rings = append(rings, w.faces...)
	}
	rings = append(rings, cornerFaceRing{surface: plan.patch.surface, sides: forwardRefs(plan.patch.sides)})
	out := make([]filletFace, 0, len(rings))
	for i, ring := range rings {
		ff, ok := cornerRingFace(plan.ledger, ring, blendClaimant(i))
		if !ok {
			return nil, fmt.Sprintf("corner weld: blend face %d of %d claims an unregistered rail (an unset handle)", i, len(rings))
		}
		out = append(out, ff)
	}
	return out, ""
}

// cornerRingFace resolves a face ring's rail claims into one filletFace loop. ok=false when a side carries
// an unset handle — a zero endSeg there would weld a degenerate segment into the loop.
func cornerRingFace(led *cornerWeldLedger, ring cornerFaceRing, by railClaimant) (filletFace, bool) {
	segs := make([]endSeg, 0, len(ring.sides))
	for _, ref := range ring.sides {
		s, ok := led.seg(ref.id, ref.dir, by)
		if !ok {
			return filletFace{}, false
		}
		segs = append(segs, s)
	}
	return filletFace{surface: ring.surface, loops: []filletLoop{loopFromSegs(segs)}, parent: ring.parent}, true
}

// forwardRefs claims every rail in a chain in its registered direction.
func forwardRefs(ids []railID) []railRef {
	out := make([]railRef, len(ids))
	for i, id := range ids {
		out[i] = railRef{id: id, dir: railForward}
	}
	return out
}

// reversedRefs claims a chain end-to-end backwards (reverse order, each piece reversed) — how the second
// face of a shared chain reads it.
func reversedRefs(ids []railID) []railRef {
	out := make([]railRef, 0, len(ids))
	for i := len(ids) - 1; i >= 0; i-- {
		out = append(out, railRef{id: ids[i], dir: railReversed})
	}
	return out
}
