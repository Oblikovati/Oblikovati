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
//   - nil then a curve → the curve is ADOPTED onto the already-built edge, reversed when this
//     consumer traverses the edge opposite the sense it was stored in (rec.from != a — the same
//     rule `use` already applies to the co-edge's Reversed flag). A nil is an absence of
//     information, not an assertion of straightness, so the consumer that HAS geometry supplies it
//     and the straight chord goes away. ★ The orientation is the whole of it: adopting the curve
//     in the offering consumer's own direction, on an edge stored the other way, moves the region
//     the receiving face's loop walk bounds and inflates it (simple/T3's blend torus 2827.227365 →
//     13816.882599 mesh area at PropertyQuality, saturating the 262144-triangle cell cap, and the
//     corpus rollup 114 → 110 — measured here, and reproducible by deleting the reversal). Adopted
//     the right way round the same face reads 2826.791716 against live DRAWEXE's 2826.04, a 1.58×
//     improvement, and the rollup is unchanged. The nil offer is still recorded as Warning debt:
//     the consumer that declined should carry its own boundary, and the corpus gate holds that
//     population shrink-only.
//   - a curve then nil → the curve already stands (it was the first writer); same Warning debt.
//   - two curves that disagree geometrically → a real conflict. The assembler cannot know which
//     one is right, so it must not silently arbitrate: it keeps the first (so the build stays
//     deterministic) and records a Defect naming the welded pair, its endpoints, BOTH curves and
//     the measured deviation. `simple/M8` shipped exactly one of these — a 255.52° boss rim arc
//     against its own 104.48° COMPLEMENT, 22.01 apart, decided only by the plane being built
//     before the cylinder (fixed at source in rebuildArcSeg). The standing corpus gate holds this
//     class at ZERO.
//   - two offers within the model weld of each other → not a disagreement at all; silent. This
//     applies to BOTH nil branches — nil-then-curve AND curve-then-nil: a curve that departs its
//     own chord by less than the weld IS the chord, so it is neither adopted (which would churn
//     bytes for nothing) nor recorded (which would report float noise as consumer-side debt — 90
//     of the corpus's 362 records were offers 4.4e-16…2.0e-15 off their own chord, three orders
//     below any model weld here). The curve-then-nil arm first shipped WITHOUT the threshold while
//     this docstring already claimed it (adversarial-review finding m-1); it was latent — the
//     corpus's only curve-first records are complex/F2's two, 2.92426 off their chord against a
//     1.76388e-07 weld, which the threshold keeps — and is now the same rule, gated both ways by
//     TestCurveFirstKeepUsesTheSameWeldThreshold.

// CodeAssembleCurveConflict marks a welded edge two consumers offered geometrically DIFFERENT
// curves for. It is a Defect: the shipped geometry was chosen by build order, not by any rule.
const CodeAssembleCurveConflict diag.Code = "assemble.edge-curve-conflict"

// CodeAssembleCurveNilOffer marks a welded edge where one consumer offered a curve and the other
// offered nil. The curve wins — adopted if it arrived second. The count is the outstanding debt of
// consumers that decline to carry a boundary they share with a curved neighbour.
const CodeAssembleCurveNilOffer diag.Code = "assemble.edge-curve-nil-offer"

// CodeAssembleEdgeCatalog marks a body as assembled THROUGH this catalog, carrying the number of
// shared edges it welded. It is what keeps the catalog's standing corpus gate from being vacuous:
// a body's build report is dropped by every rebuild (topo.Body.BuildDiagnostics), and an empty
// report is otherwise indistinguishable from "this body never went through the catalog at all" —
// which is true of 22 of the 125 shipped corpus bodies, because fillet.go's three early returns
// (FilletCylinderRim, FilletCylinderArc, filletTangentStripe) assemble on their own topo.Builder.
const CodeAssembleEdgeCatalog diag.Code = "assemble.edge-catalog"

// curveAgreementStations is how many interior stations the two offers are compared at.
const curveAgreementStations = 8

// curveOfferDeviation is the largest distance between the two offers at matched DOMAIN fractions,
// minimised over the two traversal senses. Both curves are known to share the welded endpoints;
// this measures how differently they get from one to the other — which is what separates the SAME
// arc built two ways (≈0) from an arc and its COMPLEMENT (≈2R·sin(half the minor span); 22.01 on
// simple/M8's r=25 rim).
//
// Two caveats a reader must know, because this feeds a HARD-ZERO gate:
//
//   - ★ matched DOMAIN fractions, not matched arc-length. For geom.Arc3d and geom.LineSegment —
//     the only shapes this branch sees today — the two coincide exactly, because both are
//     parameterised affinely in arc length. For a BSpline rail or a rational conic they do not, so
//     two parameterisations of the SAME geometry could read a nonzero deviation and be reported as
//     a conflict. That would be a false RED, never a false green.
//   - ★ the minimum over both senses, rather than picking a sense by an endpoint tie-break. On a
//     CLOSED seam (both endpoints welded to one vertex) the endpoints carry no information at all
//     about which way either offer runs, so a tie-break there decides a hard-zero gate on the
//     ordering of two nearly-equal floating-point sums. Taking the better of the two senses cannot
//     invent a conflict and cannot hide one: an offer that matches `kept` reversed IS the same
//     geometry, and orientation is the catalog's job, not a disagreement.
func curveOfferDeviation(kept, offered geom.Curve3) float64 {
	fwd := sampledCurveGap(kept, offered, false)
	rev := sampledCurveGap(kept, offered, true)
	if rev < fwd {
		return rev
	}
	return fwd
}

