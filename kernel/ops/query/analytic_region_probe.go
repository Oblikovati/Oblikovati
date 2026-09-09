// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
)

// The search for one parameter point on a chosen side of a face's loops (M48/C3 #3453), and the
// WINDOWS it grids. Two callers need it: the integrator's side test, which asks for a point inside
// the region the loops enclose, and FaceInteriorPoint, which asks for a point the FACE holds — the
// enclosed region on most faces, its complement on a closed surface where the face is the far side.
//
// The window is what this file exists for. One grid over the loops' shared bounding box is a fixed
// number of samples spread over the whole box, so it resolves a region only while that region is
// larger than one of its cells. On a face whose loops are DISJOINT that is not the region's own
// scale: an axial drill through a ring leaves the torus face two bore mouths half a tube-turn apart,
// so the shared box spans ~π in v while each mouth spans 2·bore/minor. Measured on the RING pair
// (Oblikovati/Oblikovati#3516): every grid point missed both mouths at every bore below
// π·minor/(2·regionProbeGrid) = 0.0714 — declining at 0.0631 and certifying at 0.08 — so the side
// test could not be certified, the analytic integral declined the face, and the tessellated fallback
// measured the ring 2.839 mm³ light, 300000x the material a 1e-3 bore removes. The boolean's volume
// bracket then rejected a body that was right; nothing could measure it.
//
// So the grid is laid over each loop's OWN box as well as the shared one, shared box first. A face
// whose shared box already yields a deep probe on its first rows — the ordinary trimmed face, and
// the case the early exit was written for — takes exactly the window and the point it always did.
// A face whose shared box does NOT reach that depth on its own, which is the slender region the
// retained comment below says needs the full sweep, now also searches the per-loop windows and may
// return a different, deeper point than before. That is the intended change: deeper is better for
// both callers, and it is not "unchanged".

// regionProbeGrid is the resolution of the search for one point strictly on the wanted side of the
// loops. It only has to find A point, not a particular one, so the count is a robustness margin for
// a slender region, not an accuracy parameter: nothing downstream depends on which point it returns.
const regionProbeGrid = 33

// regionProbeDeepEnough is the depth, as a fraction of the window's diagonal, past which a probe is
// unambiguous and the search stops. The scan is over a grid of points each measured against every
// boundary sample, so on an ordinary face — where the first row already lands well inside — this
// turns a full sweep into a few rows. A slender region never reaches it and falls back to the full
// sweep, which is the case that needs one.
const regionProbeDeepEnough = 0.1 // tol:parametric — probe depth that ends the search, relative

// probeWindow is one parameter-space box the interior search grids.
type probeWindow struct{ uLo, uHi, vLo, vHi float64 }

// spans reports whether the window has a positive extent in both parameters.
func (w probeWindow) spans() bool { return w.uLo < w.uHi && w.vLo < w.vHi }

// diagonal is the window's parameter-space diagonal, the length regionProbeDeepEnough is taken of.
func (w probeWindow) diagonal() float64 { return stdmath.Hypot(w.uHi-w.uLo, w.vHi-w.vLo) }

// at returns the (i, j) interior grid point of the window.
func (w probeWindow) at(i, j int) (u, v float64) {
	return w.uLo + (w.uHi-w.uLo)*float64(i)/regionProbeGrid,
		w.vLo + (w.vHi-w.vLo)*float64(j)/regionProbeGrid
}

// regionProbeWindows are the boxes the interior search grids, in the order it tries them: the loops'
// SHARED box first — the ordinary face's own box, and the one that answers in a single row — then
// each loop's box, so a loop far smaller than the shared box is still resolved at its own scale.
// A single-loop face has only the shared box, which is that loop's, so it grids exactly one window.
func regionProbeWindows(polys [][]arcSample) []probeWindow {
	shared := probeWindow{}
	shared.uLo, shared.uHi, shared.vLo, shared.vHi = uvPolygonBounds(polys)
	if len(polys) < 2 {
		return []probeWindow{shared}
	}
	out := make([]probeWindow, 1, len(polys)+1)
	out[0] = shared
	for _, poly := range polys {
		var w probeWindow
		w.uLo, w.uHi, w.vLo, w.vHi = uvPolygonBounds([][]arcSample{poly})
		out = append(out, w)
	}
	return out
}

// regionInteriorUV returns one parameter point strictly inside the region the loops enclose: of the
// grid points with an ODD even-odd crossing count — inside the outer loop and outside every hole —
// it takes the one FARTHEST from the boundary. Depth matters: a probe a hair inside the trim is a
// point where two independent classifiers may legitimately disagree, and the answer here selects a
// branch, so the point must be unambiguous rather than merely inside. The samples are the loops'
// unwrapped uv polylines, so a seam-crossing loop stays a simple polygon here.
func regionInteriorUV(loops []faceLoop) (u, v float64, ok bool) {
	polys, per := loopUVPolygons(loops), loopsUVPeriod(loops)
	return deepestProbe(polys, per, regionProbeWindows(polys), true)
}

