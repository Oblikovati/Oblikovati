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
// said nothing at all. It was not a small gap: measured over the thirteen byte-identity pin bodies,
// 139 of their 318 non-planar faces exceed PropertyQuality's 1e-3 mm, and NINETY of those carried no
// diagnostic of any kind — across 42 bodies, the worst at 3250× (occtparity C2 face 1). Every gate in
// the repo was blind to all ninety, because a whole-body area or volume sum absorbs a single face's
// chord deficit.
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
// The measurement costs one surface inversion per mesh edge. Measured: TestTessellationBudget is
// unmoved at 0.23 s against its 2.15 s ceiling, and the occtparity tier at 632 s against 2400 s.

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
// chord deviation. The midpoint is where a chord departs furthest from a convex arc, and the surface's
// own ParamAt is the metric nearest point, so this is the same quantity Quality.Tol() bounds.
func worstEdgeChord(m *Mesh, s geom.Surface) float64 {
	worst := 0.0
	for i := 0; i+2 < len(m.Indices); i += 3 {
		for k := range 3 {
			a, b := m.Positions[m.Indices[i+k]], m.Positions[m.Indices[i+(k+1)%3]]
			mid := math.P3((a.X+b.X)/2, (a.Y+b.Y)/2, (a.Z+b.Z)/2)
			u, v := s.ParamAt(mid)
			worst = stdmath.Max(worst, float64(mid.DistanceTo(s.PointAt(u, v))))
		}
	}
	return worst
}
