// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Wall recognition for the mixed per-face boolean (ADR-0060). A wall is any cylinder or cone face whose
// boundary the loop-framed chart (ruledFaceUV) can carry: every edge a ruling or a plane section, so
// each frame×imprint incidence has a closed form. The face's LOOPS are its frame — a bare two-rim band,
// a side with an oblique elliptical rim, a side already notched by an earlier cut, a partial patch —
// so there is no band recogniser to satisfy and no second chart for the already-cut case. The axial
// window the ruledSide carries is only the frame's exact extent, read by the pair-clearance gates.

// ruledFaceOf resolves a face to the ruled side the mixed boolean trims, or ok=false when it is not a
// cylinder/cone face, or a loop edge is not a ruling or a plane section.
//
// A cone face may reach its APEX, and that is not a decline (ADR-0062). The apex is no frame edge —
// the whole azimuth there is one point — but a face closed by it carries a RULING out to it, and that
// ruling IS a frame edge, so the window reaches the apex on the face's own loops. The chart then closes
// its parameter rectangle there (ruledFaceUV.apexSegments), exactly as sphereFaceUV closes its own at a
// pole. A face STRADDLING the apex is still refused: the radius changes sign across it, so the two
// nappes are two surfaces and one chart cannot carry both.
func ruledFaceOf(f curvedFace) (ruledSide, bool) {
	frame, ok := geom.RuledFrameOf(f.surface)
	if !ok || len(f.loops) == 0 {
		return ruledSide{}, false
	}
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, l := range f.loops {
		if !loopChainCloses(l, geom.ResolutionForBox(faceLoopBox(f))) {
			return ruledSide{}, false // the frame's even-odd containment needs every loop closed
		}
		for _, e := range l.edges {
			elo, ehi, ok := geom.AxialExtent(e.curve, e.t0, e.t1, frame.Base, frame.Axis)
			if !ok || e.t0 == e.t1 {
				return ruledSide{}, false
			}
			lo, hi = stdmath.Min(lo, elo), stdmath.Max(hi, ehi)
		}
	}
	lo, hi, ok = apexBoundedWindow(frame, lo, hi)
	if !ok {
		return ruledSide{}, false
	}
	return ruledSide{surface: f.surface, axis: frame.Axis, frame: frame, band: axialWindow(frame, lo, hi)}, true
}

// apexBoundedWindow admits a cone face that reaches its APEX. The apex is not a frame edge — the whole
// azimuth there is one point — but a face closed by it carries a RULING out to it and back, and that
// ruling is a frame edge, so the face's own loops already reach v=0 and the window needs no extension
// (ADR-0060: the loops are the frame, and here they are enough). What the recognizer used to do was
// refuse the face outright, which sent every apex cone to the pass bucket and declined the whole
// boolean (ADR-0062).
//
// ok=false only for a face STRADDLING the apex: the radius changes sign across it, so the two nappes
// are two surfaces and one chart cannot carry both.
func apexBoundedWindow(frame geom.RuledFrame, lo, hi float64) (float64, float64, bool) {
	if frame.RadSlope != 0 && lo < 0 && hi > 0 {
		return 0, 0, false
	}
	return lo, hi, true
}

// axialWindow is the band the pair gates read: the frame's exact axial extent and the radii there.
func axialWindow(frame geom.RuledFrame, lo, hi float64) coneSideBand_ {
	return coneSideBand_{
		bottom: frame.Base.TranslateBy(frame.Axis.Scale(math.Scalar(lo))),
		top:    frame.Base.TranslateBy(frame.Axis.Scale(math.Scalar(hi))),
		vMin:   lo, vMax: hi, rBot: frame.Radius(lo), rTop: frame.Radius(hi),
	}
}

// loopChainCloses reports whether a loop's edges form one closed chain: each edge ends where the
// next begins, and a lone edge closes on itself.
func loopChainCloses(l curvedLoop, res geom.Resolution) bool {
	if len(l.edges) == 0 {
		return false
	}
	for i, e := range l.edges {
		next := l.edges[(i+1)%len(l.edges)]
		if float64(e.end().DistanceTo(next.start())) > res.Weld() {
			return false
		}
	}
	return true
}
