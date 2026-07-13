// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// weldRings welds each face loop's points to shared-vertex indices (identity-preserving, #1600) and
// collapses its zero-length self-loop stubs in lock-step (collapseDeadLoop mutates the loop). Returns
// the per-face rings the rest of assembleBody keys on.
func weldRings(faces []filletFace, w *pointWelder, weld float64) [][][]int {
	rings := make([][][]int, len(faces))
	for i := range faces {
		for li := range faces[i].loops {
			l := &faces[i].loops[li]
			ring := w.weldRingID(l.pts, l.srcV)
			rings[i] = append(rings[i], collapseDeadLoop(l, ring, weld))
		}
	}
	return rings
}

// collapseDeadLoop removes zero-length self-loop segments from a welded fillet loop — a segment whose
// two endpoints welded to ONE vertex (ring[k]==ring[k+1]) AND whose curve is itself degenerate (a nil
// straight, or a near-zero-length arc). Such a segment mints a zero-length edge used once, which opens
// the shell into a non-solid (the F6/T6/U3 invalid-solid class: a corner build leaves a coincident-
// point stub on a loop). It drops index k from ALL FOUR loop arrays AND the ring together, so the loop
// the B2 orientation pass later reverses stays internally consistent (pts/srcV ride the point, curves/
// srcE the leaving segment). Returns the collapsed ring.
//
// A LEGITIMATE closed seam is also ring[k]==ring[k+1] (a full-circle rim welds both ends to one vertex,
// B1) — but its curve is a full-circle arc that travels 2·radius from its start, so deadCurve keeps it.
// A clean loop (no dead segment) is left untouched — a no-op for the passing cases.
func collapseDeadLoop(l *filletLoop, ring []int, weld float64) []int {
	n := len(ring)
	outR := make([]int, 0, n)
	var pts []math.Point3
	var srcV, srcE []uint64
	var curves []geom.Curve3
	for k := 0; k < n; k++ {
		if ring[k] == ring[(k+1)%n] && deadCurve(curveAt(l.curves, k), weld) {
			continue // dead self-loop: drop segment k; the vertex survives via the next segment's start
		}
		outR = append(outR, ring[k])
		pts = append(pts, l.pts[k])
		srcV = append(srcV, srcIDAt(l.srcV, k))
		curves = append(curves, curveAt(l.curves, k))
		srcE = append(srcE, srcIDAt(l.srcE, k))
	}
	if len(outR) < 3 {
		return ring // collapsing would degenerate the loop — leave it for Validate to reject honestly
	}
	l.pts, l.srcV, l.curves, l.srcE = pts, srcV, curves, srcE
	return outR
}

// deadCurve reports whether a curve is degenerate to a single point (a zero-length line or arc), as
// opposed to a real edge. Crucially a full-circle seam arc — whose midpoint travels 2·radius from its
// start — is NOT dead; a nil curve on a coincident-endpoint segment is a straight zero-length
// degeneracy. Sampled rather than measured by length so it needs nothing beyond the Curve3 interface.
func deadCurve(c geom.Curve3, weld float64) bool {
	if c == nil {
		return true
	}
	lo, hi := c.Domain()
	p0 := c.PointAt(lo)
	for _, t := range []float64{0.25, 0.5, 0.75} {
		if c.PointAt(lo+(hi-lo)*t).DistanceTo(p0) >= weld {
			return false // travels away from its start — a real edge (incl. a full-circle seam)
		}
	}
	return c.PointAt(hi).DistanceTo(p0) < weld
}
