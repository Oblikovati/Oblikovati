// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"fmt"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// loopSegOnFaceTol is the default budget for "every loop segment lies on the face it bounds",
// expressed RELATIVE to the result body's bounding diagonal so it is scale-invariant (a tscale
// SCALE=1000 fixture gets the same budget as its unit twin, ADR-0042). 1e-6 of the diagonal is
// ~4 decades above the construction noise a correctly-built rail carries (analytic arcs land at
// 1e-16..1e-13 relative; a canal loft's own approximation at ~1e-10), so it separates a
// CONSTRUCTION defect from arithmetic without being a tolerance to tune.
const loopSegOnFaceTol = 1e-6

// TestEveryLoopSegmentLiesOnItsFace is the corpus-wide invariant guard: on the SHIPPED body of every
// scored case, every boundary edge of every face must have its CURVE on that face's own surface.
//
// This is the invariant a whole class of defects violates silently — a face whose boundary is not on
// its own surface still validates as a watertight solid (the vertices weld, the loops close), so the
// scoreboard cannot see it, but the tessellator then tiles a boundary that is not on the surface and
// every downstream consumer inherits the error. B5/C4/D7 shipped an arc 12.36 (94% of the fillet
// radius) off its own cylinder while two of them were merely FAIL(area) and L3 was fully GREEN.
//
// Cases still carrying a residual are listed EXPLICITLY in knownOffSurfaceDebt with the measured
// ceiling, so the guard is a RATCHET, not a tolerance: a listed case may improve freely, but growing
// past its ceiling — or ANY unlisted case exceeding loopSegOnFaceTol — fails loud. Each debt entry is
// a real, separately-rooted defect (see offsurface-loopseg-report.md §2 for the taxonomy); the table
// shrinks as those roots are fixed and must never be widened to accommodate a regression.
func TestEveryLoopSegmentLiesOnItsFace(t *testing.T) {
	dir := CorpusFixtureDir()
	debt := offSurfaceDebtIndex()
	for _, r := range Corpus() {
		body, ok := shippedCaseBody(r, dir)
		if !ok {
			continue // skipped / faulty: no single healthy body to measure
		}
		assertLoopSegmentsOnFaces(t, r, body, debt[r.Grid+"/"+r.Case])
	}
}

// assertLoopSegmentsOnFaces fails when the body's worst boundary-edge-off-its-own-face residual, taken
// relative to the bounding diagonal, exceeds the case's budget (its debt ceiling, else the default).
func assertLoopSegmentsOnFaces(t *testing.T, r Record, body *topo.Body, budget float64) {
	t.Helper()
	if budget == 0 {
		budget = loopSegOnFaceTol
	}
	worst, where := worstLoopSegmentOffFace(body)
	if rel := worst / boundingDiag(body); rel > budget {
		t.Errorf("%s/%s: loop segment leaves the face it bounds by %.6g (rel %.4g of the %.4g diagonal, budget %.4g) — %s",
			r.Grid, r.Case, worst, rel, boundingDiag(body), budget, where)
	}
}

// shippedCaseBody runs one case's real fillet and returns its single result body, or ok=false when the
// case is skipped (TODO / variable-radius / quarantine / import divergence) or the fillet is unhealthy —
// a faulty result has no invariant to hold.
func shippedCaseBody(r Record, dir string) (*topo.Body, bool) {
	if _, decided := preRunOutcome(r); decided {
		return nil, false
	}
	body, err := importInput(filepath.Join(dir, r.InputStep))
	if err != nil {
		return nil, false
	}
	sets, ok := scoreLocate(r, body)
	if !ok {
		return nil, false
	}
	res, filletOK, _ := runFillet(body, sets)
	if !filletOK || len(res) != 1 || res[0] == nil {
		return nil, false
	}
	return res[0], true
}

