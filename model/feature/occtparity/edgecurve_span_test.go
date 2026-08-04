// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/topo"
)

// edgeSpanTol is the budget for "an edge's curve runs FROM its start vertex TO its end vertex", relative to
// the result body's bounding diagonal (scale-invariant, ADR-0042). A correctly built edge holds this to
// 1e-16..1e-14 relative. It catches two distinct construction defects: a carried rim arc sweeping PAST the
// loop's vertex (0.023–0.19), and a carried rim whose curve runs BACKWARDS with respect to its vertices
// (the gap is then the chord between the curve's own endpoints — F6's whole 300-long cap rim). 1e-9
// therefore separates construction from arithmetic with no tolerance to tune.
const edgeSpanTol = 1e-9

// TestEveryEdgeCurveSpansItsVertices is the corpus-wide invariant that an edge's GEOMETRY agrees with its
// TOPOLOGY: the curve's domain endpoints must be its bounding vertices.
//
// Violating it is silent — the body still welds and validates, because the weld uses the vertices — but the
// tessellator parameterises the FACE from the curve, so a curve that overshoots its vertex tiles a boundary
// past the loop's own corner. That is what put 3 fold edges in E1's meridian plane, 1 in N5's set-back seam,
// and −45.3% on D7's host sphere zone (which had been read as a sphere-zone mesher defect). The `default`
// survivor branch of transformLoop carried an untouched rim arc whole even when the segment's other end had
// been pulled back to a fillet tangent point; alignCarriedArcsToSegments now re-trims it.
//
// Cases still carrying an overshoot are listed EXPLICITLY in knownEdgeSpanDebt, so this is a RATCHET: a
// listed case may improve freely, growing past its ceiling — or ANY unlisted case exceeding edgeSpanTol —
// fails loud. The table must shrink, never widen.
func TestEveryEdgeCurveSpansItsVertices(t *testing.T) {
	t.Parallel()
	dir := CorpusFixtureDir()
	debt := edgeSpanDebtIndex()
	for _, r := range Corpus() {
		body, ok := shippedCaseBody(r, dir)
		if !ok {
			continue
		}
		assertEdgeCurvesSpanTheirVertices(t, r, body, debt[r.Grid+"/"+r.Case])
	}
}

// assertEdgeCurvesSpanTheirVertices fails when the body's worst curve-domain-vs-vertex gap, relative to the
// bounding diagonal, exceeds the case's budget (its debt ceiling, else the default).
func assertEdgeCurvesSpanTheirVertices(t *testing.T, r Record, body *topo.Body, budget float64) {
	t.Helper()
	if budget == 0 {
		budget = edgeSpanTol
	}
	worst, where := worstEdgeSpanGap(body)
	if rel := worst / boundingDiag(body); rel > budget {
		t.Errorf("%s/%s: edge curve disagrees with its own vertices by %.6g (rel %.4g of the %.4g diagonal, budget %.4g) — %s",
			r.Grid, r.Case, worst, rel, boundingDiag(body), budget, where)
	}
}

// worstEdgeSpanGap returns the largest distance from a curve's domain endpoint to the vertex that bounds it,
// over every DISTINCT edge of the body, plus a description of the offender. It is the same measure whether
// the curve overshoots its vertex or runs backwards between the two.
func worstEdgeSpanGap(b *topo.Body) (float64, string) {
	worst, where := 0.0, "(none)"
	seen := map[uint64]bool{}
	for _, f := range b.Faces() {
		for _, e := range boundaryEdgesOf(f) {
			if seen[e.ID()] {
				continue
			}
			seen[e.ID()] = true
			if d := edgeSpanGap(e); d > worst {
				worst = d
				where = fmt.Sprintf("edge %d (%T)", e.ID(), e.Geometry())
			}
		}
	}
	return worst, where
}

