// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// spherePatchFace builds the corner sphere patch: a spherical triangle bounded by the blend's three
// arcs (each shared with a cylinder fillet), wound so its normal points outward (away from the sphere
// centre). The SURFACE is now VALIDATED through the generalized engine (extractTrihedral → resolveBlend
// recognizes the exact sphere), proving the RailLoop path agrees; the boundary LOOP is still
// chainArcs(cb.arcs) so the assembled face is byte-for-byte with the pre-strangler output (the
// trim/arcs are the byte-for-byte risk — see the extractor-wave spec §2/§3).
func spherePatchFace(cb *cornerBlend) filletFace {
	surface := sphereSurfaceViaRail(cb) // extractor-recognized sphere, == cb.sphere by construction
	loop := chainArcs(cb.arcs)
	if spherePatchFlipped(cb, loop) {
		loop = reverseArcLoop(loop)
	}
	return filletFace{surface: surface, loops: []filletLoop{loop}}
}

// sphereSurfaceViaRail routes the corner sphere through the RailLoop engine and returns the recognized
// analytic sphere; it falls back to cb.sphere if extraction/recognition declines (do-no-harm), so a
// mis-extraction can never change the byte-for-byte output.
func sphereSurfaceViaRail(cb *cornerBlend) geom.Surface {
	loop, ok := extractTrihedral(cb)
	if !ok {
		return cb.sphere
	}
	patch, ok := resolveBlend(loop, sphereScale(cb))
	if !ok || patch.Kind != BlendKindSphere {
		return cb.sphere
	}
	return patch.Surface
}

// sphereScale is the model-relative tolerance for the sphere recognition, derived from the arc tangent
// points so it never uses a bare epsilon (ADR-0042); spherePatchFace's caller has no Resolution in
// scope, so it is recomputed here from the same geometry the loop is built from.
func sphereScale(cb *cornerBlend) Resolution {
	pts := make([]math.Point3, 0, len(cb.arcs)*2)
	for _, a := range cb.arcs {
		pts = append(pts, a.ta, a.tb)
	}
	return ResolutionForPoints(pts)
}

// chainArcs links the blend's arcs head-to-tail into a closed loop (each tangent point is an
// endpoint of exactly two arcs), with the arc curve on each segment.
func chainArcs(arcs []blendArc) filletLoop {
	used := make([]bool, len(arcs))
	var fl filletLoop
	cur := arcs[0].ta
	for range arcs {
		for j, a := range arcs {
			if used[j] {
				continue
			}
			from, to, ok := arcEndpoints(a, cur)
			if !ok {
				continue
			}
			used[j] = true
			appendBlendArc(&fl, a, from, to)
			cur = to
			break
		}
	}
	return fl
}

// appendBlendArc appends one blend arc oriented from→to: a faceted chord polyline (straight segments)
// when the arc is shared with a variable cone, otherwise a single Arc3d through its midpoint. Only the
// segment's start points are added (its end is the next arc's start), so the loop stays closed.
func appendBlendArc(fl *filletLoop, a blendArc, from, to math.Point3) {
	if a.chords == nil {
		arc, _ := geom.Arc3dByThreePoints(from, a.mid, to)
		fl.pts = append(fl.pts, from)
		fl.curves = append(fl.curves, arc)
		return
	}
	pts := orientedChords(a.chords, from) // ta…tb, reversed when entering from tb
	for i := 0; i+1 < len(pts); i++ {
		fl.pts = append(fl.pts, pts[i])
		fl.curves = append(fl.curves, nil) // straight chord matching the cone's ruling end
	}
}

// arcEndpoints orients an arc so it starts at cur (returning from=cur, to=other), or ok=false
// when neither end is cur.
func arcEndpoints(a blendArc, cur math.Point3) (from, to math.Point3, ok bool) {
	switch {
	case a.ta.DistanceTo(cur) < 1e-7:
		return a.ta, a.tb, true
	case a.tb.DistanceTo(cur) < 1e-7:
		return a.tb, a.ta, true
	}
	return math.Point3{}, math.Point3{}, false
}

// spherePatchFlipped reports whether the loop winds against the sphere's outward normal at the
// patch centroid (so it should be reversed to face outward).
func spherePatchFlipped(cb *cornerBlend, loop filletLoop) bool {
	c := centroidPts(loop.pts)
	n := loop.pts[0].VectorTo(loop.pts[1]).Cross(loop.pts[0].VectorTo(loop.pts[2]))
	return n.Dot(cb.center.VectorTo(c)) < 0
}

// reverseArcLoop reverses a closed arc loop, re-deriving each segment's arc in the new direction
// (the arc midpoints are recovered from the original arcs). Straight chord segments (nil curve, from a
// faceted variable-cone arc) stay straight rather than being mis-fitted to an arc through the origin.
func reverseArcLoop(loop filletLoop) filletLoop {
	n := len(loop.pts)
	mids := arcMidpoints(loop)
	var out filletLoop
	for i := range n {
		from := loop.pts[(n-i)%n]
		to := loop.pts[(n-i-1+n)%n]
		src := (n - i - 1 + n) % n
		var curve geom.Curve3
		if loop.curves[src] != nil {
			curve, _ = geom.Arc3dByThreePoints(from, mids[src], to)
		}
		out.pts = append(out.pts, from)
		out.curves = append(out.curves, curve)
	}
	return out
}

// arcMidpoints samples each segment's arc curve at its midpoint (for re-orienting the loop).
func arcMidpoints(loop filletLoop) []math.Point3 {
	mids := make([]math.Point3, len(loop.curves))
	for i, c := range loop.curves {
		if c != nil {
			mids[i] = c.PointAt(0.5)
		}
	}
	return mids
}