// faceComplementUV returns one parameter point OUTSIDE every loop. It is the probe for a face on a
// CLOSED surface that holds the complement of what its loops enclose — a torus with a bore through
// it, a sphere with a hole — where the enclosed-region probe lands on the side the face does not own
// and there is otherwise no representative point at all. An unbounded rectangle has no grid to lay
// and declines.
//
// It grids the SAME window list the enclosed side does, plus the surface's own rectangle, so there is
// one mechanism and not two. What that reaches is complement area lying inside some loop's own
// bounding box, at that loop's scale. What it does NOT reach — stated because it would otherwise read
// as fixed — is a thin complement lying BETWEEN two disjoint loops: no loop's box contains it, and
// the surface rectangle grids it at one fixed scale, which is this file's own indictment applied to
// the far side. A probe built from the gap between two loops' nearest samples would reach it; there
// is no measured case demanding one yet, and four faces in kernel/ops/boolean are still unprobed.
//
// A loop set that WRAPS the parameter seam is searched too, and its probe is UNRANKED. regionProbeUV
// refuses to put such a loop to the even-odd test one file over, on the ground that a wrapping loop
// is not a closed polygon and its crossing parity says nothing — and by that argument the depth this
// file ranks by says nothing either. Declining on it was tried and REVERTED: it cost probe coverage
// on 22 faces across kernel/ops/boolean and 5 on its drill/torus/ring rows, in exchange for a
// robustness argument with no measured case behind it. brep.PointInFaceTrim certifies every probe
// this returns against the face itself, so what a meaningless ranking can produce is a probe near the
// trim rather than deep inside it — and a certified probe near the trim is strictly more proof than
// no probe at all, which is what the certificate has for a face it cannot probe. The ambiguity
// regionProbeDeepEnough guards against is real for the SIDE test, which chooses a branch; this
// caller only needs a point the face holds. Pinned by TestTheComplementProbeAnswersForASeamWrappingLoop,
// which fails the moment the decline returns, and by TestEveryFaceOfTheCorpusPairsIsProbeable, which
// holds the coverage the decline cost.
func faceComplementUV(s geom.Surface, loops []faceLoop) (u, v float64, ok bool) {
	var rect probeWindow
	rect.uLo, rect.uHi = s.UDomain()
	rect.vLo, rect.vHi = s.VDomain()
	if !allFinite(rect.uLo, rect.uHi, rect.vLo, rect.vHi) {
		return 0, 0, false
	}
	polys, per := loopUVPolygons(loops), loopsUVPeriod(loops)
	return deepestProbe(polys, per, append([]probeWindow{rect}, regionProbeWindows(polys)...), false)
}

// deepestProbe grids each window in turn and keeps the point that sits deepest RELATIVE to the window
// that found it, stopping as soon as one reaches regionProbeDeepEnough of its own diagonal.
//
// The comparison is relative because the ACCEPTANCE is: deepestInWindow stops at a fraction of the
// window's diagonal, so ranking windows by absolute depth would let a probe 2% inside a shared box
// spanning half the chart beat a probe 30% inside a small loop's own box — and 2% inside is exactly
// the ambiguous point regionProbeDeepEnough exists to reject. One scale for the rule and another for
// the ranking is two rules.
func deepestProbe(polys [][]arcSample, per uvPeriod, windows []probeWindow, inside bool) (u, v float64, ok bool) {
	best := 0.0
	for _, w := range windows {
		pu, pv, d := deepestInWindow(polys, per, w, inside)
		if d > 0 && d/w.diagonal() > best {
			best, u, v = d/w.diagonal(), pu, pv
		}
		if best >= regionProbeDeepEnough {
			break
		}
	}
	return u, v, best > 0
}

// deepestInWindow returns the window's grid point farthest from the loops on the wanted side, or a
// depth of −1 when the window holds none.
func deepestInWindow(polys [][]arcSample, per uvPeriod, w probeWindow, inside bool) (u, v, depth float64) {
	if !w.spans() {
		return 0, 0, -1
	}
	depth, deepEnough := -1.0, regionProbeDeepEnough*w.diagonal()
	for i := 1; i < regionProbeGrid && depth < deepEnough; i++ {
		for j := 1; j < regionProbeGrid; j++ {
			pu, pv := w.at(i, j)
			if d := uvDepthOn(polys, pu, pv, per, inside); d > depth {
				depth, u, v = d, pu, pv
			}
		}
	}
	return u, v, depth
}

// uvDepthOn is how far (u, v) sits from the loops on the wanted side — inside the region they
// enclose when inside is true, outside every one of them when it is false — measured as the distance
// to the nearest boundary sample. It is −1 on the other side.
func uvDepthOn(polys [][]arcSample, u, v float64, per uvPeriod, inside bool) float64 {
	if uvCrossingsOdd(polys, u, v, per) != inside {
		return -1
	}
	nearest := stdmath.Inf(1)
	for _, poly := range polys {
		for _, s := range poly {
			nearest = stdmath.Min(nearest, stdmath.Hypot(s.u-u, s.v-v))
		}
	}
	return nearest
}

// uvPolygonBounds is the parameter-space box of every polygon.
func uvPolygonBounds(polys [][]arcSample) (uLo, uHi, vLo, vHi float64) {
	uLo, vLo = stdmath.Inf(1), stdmath.Inf(1)
	uHi, vHi = stdmath.Inf(-1), stdmath.Inf(-1)
	for _, poly := range polys {
		for _, s := range poly {
			uLo, uHi = stdmath.Min(uLo, s.u), stdmath.Max(uHi, s.u)
			vLo, vHi = stdmath.Min(vLo, s.v), stdmath.Max(vHi, s.v)
		}
	}
	return uLo, uHi, vLo, vHi
}
