// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
)

// Exact analytic curved-boolean paths for crossing/coaxial cylinder & cone pairs (M2, EPIC #1403).
// Each path routes a recognised (op, target, tool) pair to a brep builder that constructs the result
// straight from the traced SSI imprint as a watertight ANALYTIC solid — keeping the exact cylinder/cone
// surfaces instead of triangle-soup CSG. A pair the builder does not handle (a non-matching op, an
// out-of-scope configuration, an inside-out or open assembly) returns ok=false so booleanGeneral keeps
// its planar CSG fallback — no regression.
//
// Every path is the SAME two checks around a builder — the op guard and the validBooleanSolid gate —
// so they are written ONCE in gatedCurved (#1502). The gate is a correctness invariant (an unvalidated
// boolean must never be adopted); centralising it means a pair added to the table below cannot forget
// it, the copy-paste hazard this file used to carry as 18 near-identical handlers.

// ruledBuild builds the exact analytic result for one curved pair. The general SSI pipeline builders
// (brep.*General) take the imprint recorder; the bespoke analytic constructors (equal-radius Steinmetz,
// drill-through, coaxial, boss) take none and are adapted with withoutRecorder.
type ruledBuild func(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool)

// gatedCurved wraps a builder with the op guard and the single validBooleanSolid gate every curved pair
// must pass, returning a curvedExactPaths entry. ok=false (defer to the CSG fallback) when the op does
// not match or the analytic result is not a valid closed manifold solid.
func gatedCurved(want PartFeatureOperation, build ruledBuild) func(PartFeatureOperation, *topo.Body, *topo.Body, *diag.Recorder) (*topo.Body, bool) {
	return func(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
		if op != want {
			return nil, false
		}
		res, ok := build(target, tool, rec)
		if !ok || !validBooleanSolid(res) {
			return nil, false
		}
		return res, true
	}
}

// withoutRecorder adapts a bespoke analytic constructor (one that traces no SSI imprint) to ruledBuild.
func withoutRecorder(build func(target, tool *topo.Body) (*topo.Body, bool)) ruledBuild {
	return func(target, tool *topo.Body, _ *diag.Recorder) (*topo.Body, bool) { return build(target, tool) }
}

// The curved-pair paths, each an (op, builder) gated once by gatedCurved. Names are referenced by
// curvedExactPaths in boolean.go, which fixes their try-order; the comment on each records the pair it
// handles. The general SSI→trim→classify→stitch pipeline (#1403/#1476) builds every ruled pair; the
// equal-radius Steinmetz INTERSECT now rides it too (#1403) — its self-intersecting imprint is split at the
// analytic pinches into four open arcs so the arrangement never sees the crossing (brep.SteinmetzIntersect-
// General). The Steinmetz CUT and JOIN keep the bespoke band assembler for now (their kept region is the
// cylinders' pinched OUTSIDE bands, not lobes; folding those into the general pipeline is tracked follow-up).
var (
	// Intersect — the band of the thin operand plus the fat operand's two lens caps.
	curvedCrossingIntersect     = gatedCurved(Intersect, brep.CrossingCylinderIntersectGeneral) // two crossing cylinders
	curvedSteinmetzIntersect    = gatedCurved(Intersect, brep.SteinmetzIntersectGeneral)        // equal-R bicylinder, general pipeline
	curvedConeCylinderIntersect = gatedCurved(Intersect, brep.ConeCylinderIntersectGeneral)     // cone ∩ cylinder
	curvedConeConeIntersect     = gatedCurved(Intersect, brep.ConeConeIntersectGeneral)         // cone ∩ fatter cone
	curvedPartialIntersect      = gatedCurved(Intersect, brep.PartialPenetrationIntersectGeneral)

	// Cut — drilling the target with the tool (through, blind, or two stubs of tool − target).
	curvedCylindricalHoleCut = gatedCurved(Cut, withoutRecorder(brep.DrillThroughHole)) // straight cylinder through a planar slab
	curvedPartialCut         = gatedCurved(Cut, brep.PartialPenetrationCutGeneral)      // blind rod hole
	curvedSteinmetzCut       = gatedCurved(Cut, brep.SteinmetzCutGeneral)               // equal-R bicylinder bite, general pipeline
	curvedConeCylinderCut    = gatedCurved(Cut, brep.ConeCylinderCutGeneral)
	curvedConeConeCut        = gatedCurved(Cut, brep.ConeConeCutGeneral)
	curvedCrossingCut        = gatedCurved(Cut, brep.CrossingCylinderCutGeneral)

	// Join — the union, keeping the analytic wall where the operands' faces are coincident or breached.
	curvedCoaxialJoin      = gatedCurved(Join, withoutRecorder(brep.CoaxialCylinderUnion)) // coaxial equal-radius cylinders
	curvedCylinderBossJoin = gatedCurved(Join, withoutRecorder(brep.JoinCylindricalBoss))  // cylinder seated flush on a face
	curvedPartialJoin      = gatedCurved(Join, brep.PartialPenetrationJoinGeneral)         // fat + entry stub
	curvedConeCylinderJoin = gatedCurved(Join, brep.ConeCylinderJoinGeneral)
	curvedConeConeJoin     = gatedCurved(Join, brep.ConeConeJoinGeneral)
	curvedCrossingJoin     = gatedCurved(Join, brep.CrossingCylinderJoinGeneral)
	curvedSteinmetzJoin    = gatedCurved(Join, brep.SteinmetzJoinGeneral) // equal-R bicylinder union, general pipeline
)