// worstLoopSegmentOffFace returns the largest distance from any face's boundary-edge curve to that
// face's OWN surface, plus a description of the offender (face surface type, edge id, curve type).
func worstLoopSegmentOffFace(b *topo.Body) (float64, string) {
	worst, where := 0.0, "(none)"
	for _, f := range b.Faces() {
		s := f.Geometry()
		if s == nil {
			continue
		}
		for _, e := range boundaryEdgesOf(f) {
			d, ok := curveOffSurface(e.Geometry(), s)
			if !ok || d <= worst {
				continue
			}
			worst = d
			where = fmt.Sprintf("face %d (%T) bounded by edge %d (%T)", f.ID(), s, e.ID(), e.Geometry())
		}
	}
	return worst, where
}

// boundaryEdgesOf returns each DISTINCT edge in f's loops (a seam edge used twice is measured once).
func boundaryEdgesOf(f *topo.Face) []*topo.Edge {
	seen := map[uint64]bool{}
	var out []*topo.Edge
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			if e := u.Edge(); e != nil && !seen[e.ID()] {
				seen[e.ID()] = true
				out = append(out, e)
			}
		}
	}
	return out
}

// curveOffSurface returns the max distance from 17 evenly-spaced samples of c to its closest point on
// s. Sampling (rather than a closed-form residual) is what makes this uniform across every curve and
// surface pair the kernel builds; 17 stations resolve a quarter-arc's mid-span bulge to <0.2% of the
// sagitta, far finer than the defects it must catch.
func curveOffSurface(c geom.Curve3, s geom.Surface) (float64, bool) {
	if c == nil {
		return 0, false
	}
	lo, hi := c.Domain()
	const stations = 17
	worst := 0.0
	for i := 0; i <= stations; i++ {
		p := c.PointAt(lo + (hi-lo)*float64(i)/float64(stations))
		_, _, foot := geom.ClosestPointOnSurface(s, p)
		if d := p.DistanceTo(foot); d > worst {
			worst = d
		}
	}
	return worst, true
}

// offSurfaceDebtEntry is one case's measured off-surface ceiling, relative to its bounding diagonal.
type offSurfaceDebtEntry struct {
	name string
	grid string
	rel  float64
}

// offSurfaceDebtIndex keys knownOffSurfaceDebt by "grid/case" for O(1) lookup.
func offSurfaceDebtIndex() map[string]float64 {
	out := make(map[string]float64, len(knownOffSurfaceDebt()))
	for _, d := range knownOffSurfaceDebt() {
		out[d.grid+"/"+d.name] = d.rel
	}
	return out
}

