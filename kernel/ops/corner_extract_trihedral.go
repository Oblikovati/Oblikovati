// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/topo"
)

// extractTrihedral turns a solved planar-trihedral cornerBlend into the 3-sided RailLoop the exact
// sphere tier recognizes (three concentric equal-radius great-circle arcs on the corner sphere). Each
// Side's Curve is the boundary arc between successive arm tangent points, lying on cb.sphere; Adjacent
// is the sphere itself (recognition is rails-only, so Adjacent is not load-bearing here) and Cont is
// G1. This is the PLANAR strangler path.
//
// The Sides MUST be in head-to-tail chain order — RailLoop.Closed (and thus the sphere provider's
// certificate) requires consecutive sides to share endpoints — but cb.arcs is appended in arbitrary
// edge-pick order (fillet.go registerBlendArc). We therefore reuse chainArcs, which re-orders the same
// arcs head-to-tail and reconstructs each one with the SAME Arc3dByThreePoints construction the
// byte-for-byte boundary loop uses, so the extractor's rails are IDENTICAL to that loop's. ok=false if
// the arcs don't form three analytic arcs (e.g. a variable-cone chord segment) → honest-reject, letting
// spherePatchFace fall back to cb.sphere (do-no-harm).
//
// Usage: loop, ok := extractTrihedral(cb); if ok { patch, ok := resolveBlend(loop, scale) }.
func extractTrihedral(cb *cornerBlend) (RailLoop, bool) {
	if len(cb.arcs) != 3 {
		return RailLoop{}, false
	}
	loop := chainArcs(cb.arcs)
	if len(loop.curves) != 3 {
		return RailLoop{}, false
	}
	sides := make([]Side, 0, 3)
	for _, curve := range loop.curves {
		if curve == nil { // a faceted variable-cone chord: not the analytic sphere path
			return RailLoop{}, false
		}
		sides = append(sides, Side{Curve: curve, Adjacent: cb.sphere, Cont: G1})
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}
