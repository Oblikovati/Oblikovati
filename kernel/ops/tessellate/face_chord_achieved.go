// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The achieved chord of a curved face's mesh (#3517 review 4).
//
// "Achieved tolerance is a MEASURED output of an operation" and "never degrade silently" are two of
// this kernel's ground rules, and until now a curved face that missed the chord tolerance it was handed
// said nothing at all. It was not a small gap: measured over ALL 88 rows of occtparity's
// byteIdentityPins corpus, 135 of their 318 non-planar faces exceed PropertyQuality's 1e-3 mm across 61
// bodies — the worst at 3250× (C2 face 1) — and 86 of those carry no OTHER diagnostic, so nothing in
// the repo said anything about them. A whole-body area or volume sum absorbs a single face's chord
// deficit, which is what made them invisible.
//
// That is what let #3517's own round-5 regressions ship silently: three faces (J5 f00, K2 f03 and
// RODB∩ f01) came out of the round WORSE than the mesher that was deleted and nothing said so. Pinning
// those three would have gated those three. This reports every face that misses, which is the rule the
// ground rules actually state.
//
// It is a REPORT and not a refusal, deliberately. The mesh still ships — a coarse face beats a missing
// one in a viewport, and the chord is an approximation tolerance, not a modelling one — and the facet
// count that would close the gap is a property of the shared curve discretizer (ADR-0061 §R4.5), not
// of any one mesher. What changes is that the approximation is now visible to feature health, the API
// and the UI instead of being discoverable only by writing a probe.
//
// The measurement costs one surface POINT INVERSION per mesh edge. On an analytic surface that is a few
// Gauss-Newton steps; on a NURBS face it is the expensive one, and the per-face cost is real: measured,
// occtparity J3 face 3 goes 13 ms → 808 ms and K2 face 4 26 ms → 2.46 s, 50–95×. In aggregate it does
// not signify — TestTessellationBudget is unmoved at 0.24 s against its 2.15 s ceiling,
// TestHeavyModelBudget on EDF.STEP goes 0.34 s → 0.41 s of its 700 ms budget, and the occtparity tier
// reads 623 s against 2400 s — but the committed fixtures do not reach NURBS, so TestHeavyModelBudget is
// the only row that gates this cost and says so in its own doc.

// CodeFaceChordNotMet marks a curved face whose mesh does not achieve the chord tolerance it was asked
// for: the largest distance from one of its edges to the surface it approximates exceeds Quality.Tol().
//
// Everything integrated from such a face — its own area, and the body's volume through it — carries
// that error, and a whole-body sum hides it. The mesh ships anyway; this is what says it is coarser
// than requested.
const CodeFaceChordNotMet diag.Code = "tessellate.face-chord-not-met"

// recordAchievedChord measures the mesh's worst departure from the surface it approximates and reports
// it when that exceeds the tolerance the caller asked for.
func recordAchievedChord(m *Mesh, s geom.Surface, q Quality) *Mesh {
	if m == nil || m.TriangleCount() == 0 {
		return m
	}
	worst := worstEdgeChord(m, s)
	if worst <= q.Tol() {
		return m
	}
	m.Diagnose(diag.Diagnostic{
		Code:     CodeFaceChordNotMet,
		Severity: diag.Defect,
		Detail: fmt.Sprintf("a %T face meshed to a worst chord of %.6g mm against the %.6g mm asked for "+
			"(%.4g× the tolerance, %d triangles): the face's own area and the body's volume through it "+
			"carry that error", s, worst, q.Tol(), worst/q.Tol(), m.TriangleCount()),
	})
	return m
}

// worstEdgeChord is the largest distance from a mesh edge's midpoint to the surface — the achieved
// chord deviation, the same quantity Quality.Tol() bounds. The midpoint is where a chord departs
// furthest from a convex arc.
//
// It measures through geom.ClosestPointOnSurface and NOT through ParamAt, and the difference is not
// cosmetic. geom.Surface's own contract (surface.go) says ParamAt off-surface "returns the frame
// projection along the parameter directions, which equals the metric nearest point for the plane,
// cylinder and sphere but NOT exactly for the cone or torus" — so on a cone or an elliptical surface it
// OVER-states the gap, always in that direction, and a face inside tolerance can be reported outside
// it. This file's first version asserted the opposite of that contract and did exactly that: measured
// over the pin corpus, four faces at each faceting fired while genuinely inside tolerance (T7 f07
// reported 1.0519 against a true 0.9929, A7 f06 1.2239 against 0.9994, J6 f00 1.2098 against 0.9986,
// J8 f01 1.0145 against 0.9724), and thirteen of the rows that DID belong over the line carried a
// figure inflated by up to 1.41× — which would have propagated into the facet-count decision this
// corpus is meant to size (ADR-0061 §R4.5).
//
// ClosestPointOnSurface runs the same damped Gauss-Newton point inversion every other exact query in
// the kernel uses, so the reading is the metric distance on every surface kind. The census falls
// 139 → 135 at PropertyQuality and 103 → 99 at DefaultQuality, and those four faces are the only rows
// that move. The error was one-sided, so it could never hide a face: zero faces are over-and-silent at
// either faceting, before or after.
//
// occtparity's worstChordRatio helper must keep reading the SAME oracle, or a pin there asserts a
// different question from the diagnostic beside it.
func worstEdgeChord(m *Mesh, s geom.Surface) float64 {
	worst := 0.0
	for i := 0; i+2 < len(m.Indices); i += 3 {
		for k := range 3 {
			a, b := m.Positions[m.Indices[i+k]], m.Positions[m.Indices[i+(k+1)%3]]
			mid := math.P3((a.X+b.X)/2, (a.Y+b.Y)/2, (a.Z+b.Z)/2)
			_, _, foot := geom.ClosestPointOnSurface(s, mid)
			worst = stdmath.Max(worst, float64(mid.DistanceTo(foot)))
		}
	}
	return worst
}