// edgeSpanGap is the worse of an edge's two curve-end-to-vertex distances.
func edgeSpanGap(e *topo.Edge) float64 {
	c := e.Geometry()
	if c == nil {
		return 0
	}
	lo, hi := c.Domain()
	worst := 0.0
	for _, pair := range []struct {
		t float64
		v *topo.Vertex
	}{{lo, e.StartVertex()}, {hi, e.EndVertex()}} {
		if pair.v == nil {
			continue
		}
		if d := c.PointAt(pair.t).DistanceTo(pair.v.Point()); d > worst {
			worst = d
		}
	}
	return worst
}

// edgeSpanDebtIndex keys knownEdgeSpanDebt by "grid/case" for O(1) lookup.
func edgeSpanDebtIndex() map[string]float64 {
	out := make(map[string]float64, len(knownEdgeSpanDebt()))
	for _, d := range knownEdgeSpanDebt() {
		out[d.grid+"/"+d.name] = d.rel
	}
	return out
}

// knownEdgeSpanDebt is down to ONE case. It was 17 before alignCarriedArcsToSegments, 5 after, then 4 once
// the ELLIPTIC survivor-rim carry (fillet_survivor_rim_ellipse.go) retired F6 and shrank complex/F2 6×.
//
// Three whole cases — simple/M4, simple/N9, simple/N3 — plus the LARGER half of complex/F2's residual were
// one root, and it was a curve running BACKWARDS between its own vertices, not an overshoot
// (planar-retrim-selfcross-report.md). Measured worst gaps, absolute: simple/M4 17.3205 → 6.2e-15,
// simple/N9 8.16497 → 1.4e-14, simple/N3 6.32456 → 4.2e-12, complex/F2 14.2602 → 10.9131 (F2's remaining
// offender is a different curve family, below).
//
//   - the CONCAVE-ARM CAP RETRIM (matchArcFeet) may pair a far cross-section arc's two feet onto the
//     flanking cap edges the other way round; it used to hand back the swapped POINTS while the caller
//     re-wrapped the arc's ORIGINAL curve, so the spliced segment ran to→from. It now returns the arc
//     reversed (reversedEndSeg). That retires M4, N9 and N3, whose loops the concave-arm rebuild path
//     assembles and which therefore never reach transformLoop's carried-arc pass.
//   - transformLoop's `default` SURVIVOR arm carried a non-arc curved survivor unchanged, so a REVERSED
//     use of an OPEN geom.BSplineCurve retrim curve ran end→start — complex/F2's 14.2602. The Arc3d arm
//     had always flipped its sweep; the non-arc arm now does too (orientedOpenSurvivor).
//
// What remains on complex/F2 (0.089 → 0.068, a 1.31× tightening) is an OVERSHOOT, not a reversal: a
// geom.EllipticalArc carried 10.9131 past its own loop vertex (rel 0.0618696, capped at 1.1× as the rest of
// the table is). It needs the ellipse-aware retained-span algebra, not an orientation fix — the same
// algebra that is why orientedOpenSurvivor deliberately leaves the elliptic family untouched.
//
// The table must SHRINK, never widen (ellipse-carry-report.md §7).
func knownEdgeSpanDebt() []offSurfaceDebtEntry {
	return []offSurfaceDebtEntry{
		{"F2", "complex", 0.068}, // 1.1 x the measured 0.0618696
	}
}

// TestEdgeSpanDebtIsWellFormed keeps that table honest: no duplicate key, and every ceiling LOOSER than the
// default budget (an entry at or below it would be dead weight hiding nothing).
func TestEdgeSpanDebtIsWellFormed(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for _, d := range knownEdgeSpanDebt() {
		key := d.grid + "/" + d.name
		if seen[key] {
			t.Errorf("duplicate edge-span debt entry %s", key)
		}
		seen[key] = true
		if d.rel <= edgeSpanTol {
			t.Errorf("%s debt ceiling %g is within the default budget %g — delete the entry instead", key, d.rel, edgeSpanTol)
		}
	}
}
