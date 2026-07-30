// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
)

// The edge catalog welds one shared topo edge per (vertex pair, identity class), and BOTH faces
// that meet on it offer a curve for it. Historically the FIRST offer won and every later one was
// discarded silently, so which geometry shipped was decided by the order the assembler happened to
// visit the faces in. This file is the resolution rule that replaces that:
//
//   - two curves that disagree geometrically → a real conflict. The assembler cannot know which
//     one is right, so it must not silently arbitrate: it keeps the first (so the build stays
//     deterministic) and records a Defect naming the welded pair, its endpoints, BOTH curves and
//     the measured deviation. `simple/M8` shipped exactly one of these — a 255.52° boss rim arc
//     against its own 104.48° COMPLEMENT, 22.01 apart, decided only by the plane being built
//     before the cylinder (fixed at source in rebuildArcSeg). The standing corpus gate holds this
//     class at ZERO.
//   - nil vs a curve → recorded as a Warning debt, and the FIRST offer still stands. A nil is an
//     absence of information rather than an assertion of straightness, so the curve is the better
//     geometry FOR THE EDGE — measured: adopting it takes the off-surface residual of every case
//     that carries this debt down 2.8–4.6×. It is nevertheless NOT adopted here, because adopting
//     it corpus-wide was measured and REFUTED: the face that offered nil then bounds a different
//     REGION than its own loop walk was built for, and simple/T3's blend torus goes
//     2827.227365 → 13816.882599 (4.9×, mesh area at PropertyQuality) with S9/T1/T3/T4 all falling
//     out of OCCT's 1 % area gate. The residual belongs to the CONSUMER that drops the curve —
//     fix it there, as the last three slices did — not to an assembler-side override. The count is
//     the debt, and the corpus gate is shrink-only so it cannot grow back.
//   - two curves that agree within the model weld → not a disagreement at all; silent.

// CodeAssembleCurveConflict marks a welded edge two consumers offered geometrically DIFFERENT
// curves for. It is a Defect: the shipped geometry was chosen by build order, not by any rule.
const CodeAssembleCurveConflict diag.Code = "assemble.edge-curve-conflict"

// CodeAssembleCurveNilOffer marks a welded edge where one consumer offered a curve and the other
// offered nil. The curve wins; the count is the outstanding debt of consumers that drop a curve
// they should carry.
const CodeAssembleCurveNilOffer diag.Code = "assemble.edge-curve-nil-offer"

// curveAgreementStations is how many interior stations the two offers are compared at. Both curves
// span the SAME welded vertex pair by construction (that pair is the catalog key), so a
// matched-fraction comparison is a sound two-sided deviation: it cannot be passed by a sub-span
// (a shorter curve would weld to a different pair) and it is symmetric in the two offers.
const curveAgreementStations = 8

// curveOfferDeviation is the largest distance between the two offers at matched arc-fractions,
// after orienting the second to the first by its endpoints. Both curves are known to share the
// welded endpoints; this measures how differently they get from one to the other — which is what
// separates the SAME arc built two ways (≈0) from an arc and its COMPLEMENT (≈2R).
func curveOfferDeviation(kept, offered geom.Curve3) float64 {
	kLo, kHi := kept.Domain()
	oLo, oHi := offered.Domain()
	if reversedOffer(kept, offered) {
		oLo, oHi = oHi, oLo
	}
	worst := 0.0
	for i := 1; i < curveAgreementStations; i++ {
		f := float64(i) / curveAgreementStations
		d := float64(kept.PointAt(kLo + f*(kHi-kLo)).DistanceTo(offered.PointAt(oLo + f*(oHi-oLo))))
		if d > worst {
			worst = d
		}
	}
	return worst
}

// reversedOffer reports whether offered runs end→start relative to kept, decided by which pairing
// of the two curves' endpoints is closer (they weld to the same vertex pair, so one pairing is
// exact and the other is a chord apart — except on a closed seam, where both are equal and the
// answer is "forward", the orientation the catalog's manifold-parity rule already imposes).
func reversedOffer(kept, offered geom.Curve3) bool {
	kLo, kHi := kept.Domain()
	oLo, oHi := offered.Domain()
	k0, k1 := kept.PointAt(kLo), kept.PointAt(kHi)
	o0, o1 := offered.PointAt(oLo), offered.PointAt(oHi)
	fwd := float64(k0.DistanceTo(o0)) + float64(k1.DistanceTo(o1))
	rev := float64(k0.DistanceTo(o1)) + float64(k1.DistanceTo(o0))
	return rev < fwd
}

