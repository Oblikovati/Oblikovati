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

// ruledBuild builds the exact analytic result for one curved pair. The transversal SSI pipeline builders
// (brep.*General) take the imprint recorder; the curved-on-planar and degenerate-overlap constructors
// (drill-through, coaxial, boss — ADR-0045) trace no SSI imprint, take none, and are adapted with
// withoutRecorder.
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
		if !ok || !Validate(res).ValidSolid() {
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
// handles. Most are the TRANSVERSAL-crossing KIND (ADR-0045): the general SSI→trim→classify→stitch pipeline
// (#1403/#1476) builds every ruled pair, and the whole equal-radius Steinmetz family — intersect, cut AND
// join — now rides it (#1403), its self-intersecting imprint split at the analytic pinches into four open
// arcs so the arrangement never sees the crossing (brep.Steinmetz*General). The rest correctly do NOT ride
// the (u,v) arrangement: the drill through-hole and the cylinder boss are CURVED-ON-PLANAR (a tool cylinder
// crossing a planar face in a strictly-interior circle), the coaxial cylinder union is a DEGENERATE OVERLAP
// (coincident side surfaces, no SSI curve), and the coaxial ball-and-rod family is transversal but meets in
// one PLANAR circle on a surface that is not a rim-bounded band, so the split is by construction (#2036).
// See ADR-0045 for why each stays a distinct analytic handler rather than folding into the pipeline.
var (
	// Intersect — the band of the thin operand plus the fat operand's two lens caps.
	curvedCrossingIntersect  = gatedCurved(Intersect, brep.CrossingCylinderIntersectGeneral) // two crossing cylinders
	curvedSteinmetzIntersect = gatedCurved(Intersect, brep.SteinmetzIntersectGeneral)        // equal-R bicylinder, general pipeline
	// cone ∩ cylinder AND cone ∩ fatter cone: one unified ruled-crossing driver (ADR-0058 phase 3).
	curvedConeCrossingIntersect = gatedCurved(Intersect, brep.RuledConeCrossingIntersectGeneral)
	curvedPartialIntersect      = gatedCurved(Intersect, brep.PartialPenetrationIntersectGeneral)
	curvedBallRodIntersect      = gatedCurved(Intersect, withoutRecorder(brep.CoaxialSphereRodIntersect)) // coaxial ball ∩ rod: the plug

	// Cut — drilling the target with the tool (through, blind, or two stubs of tool − target).
	curvedCylindricalHoleCut  = gatedCurved(Cut, withoutRecorder(brep.DrillThroughHole)) // straight cylinder through a planar slab, hole strictly interior
	curvedEdgeScallopCut      = gatedCurved(Cut, withoutRecorder(brep.CutEdgeScallop))   // straight cylinder through a slab, circle CLIPS one edge (#1591)
	curvedPartialCut          = gatedCurved(Cut, brep.PartialPenetrationCutGeneral)      // blind rod hole
	curvedSteinmetzCut        = gatedCurved(Cut, brep.SteinmetzCutGeneral)               // equal-R bicylinder bite, general pipeline
	curvedConeCylinderCut     = gatedCurved(Cut, brep.ConeCylinderCutGeneral)
	curvedConeConeCut         = gatedCurved(Cut, brep.ConeConeCutGeneral)
	curvedCrossingCut         = gatedCurved(Cut, brep.CrossingCylinderCutGeneral)
	curvedCapCrossCut         = gatedCurved(Cut, brep.CapCrossingCutGeneral)                // oblique tool exits one cap, ellipse inside rim (#1724)
	curvedRimCrossCut         = gatedCurved(Cut, brep.RimCrossingCutGeneral)                // oblique tool exits one cap, ellipse crosses rim (#1724 slice 2)
	curvedTwoCapCrossCut      = gatedCurved(Cut, brep.TwoCapCrossingCutGeneral)             // steep tool exits BOTH caps, wall intact (#1724)
	curvedConeCapCrossCut     = gatedCurved(Cut, brep.ConeCapCrossingCutGeneral)            // oblique CONE tool exits one cap, ellipse inside rim (#1724)
	curvedPartialRimCut       = gatedCurved(Cut, brep.PartialRimCutGeneral)                 // second cut on an already-notched cylinder side, disjoint from the notch (#1732)
	curvedPartialRimCornerCut = gatedCurved(Cut, brep.PartialRimCornerCutGeneral)           // second cut whose imprint CROSSES the notch — the coupled corner-junction (#1738, ADR-0048)
	curvedBallRodCut          = gatedCurved(Cut, withoutRecorder(brep.CoaxialSphereRodCut)) // coaxial ball − rod (a blind spherical bore) and rod − ball (a dimpled stub), #2036

	// Join — the union, keeping the analytic wall where the operands' faces are coincident or breached.
	curvedCoaxialJoin      = gatedCurved(Join, withoutRecorder(brep.CoaxialCylinderUnion)) // coaxial equal-radius cylinders
	curvedBallRodJoin      = gatedCurved(Join, withoutRecorder(brep.CoaxialSphereRodJoin)) // coaxial ball ∪ rod: the ball stud, #2036
	curvedCylinderBossJoin = gatedCurved(Join, withoutRecorder(brep.JoinCylindricalBoss))  // cylinder seated flush on a face, base strictly interior
	curvedPartialBossJoin  = gatedCurved(Join, withoutRecorder(brep.JoinPartialBoss))      // cylinder boss whose base circle straddles the seat edge (#1591)
	curvedPartialJoin      = gatedCurved(Join, brep.PartialPenetrationJoinGeneral)         // fat + entry stub
	curvedConeCylinderJoin = gatedCurved(Join, brep.ConeCylinderJoinGeneral)
	curvedConeConeJoin     = gatedCurved(Join, brep.ConeConeJoinGeneral)
	curvedCrossingJoin     = gatedCurved(Join, brep.CrossingCylinderJoinGeneral)
	curvedSteinmetzJoin    = gatedCurved(Join, brep.SteinmetzJoinGeneral) // equal-R bicylinder union, general pipeline
)
