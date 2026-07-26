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

// knownEdgeSpanDebt is the FULL remaining population of cases whose shipped body still carries an edge whose
// curve disagrees with its bounding vertices, each capped at 1.1× its measured value. It was 17 cases before
// alignCarriedArcsToSegments and 5 after; the ELLIPTIC survivor-rim carry
// (fillet_survivor_rim_ellipse.go) retires F6 outright and shrinks complex/F2 6× — 5 → 4 entries:
//
//   - complex/F2 (0.490 → 0.0809): its FOUR geom.EllipticalArc cap rims were each carried whole and
//     backwards (the gap equalled the chord between the curve's own endpoints exactly: 86.3707, 14.09,
//     10.91), because the parent's retained span could only be re-derived for a geom.Arc3d. All four are
//     now rebuilt from the parent's own eccentric angles, in the LOOP's traversal order. What remains is a
//     geom.BSplineCurve rail with the same disagreement (14.2602) — a different curve family, and one with
//     no closed-form sub-span, so it needs the fitted-rail re-parameterisation, not this fix.
//   - M4 (0.100), N9 (0.049) and N3 (0.031) carry an Arc3d on a loop the specialized obstacle / concave-arm
//     rebuild paths assemble, which never reach transformLoop's carried-arc pass.
//
// The table must SHRINK, never widen (ellipse-carry-report.md §7).
func knownEdgeSpanDebt() []offSurfaceDebtEntry {
	return []offSurfaceDebtEntry{
		{"F2", "complex", 0.089}, {"M4", "simple", 0.11},
		{"N9", "simple", 0.0542}, {"N3", "simple", 0.034},
	}
}

// TestEdgeSpanDebtIsWellFormed keeps that table honest: no duplicate key, and every ceiling LOOSER than the
// default budget (an entry at or below it would be dead weight hiding nothing).
func TestEdgeSpanDebtIsWellFormed(t *testing.T) {
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
