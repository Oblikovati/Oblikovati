// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The corner-weld PLAN — the seam of the general corner weld/assembly layer
// (corner-weld-layer-design.md ADR-1). Three bespoke welds (the legacy sphere path, M8's 2r-torus, N4's
// coons4) run the SAME six stages and differ only in a small closed set of per-stage role choices. A
// corner CLASS therefore stops being ~450 lines of orchestration and becomes a declarative plan: which
// arms are incident, how each meets the corner, how each ends far away, which patch closes them. One
// shared executor (weldCornerPlan) runs the stages.
//
// Every shared boundary in a plan is a railID into the plan's ledger, so the patch and its neighbour arm
// read ONE curve object by construction (ADR-2). A builder that does not recognise its class returns
// took=false with ZERO side effects, so the dispatch ladder's fall-through is unchanged (invariant #5).

// armNearKind is how one arm meets the corner site (design Axis A). It selects how the executor assigns
// the near boundary's two feet to the arm's hosts — A1 keeps the byte-identical assignArcFeetToHosts call
// the built welds make, A2 uses its general-curve sibling.
type armNearKind int

const (
	// armTerminatesAtArc (A1) — the arm ENDS at the corner on a radius-r cross-section arc. M8's three
	// arms; N4's concave-cyl and planar-band arms.
	armTerminatesAtArc armNearKind = iota
	// armPassesLaterally (A2) — the arm runs PAST the corner; its near boundary is a general curve on the
	// arm's own surface (N4's convex-torus arm along rail B→C).
	armPassesLaterally
)

// farTermKind is how one arm ends at its far end (design Axis B).
type farTermKind int

const (
	// farCappedVertex (B1/B2) — the far vertex carries a unique transverse capping face; the shared
	// far-runout engine (armFarRunout) dispatches perpendicular vs oblique itself.
	farCappedVertex farTermKind = iota
	// farRimContinuation (B3) — the far vertex is a G1 SEAM, not a cap: the rim continues past it on the same
	// pair of host surfaces, so no transverse capping face exists there (armFarRunout's count==0 decline).
	// The arm runs THROUGH the seam and terminates at the end of the tangent chain, splitting into one face
	// per host-face span on the way. See cornerweld_far_rim.go.
	farRimContinuation
)

// retrimSense is how a host/cap is re-clipped where the arm meets it (design Axis C): a convex arm bites
// material away (the host recedes), a concave arm adds it (the host grows outward to the contact rail).
type retrimSense int

const (
	biteInward  retrimSense = iota // convex arm: the host/cap recedes around the bite
	growOutward                    // concave arm: the host/cap grows out to the contact rail
)

// cornerArmSpec declares ONE arm incident to the corner site. role is a diagnostic label only (it names
// the arm in every decline reason, so a floor still reads like the bespoke welds' messages).
type cornerArmSpec struct {
	role     string
	ef       edgeFillet
	surface  geom.Surface
	nearKind armNearKind
	nearArc  geom.Arc3d // A1 only: the radius-r cross-section arc (also registered as near)
	near     []railID   // the near boundary chain — shared with the patch BY HANDLE
	far      farTermKind
	sense    retrimSense
}

// cornerHostMid declares a patch side that rides on a true HOST rather than on an arm (design Axis A3 —
// M8's top-contact arc (d), N4's rail D→A on the shared vertical plane). The host's two arm rails are
// joined by these rails instead of meeting at a triple point.
type cornerHostMid struct {
	face  *topo.Face
	rails []railID
}

// cornerPatchSpec is the corner patch: its surface and the ordered ring of boundary rails (each an arm's
// near rail or a host mid rail). The ring is emitted as SINGLE curve-segs, never a sampled polyline, so the
// tessellator samples each side identically on both faces.
type cornerPatchSpec struct {
	surface geom.Surface
	sides   []railID
}

// cornerWeldPlan is one corner site's weld, declaratively. weldCornerPlan is the only consumer.
type cornerWeldPlan struct {
	ledger   *cornerWeldLedger
	patch    cornerPatchSpec
	arms     []cornerArmSpec
	mids     []cornerHostMid
	vertex   math.Point3
	radius   float64
	filleted map[uint64]bool // picked edge ids at this corner (armFarRunout's interference guard)
}

// cornerPlanBuilder is THE extension point: one implementation per corner CLASS. took=false means "not my
// class" and MUST leave no trace, so the dispatch ladder falls through exactly as it does today (ultimately
// to the untouched legacy sphere path). took=true with a non-empty reason floors the op with that reason.
type cornerPlanBuilder interface {
	Plan(body *topo.Body, arms []edgeFillet, res tol.Resolution) (cornerWeldPlan, bool, string)
}

// cornerWeldLayerBody is the layer's dispatch: the ordered builder ladder. The FIRST builder that takes the
// corner owns it; if none does, took=false and the caller keeps its existing path byte-identically. Only
// classes the bespoke welds cannot serve are routed here (design ADR-4's strangler rule: the legacy
// sphere-coupled weld and M8 are NOT migrated).
func cornerWeldLayerBody(body *topo.Body, arms []edgeFillet, res tol.Resolution) (*topo.Body, string, bool) {
	for _, b := range cornerPlanBuilders() {
		plan, took, reason := b.Plan(body, arms, res)
		if !took {
			continue
		}
		if reason != "" {
			return nil, reason, true // recognised its class but could not plan — floor honestly
		}
		welded, why := weldCornerPlan(body, plan, res)
		return welded, why, true
	}
	return nil, "", false
}

// cornerPlanBuilders is the ordered ladder. Each recogniser must be DISJOINT from every prior class
// (asserted by the class-disjointness tests), so order never changes a verdict.
func cornerPlanBuilders() []cornerPlanBuilder {
	return []cornerPlanBuilder{n4CornerPlanBuilder{}, o1CornerPlanBuilder{}}
}