// recordCurveOffer classifies a SECOND consumer's offer on an already-built shared edge and records
// it on the builder, so the report travels with the body (topo.Body.BuildDiagnostics) rather than
// being swallowed here. It never changes the edge: the first offer stands, deterministically, and
// the record is what makes the choice reviewable instead of invisible.
func (c *edgeCatalog) recordCurveOffer(a, b int, rec edgeRec, offered geom.Curve3) {
	switch {
	case rec.curve == nil && offered == nil:
		return // both consumers say "straight": no disagreement
	case rec.curve == nil:
		c.diagnoseNilOffer(a, b, offered, true)
	case offered == nil:
		c.diagnoseNilOffer(a, b, rec.curve, false)
	default:
		if dev := curveOfferDeviation(rec.curve, offered); dev > c.weld {
			c.diagnoseCurveConflict(a, b, rec.curve, offered, dev)
		}
	}
}

// diagnoseNilOffer records a nil-vs-curve offer: the welded pair and its endpoints, the offered
// curve (with its OWN endpoints and, for an arc, its sweep), and how far that curve departs from
// the straight chord the nil would ship — the quantity that says whether the disagreement matters.
// second reports whether the curve arrived after the nil (the order that would discard it).
func (c *edgeCatalog) diagnoseNilOffer(a, b int, curve geom.Curve3, second bool) {
	c.bld.Diagnose(diag.Diagnostic{
		Code: CodeAssembleCurveNilOffer, Severity: diag.Warning,
		Detail: fmt.Sprintf("welded pair (%d,%d) %v→%v: one consumer offered %s, the other nil "+
			"(curve offered second=%v); it departs the straight chord by %.6g (weld %.6g)",
			a, b, c.verts[a], c.verts[b], describeOfferedCurve(curve), second,
			curveChordDeparture(curve), c.weld),
	})
}

// curveChordDeparture is how far the curve gets from the straight chord between its own endpoints —
// 0 for a curve that is effectively the chord, ~R for an arc, ~2R for a full circle.
func curveChordDeparture(c geom.Curve3) float64 {
	lo, hi := c.Domain()
	chord := geom.NewLineSegment(c.PointAt(lo), c.PointAt(hi))
	worst := 0.0
	for i := 1; i < curveAgreementStations; i++ {
		f := float64(i) / curveAgreementStations
		d := float64(c.PointAt(lo + f*(hi-lo)).DistanceTo(chord.PointAt(f)))
		if d > worst {
			worst = d
		}
	}
	return worst
}

// diagnoseCurveConflict records two geometrically different curves offered for one welded edge —
// the offending pair, its endpoints, BOTH curves and the deviation, so the reader can reproduce it.
func (c *edgeCatalog) diagnoseCurveConflict(a, b int, kept, offered geom.Curve3, dev float64) {
	c.bld.Diagnose(diag.Diagnostic{
		Code: CodeAssembleCurveConflict, Severity: diag.Defect,
		Detail: fmt.Sprintf("welded pair (%d,%d) %v→%v: two consumers offered curves %.6g apart (weld %.6g); "+
			"kept %s, discarded %s — the shipped geometry would be decided by build order, not by a rule",
			a, b, c.verts[a], c.verts[b], dev, c.weld, describeOfferedCurve(kept), describeOfferedCurve(offered)),
	})
}

// describeOfferedCurve renders a curve for a conflict message: its type, its endpoints, and — for a
// circular arc, the family this defect lives in — its centre, radius and SWEPT ANGLE, which is the
// one quantity that separates an arc from its complement.
func describeOfferedCurve(c geom.Curve3) string {
	lo, hi := c.Domain()
	if arc, ok := c.(geom.Arc3d); ok {
		return fmt.Sprintf("Arc3d(centre %v r=%.9f sweep=%.9frad %v→%v)",
			arc.Center, arc.Radius, arc.SweepAngle, c.PointAt(lo), c.PointAt(hi))
	}
	return fmt.Sprintf("%T(%v→%v)", c, c.PointAt(lo), c.PointAt(hi))
}