// sampledCurveGap is the worst distance between kept and offered at matched domain fractions, with
// offered walked backwards when reversed.
func sampledCurveGap(kept, offered geom.Curve3, reversed bool) float64 {
	kLo, kHi := kept.Domain()
	oLo, oHi := offered.Domain()
	if reversed {
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

// resolveCurveOffer settles a SECOND consumer's offer on an already-built shared edge: it adopts
// real geometry the edge does not yet carry, refuses to arbitrate a genuine conflict, and records
// what it did on the builder so the report travels with the body (topo.Body.BuildDiagnostics).
func (c *edgeCatalog) resolveCurveOffer(key seamEdgeKey, a, b int, rec edgeRec, offered geom.Curve3) {
	switch {
	case rec.curve == nil && offered == nil:
		return // both consumers say "straight": no disagreement
	case rec.curve == nil:
		c.adoptOfferedCurve(key, a, b, rec, offered)
	case offered == nil:
		// The same weld threshold adoptOfferedCurve applies (file rule above): a kept curve within
		// the weld of its own chord IS the chord, so the later nil offer AGREES with it — silence.
		if curveChordDeparture(rec.curve) > c.weld {
			c.diagnoseNilOffer(a, b, rec.curve, "kept (it was the first offer)")
		}
	default:
		if dev := curveOfferDeviation(rec.curve, offered); dev > c.weld {
			c.diagnoseCurveConflict(a, b, rec.curve, offered, dev)
		}
	}
}

// adoptOfferedCurve replaces the straight chord on an already-built edge with the curve its second
// consumer offers, oriented to the sense the edge was STORED in: rec.from != a means this consumer
// walks the edge backwards, so its curve must be reversed before it can become the edge's geometry.
//
// It declines in exactly two cases, both of which are a nil that MEANS something:
//
//   - a departure within the model weld — the offer IS the chord already on the edge (90 of the
//     corpus's 362 recorded offers departed by 4.4e-16…2.0e-15, below every model weld here);
//   - a==b, a pair welded to ONE vertex. That is the shape `subdividedSurvivorCurve` deliberately
//     drops a closed-conic rim to nil at: the edge is a zero-length degeneracy re-traced as chords
//     by its inserts, and hanging the full circle on it would make that one edge tessellate the
//     WHOLE circle and self-cross the loop. A deliberate decline is therefore distinguishable by
//     SHAPE, and needs no new flag on the offer.
//
// (`onRimParam`'s decline — U4's coupled node, 4.04e-03 off its own rim — is the other deliberate
// one, and it needs nothing here: it is a single shared computation read by ALL consumers of the
// segment (rimNodeTrimsOf, rimSubArcBetween), so it hands every one of them nil and never presents
// as a nil-vs-curve disagreement. Measured: no case carrying that decline carries this debt.)
func (c *edgeCatalog) adoptOfferedCurve(key seamEdgeKey, a, b int, rec edgeRec, offered geom.Curve3) {
	if curveChordDeparture(offered) <= c.weld {
		return // the offer is the chord the edge already carries
	}
	if a == b {
		c.diagnoseNilOffer(a, b, offered, "DECLINED: the pair welds to one vertex, a nil there is a deliberate drop")
		return
	}
	oriented := offered
	if rec.from != a {
		oriented = geom.ReverseCurve3(offered)
	}
	c.bld.ReplaceEdgeCurve(rec.edge, oriented)
	rec.curve = oriented
	c.edges[key] = rec
	c.diagnoseNilOffer(a, b, offered, fmt.Sprintf("ADOPTED (reversed=%v)", rec.from != a))
}

// diagnoseNilOffer records a nil-vs-curve offer: the welded pair and its endpoints, the offered
// curve (with its OWN endpoints and, for an arc, its sweep), how far that curve departs the
// straight chord — the quantity that says whether the disagreement matters — and the disposition.
func (c *edgeCatalog) diagnoseNilOffer(a, b int, curve geom.Curve3, disposition string) {
	c.bld.Diagnose(diag.Diagnostic{
		Code: CodeAssembleCurveNilOffer, Severity: diag.Warning,
		Detail: fmt.Sprintf("welded pair (%d,%d) %v→%v: one consumer offered %s, the other nil; "+
			"it departs the straight chord by %.6g (weld %.6g) — %s",
			a, b, c.verts[a], c.verts[b], describeOfferedCurve(curve),
			curveChordDeparture(curve), c.weld, disposition),
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

// diagnoseCatalogUse stamps the body with the fact that THIS catalog assembled it, and with how
// many shared edges it welded — the positive marker a corpus gate needs to tell "the catalog
// reported nothing" from "the catalog never ran" (see CodeAssembleEdgeCatalog).
func (c *edgeCatalog) diagnoseCatalogUse() {
	c.bld.Diagnose(diag.Diagnostic{
		Code: CodeAssembleEdgeCatalog, Severity: diag.Info,
		Detail: fmt.Sprintf("assembled through the fillet edge catalog: %d welded shared edges (weld %.6g)",
			len(c.edges), c.weld),
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
