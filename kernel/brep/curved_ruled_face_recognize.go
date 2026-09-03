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
// cylinder/cone face, a loop edge is not a ruling or a plane section, or a cone face reaches its apex
// (a pole, which is no frame edge).
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
	if frame.RadSlope != 0 && lo <= 0 {
		return ruledSide{}, false
	}
	return ruledSide{surface: f.surface, axis: frame.Axis, frame: frame, band: axialWindow(frame, lo, hi)}, true
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
