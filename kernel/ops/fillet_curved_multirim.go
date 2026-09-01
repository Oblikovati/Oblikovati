// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
)

// Multiple INDEPENDENT closed rims in one fillet op (W-DH capability wave, bfuseblend/B3). A
// through-cylinder fused to a box leaves TWO closed concave Cylinder∧Plane rims — one per exit face.
// Each rim alone is the proven single-rim band (FilletCylinderRim: the A1/W6/W8 boss-base cove and the
// J1/K1/Z1 convex/bore bands), but the multi-pick op fell through to the trihedral corner weld and
// floored on "trihedral corner needs 3 arms" — there is no corner: the rims never touch. This file
// routes a pick set of ≥2 pairwise face-disjoint closed rims through the SAME single-rim rebuild,
// sequentially: each rebuild copies every untouched vertex/edge/face with its Lineage carried verbatim
// (fillet_rim_build.go copyEdges), so the next rim's ReferenceKey resolves on the intermediate body
// exactly as on the original — the ADR-0043 stable-name contract doing the composition work.

// independentClosedRimsBody welds N ≥ 2 pairwise-disjoint CLOSED circular Cylinder∧Plane rims by
// applying the single-rim band rebuild once per rim. took=false leaves every other pick set on its
// existing path (single rims and miters never reach it; corner sets fail the all-closed gate).
// A resolve/rebuild decline returns the honest reason — do-no-harm, never a partial body.
func independentClosedRimsBody(body *topo.Body, fils []edgeFillet) (*topo.Body, string, bool) {
	if !allDisjointClosedRims(fils) {
		return nil, "", false
	}
	cur := body
	for _, ef := range fils {
		r, ok := rimArmRadius(ef)
		if !ok {
			return nil, fmt.Sprintf("closed rim edge %d carries no analytic arm radius", ef.edge.ID()), true
		}
		b, err := FilletCylinderRim(cur, ef.edge.ReferenceKey(), r)
		if err != nil {
			return nil, fmt.Sprintf("independent closed rim edge %d declined: %v", ef.edge.ID(), err), true
		}
		cur = b
	}
	return cur, "", true
}

// allDisjointClosedRims reports whether EVERY pick is a constant-radius CLOSED circular Cylinder∧Plane
// rim and no two rims share a host face. ≥2 rims required — a single rim already routes through
// loneRimPick before the weld dispatch ever runs, so this branch fires only for the multi-rim op.
func allDisjointClosedRims(fils []edgeFillet) bool {
	if len(fils) < 2 {
		return false
	}
	seen := map[*topo.Face]bool{}
	for _, ef := range fils {
		if ef.varying || ef.edge == nil || !probe.IsClosedCircularEdge(ef.edge) {
			return false
		}
		if _, _, ok := cylinderPlaneEdge(ef.edge); !ok {
			return false // only the Cylinder∧Plane rim family has the single-rim band rebuild
		}
		if seen[ef.a] || seen[ef.b] {
			return false // two rims biting one face need a composed retrim — not this slice
		}
		seen[ef.a], seen[ef.b] = true, true
	}
	return true
}

// rimArmRadius is the pick's rolling-ball radius, read from the exact analytic arm the fillet solver
// attached: a cove/band torus arm's minor radius, or a cylinder arm's radius. ok=false when the arm is
// neither (the pick then cannot be a closed circular rim of this family).
func rimArmRadius(ef edgeFillet) (float64, bool) {
	switch s := ef.armSurface.(type) {
	case geom.Torus:
		return s.MinorRadius, true
	case geom.Cylinder:
		return s.Radius, true
	}
	return 0, false
}