// knownOffSurfaceDebt is the FULL population of cases whose shipped body still carries a boundary edge
// off its own face, each capped at 1.1× its measured residual (relative to the bounding diagonal) so a
// regression fires while an improvement is free. Derived by an instrumented corpus-wide sweep, not from
// any report. It was 52 entries; the far-end wall trim (fillet_farend_trim.go) retired 17 of them —
// A5 A6 A8 B1 B5 B9 C4 C7 D3 D7 E1 E2 F2 F6 I3 M2 Q5 are now CLEAN — and shrank 5 more (N5 0.0216→0.000129,
// V1 0.00451→0.000295, V3 0.00712→0.00199, V5 0.00145→0.000313, complex/D8 0.0461→0.037), and the elliptic
// survivor-rim carry (fillet_survivor_rim_ellipse.go) then retired F7 — 35 → 34 entries. The table must
// SHRINK, never widen. The roots that remain, largest first (ellipse-carry-report.md §2):
//
//   - CHORDED SEAM MERIDIAN on the rim-fillet host rebuild (J2 J4). Previously recorded here as an
//     "imported base-face loop the fillet never touched"; that attribution is FALSIFIED — both INPUT
//     bodies measure clean (J2 5.4e-13, J4 1.6e-13 absolute, i.e. 5.8e-15 / 2.2e-15 relative), so the
//     chord is fillet-created. J2's shipped hemisphere keeps its Arc3d SEAM meridian's two vertices but
//     ships the edge between them as a 90.38-long geom.LineSegment, 28.44 off the sphere (J4: 61.24 long,
//     10.43 off its torus). The parent IS a geom.Arc3d, so this is NOT the non-Arc3d carry above — it is
//     the rim-fillet host rebuild (parent `rimfillet:torus#0`), which does not reach transformLoop's
//     carried-arc passes at all. Now the largest fillet-owned off-surface root in the corpus.
//   - WRONG STOP FACE at a valence>3 vertex (complex/D8): endFaceAt picks the first face at the terminal
//     vertex that is neither A nor B, so the far-end trim lands the section on a plausible-but-wrong wall.
//     (F7's 0.29 entry, the ELLIPSE-rim chord, is retired: the elliptic survivor-rim carry
//     — fillet_survivor_rim_ellipse.go — now re-derives that rim from the parent's own eccentric angles,
//     taking F7's worst residual 89.4426 → 3.4e-14 and its per-face gross error vs DRAWEXE 40793 → 2.8.)
//   - SPREAD-FAN / oblique-plane chords (P8 P9 V8 V9 V1 V3 V5 X9 K7 L1 L7 N5): a run-out spread cap is
//     tiled as a chord fan, so each chord sits a sagitta off the curved wall it spans.
//   - CANAL / BSpline patch rails (C2 C6 C8 S1 S3 S4 S6 S7 S9 T1 T3 T4 T6 T7 U3 U4 X3 R9): the rail is
//     built on the exact rolling-ball envelope while the patch is a fitted BSpline, so rail and patch
//     agree only to the fit's own residual (reverse-segment-fix-report.md §6 concern 2).
func knownOffSurfaceDebt() []offSurfaceDebtEntry {
	return []offSurfaceDebtEntry{
		{"J2", "simple", 0.334}, {"J4", "simple", 0.165},
		{"F2", "complex", 0.0616}, {"D8", "complex", 0.037}, {"C2", "simple", 0.0192},
		{"V9", "simple", 0.017}, {"P9", "simple", 0.017}, {"X9", "simple", 0.00284},
		{"T7", "simple", 0.00216}, {"V3", "simple", 0.00199}, {"S9", "simple", 0.00199},
		{"T1", "simple", 0.00156}, {"C8", "simple", 0.00129}, {"S1", "simple", 0.00105},
		{"T4", "simple", 0.00104}, {"S4", "simple", 0.00102}, {"T3", "simple", 0.000794},
		{"S6", "simple", 0.000641}, {"P8", "simple", 0.00061}, {"V8", "simple", 0.00061},
		{"S7", "simple", 0.000499}, {"X3", "simple", 0.000333}, {"U3", "simple", 0.000323},
		{"V5", "simple", 0.000313}, {"V1", "simple", 0.000295}, {"U4", "simple", 0.000246},
		{"R9", "simple", 0.000236}, {"S3", "simple", 0.000233}, {"T6", "simple", 0.000231},
		{"K7", "simple", 0.000152}, {"L1", "simple", 0.000133}, {"N5", "simple", 0.000129},
		{"L7", "simple", 0.000128}, {"C6", "simple", 0.000121},
	}
}

// TestOffSurfaceDebtIsWellFormed keeps the debt table honest: no duplicate key, and every ceiling
// LOOSER than the default budget (an entry at or below it would be dead weight hiding nothing).
func TestOffSurfaceDebtIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range knownOffSurfaceDebt() {
		key := d.grid + "/" + d.name
		if seen[key] {
			t.Errorf("duplicate off-surface debt entry %s", key)
		}
		seen[key] = true
		if d.rel <= loopSegOnFaceTol {
			t.Errorf("%s debt ceiling %g is within the default budget %g — delete the entry instead", key, d.rel, loopSegOnFaceTol)
		}
	}
}
