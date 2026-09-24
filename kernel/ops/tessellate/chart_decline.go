// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
)

// The ONE report for a chart-driven mesher decline (Oblikovati/Oblikovati#3520).
//
// chartFaceMesh has three routes into it — the classification's kindChart arm, chartedTrimMesh, and
// singlyPeriodicWrapMesh — and reporting had grown separately in each. chartedTrimMesh spoke through
// its own fallback's code (and named the wrong mesher doing it, see namedRefusal); the kindChart arm
// said nothing whatever; and singlyPeriodicWrapMesh spoke only when isDevelopableSide(s) held AND the
// outer loop wrapped the whole period.
//
// That last condition was blind to more than the issue's wording suggests. recordUnmeshedWallWrap
// returns early for anything that is NOT a developable side, and every SPHERE reaches this call
// (IsPeriodic(U) != IsPeriodic(V)), so the silent set was: a developable whose loop does not wrap, AND
// every charted sphere giving up there. Both fell to the flat patch CDT with no defect at all.
//
// The ground rule broken is "never degrade silently", and the fix it asks for is structural rather than
// a fourth condition: a caller that has to remember to report will one day forget, and two of this
// wave's later tasks (#3517, #3518) DELETE mesher arms whose faces then fall into these same routes.
//
// So the reporting is taken away from the callers. chartFaceMesh itself records, unconditionally, into
// a *chartDeclineLog it cannot be called without, and the log is owned and stamped by
// tessellateCurvedFace — the one router every route runs under. A caller that takes the router's log
// therefore cannot forget the report and cannot condition it, because it never sees it.
//
// Go does NOT enforce that, and it is worth being exact about which half is which. The parameter makes
// forgetting visible; it does not make it impossible. chartDeclineLog is an ordinary package-level
// struct with a usable zero value, so a new file here can write chartFaceMesh(f, s, q,
// &chartDeclineLog{}), drop the answer and fall back in silence — that compiles, and it is precisely
// what a reviewer built against the first cut of this change.
//
// What forbids it is archguard's TestOnlyTheCurvedFaceRouterOwnsAChartDeclineLog, which fails three
// ways: a second production file CONSTRUCTING a log, the owner no longer constructing one, and the
// owner no longer STAMPING it with recordOn. Construction is matched in all three shapes Go offers —
// a composite literal, new(chartDeclineLog), and a var declaration — because the first cut matched only
// the brace form and a reviewer walked past it with new(). A *chartDeclineLog PARAMETER is deliberately
// not a construction; taking the router's log is how every legitimate caller works. Each of those five
// facts is planted by a probe in TestTheOwnershipGuardBitesEveryConstructionForm. Trust the guard, not
// the compiler.
//
// A decline is recorded only when the mesher OWNED the face: chartFaceMesh answers false both for "the
// chart said give this up" and for "this was never my face" (no chart recorded, or a surface that wraps
// in neither direction), and only the first is a degradation. The second is the ordinary route onto the
// generic (u,v) trim path and reporting it would cry on every healthy analytic face there is.
//
// THE MEASUREMENT. This block is the one receipt for this code; archguard's kernelNetDeltaPin cites it
// rather than restating it, because the first cut of #3520 carried two contradictory sets of numbers in
// one commit. Re-run 2026-09-09 on m48/chart-decline-recorder over the sweep described here.
//
// Bodies: 128 attempted (72 crossing-rod, 48 rod ∪/−/∩ ball, 8 ring cut by a half space), built through
// **ops.Boolean** — the general pipeline — of which 106 BUILT: 22 were refused by its own acceptance
// gate ("the exact result failed its own acceptance gate") and not scanned. Each built body harvested
// through query.BodyMeshDiagnostics at two facetings, so 212 body-faceting pairs.
//
// The entry point is part of the number, not a footnote. brep.Boolean admits the 22 that ops.Boolean
// refuses, and on the crossing-rod third alone that turns 28 hits and ONE at chord 0.05 into 40 hits and
// SEVEN — because six of the extra display-faceting hits sit on bodies the general pipeline never
// ships. A reader reproducing this through the other entry point gets a different answer and concludes
// the receipt is wrong, so: ops.Boolean.
//
// (The corpus rows in kernel/ops/query build their two bodies with brep.Boolean instead, and must:
// kernel/ops depends on kernel/ops/query, so an internal query test cannot import ops without a cycle.
// Both row bodies are in the ops.Boolean set too — they are hits under both entry points — so the rows
// assert on bodies the general pipeline really produces.)
//
// Hits: 31. Thirty at chord 0.001 (PropertyQuality) and ONE at chord 0.05 (DefaultQuality) — the
// faceting model/feature/result_diagnostics.go harvests into feature health, so this code does reach a
// user. By what else the body carried: 22 already raised CodeWallWrapUnmeshed, three
// CodeTrimIgnoredFullDomain, and SIX raised neither — those six are the case nothing named before.
//
// Cost, per FACE, against the analytic area oracle (query.AnalyticFaceArea), by where the router fell
// through to. It is not a small range and it does not follow the decline:
//
//	flat patch CDT     −28.69 %   r = 3 ∩ r = 3.5 offset 0.5, face 0 at chord 0.05 (69.390 vs 97.309)
//	flat patch CDT     −23.97 %   r = 3 ∪ r = 2 offset 2, face 4 at chord 0.001 (147.038 vs 193.394)
//	whole domain      +119.41 %   rod ∩ ball r = 2.5 offset 1, its sphere face (78.530 vs 35.791)
//	generic (u,v)      −0.0056 %  r = 3 ∩ r = 2.5 offset 1, face 0 (49.9685 vs 49.9713), which is what
//	                              its UNDECLINED neighbour on the same body came back at (−0.0062 %)
//
// RE-MEASURED AFTER #3518, and the numbers above are a snapshot of the base they were taken on. The
// boundary-side classification (chart_face_rim_side.go) removes a whole family of declines, so the
// hit count here is now stale in a direction worth naming. Swept over a family this block's sweep
// contains — the r = 3 rod along +x against a second rod and against a ball, radii 1.5 … 3.5, offsets
// 0 … 2, all three operators, both facetings, 150 bodies through brep.Boolean:
//
//	                     chart-mesher declines    of which nothing else named
//	base fff94140                           44                             8
//	after #3518                             20                             4
//
// Every rod-ROD silent decline is gone (4 → 0), which is why the corpus row below moved to the
// rod-ball third. The 128-body sweep above was not re-run; whoever needs its exact hit count should
// re-run it rather than scale these.
//
// The first row is the one that matters most for reading this code: the single DefaultQuality hit is a
// face 28.7 % short of its area, so the Defect at the display faceting is EARNED, not noise. (That body
// was already reporting CodeWallWrapUnmeshed and a torn mesh there; what this code adds is the cause.)
// And the spread is why the decline is reported rather than graded where it happens: the mesher that
// gave up returns nil and the router has not yet fallen through, so nothing at the point of decline can
// know whether the face will lose a quarter of its area or none of it.

