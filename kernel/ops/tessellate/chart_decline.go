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
// its own fallback's code; singlyPeriodicWrapMesh spoke only when isDevelopableSide(s) held AND the
// wall wrapped, so a developable whose loop does not wrap the period fell to the flat patch CDT with no
// defect at all; and the kindChart arm said nothing whatever. The ground rule broken there is "never
// degrade silently", and the fix it asks for is structural rather than a fourth condition: a caller
// that has to remember to report will one day forget, and two of this wave's later tasks (#3517,
// #3518) DELETE mesher arms whose faces then fall into these same routes.
//
// So the reporting is taken away from the callers. chartFaceMesh itself records, unconditionally, into
// a log it cannot be called without, and the log is owned and stamped by tessellateCurvedFace — the one
// router every route runs under. A new caller cannot forget because it cannot obtain a log except from
// the router, and it cannot condition the record because it never sees it.
//
// A decline is recorded only when the mesher OWNED the face: chartFaceMesh answers false both for "the
// chart said give this up" and for "this was never my face" (no chart recorded, or a surface that wraps
// in neither direction), and only the first is a degradation. The second is the ordinary route onto the
// generic (u,v) trim path and reporting it would cry on every healthy analytic face there is.
//
// What the record costs the face depends on where the router then falls through to, and the range was
// measured rather than assumed. Over a sweep of 128 attempted boolean bodies (72 crossing-rod, 48
// rod ∪/−/∩ ball, 8 ring cut by a half space) at two facetings, it is raised 31 times: 30 at chord
// 0.001 and once at chord 0.05, the faceting feature health reads. Twenty-two of those bodies already
// carried CodeWallWrapUnmeshed and three CodeTrimIgnoredFullDomain; SIX carried neither, and those six
// are the case nothing named before.
//
// Per-face against the analytic area oracle (query.AnalyticFaceArea): where the fall-through was the
// FLAT patch CDT the face came back 23.97 % under its analytic area (crossing rods r = 3 and r = 2,
// offset 2, joined — 147.038 mm² against 193.394), and where it was the generic (u, v) trim path it came
// back 0.0056 % under (r = 3 ∩ r = 2.5, offset 1 — 49.9685 against 49.9713), which is what its
// UNDECLINED neighbour on the same body came back at (0.0062 %). So the record names a fallback that
// costs anywhere from nothing measurable to a quarter of the face — which is exactly why it is reported
// rather than judged at the point of decline: the mesher that gave up cannot know where the face lands.

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

// recordOn stamps a pending decline onto the mesh the face shipped instead, and is the only place the
// code is raised. A nil mesh carries no diagnostics anywhere in this package, so it is left alone.
func (l *chartDeclineLog) recordOn(m *Mesh, s geom.Surface) *Mesh {
	if m == nil || l.why == "" {
		return m
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