// CodeChartMesherDeclined marks a face that CARRIES a chart on a periodic surface — exactly the face
// chartFaceMesh is the general mesher for (ADR-0061/ADR-0063) — which that mesher gave up, so the face
// shipped from whatever the curved-face router fell through to: the generic (u,v) trim path, the flat
// best-fit-plane CDT, or the surface's whole parametric domain.
//
// None of those was certified against the face's chart, so the covering the face ships is one nobody
// checked describes its region. The mesh still ships, because a covering nobody certified beats a
// missing face in a viewport, but the fallback reaches feature health, the API and the UI rather than
// being swallowed.
const CodeChartMesherDeclined diag.Code = "tessellate.chart-mesher-declined"

// chartDeclineLog is the router's slot for the reason the chart-driven mesher gave up a face it owns.
// The zero value is "nothing declined"; tessellateCurvedFace owns the only instance.
type chartDeclineLog struct {
	why string
}

// declined records why the chart mesher gave up a face it owns. The FIRST reason wins: a face can reach
// chartFaceMesh twice (the kindChart arm, then chartedTrimMesh after ToUVLoops fails) and the second
// call re-derives the same answer, so the later one adds nothing.
func (l *chartDeclineLog) declined(why string) {
	if l.why == "" {
		l.why = why
	}
}

// reason is why the chart mesher gave the face up, or "" when it never did. The full-domain reporter
// reads it so it can name the mesher that actually refused instead of saying nothing recognised the
// face (#3520 review I5).
func (l *chartDeclineLog) reason() string { return l.why }

// recordOn stamps a pending decline onto the mesh the face shipped instead, and is the only place the
// code is raised.
//
// A face that declined and then meshed NOTHING is the worst outcome there is — no geometry and no
// report — so the record does not depend on the fall-through having produced a mesh. The invariant is
// "a decline is always reported", not "reported as long as something else succeeded" (#3520 review M3).
//
// I found no fall-through in this router that returns nil — fullDomainGridMesh, trimmedPatchMesh and the
// generic (u,v) path all end at patchMeshFrom or a grid builder, each of which starts from &Mesh{} — but
// that is a reading of the arms, not a probe, so the branch is written as live code rather than
// documented away. If it does fire, the face ships a non-nil mesh with zero triangles carrying one
// Defect: MergeMesh contributes nothing from it, so no body's geometry changes, and
// query.BodyMeshDiagnostics harvests it (it skips nil meshes and reads the Diagnostics of every other),
// which is the whole point. Without a pending decline nothing is allocated and the ordinary face keeps
// whatever the router returned, nil included.
func (l *chartDeclineLog) recordOn(m *Mesh, s geom.Surface) *Mesh {
	if l.why == "" {
		return m
	}
	if m == nil {
		m = &Mesh{}
	}
	m.Diagnose(diag.Diagnostic{
		Code:     CodeChartMesherDeclined,
		Severity: diag.Defect,
		Detail: fmt.Sprintf("the chart-driven mesher was selected for a charted %T face and gave it up "+
			"— %s — so the face shipped from the router's fall-through path, over a region its chart "+
			"never certified", s, l.why),
	})
	return m
}
