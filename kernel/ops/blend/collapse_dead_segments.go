// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/mesh"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CodeAssembleDeadLoopCollapse marks a fillet loop whose zero-length dead segments cannot be removed
// without leaving fewer than three vertices — a loop that has collapsed to a point or an edge. Dropping
// the dead segments would degenerate it; keeping them ships a zero-length edge that opens the shell.
// Neither is a valid loop, so the assembly refuses it HERE with a named defect naming the loop, rather
// than deferring the discovery to a later, generic Validate pass (#3389).
const CodeAssembleDeadLoopCollapse diag.Code = "assemble.dead-loop-collapse"

// deadLoopRefusal names a loop weldRings could not collapse to a valid ring: which face and loop, and
// how many vertices survived the attempted collapse (< 3). assembleBody records one Defect per refusal.
type deadLoopRefusal struct {
	face, loop, survived, welded int
}

// weldRings welds each face loop's points to shared-vertex indices (identity-preserving, #1600) and
// collapses its zero-length self-loop stubs in lock-step (collapseDeadLoop mutates the loop). Returns
// the per-face rings the rest of assembleBody keys on, plus the loops whose collapse was REFUSED
// because it would degenerate the loop (< 3 vertices) — the caller records those as defects (#3389).
func weldRings(faces []filletFace, w *mesh.PointWelder, weld float64) ([][][]int, []deadLoopRefusal) {
	rings := make([][][]int, len(faces))
	var refused []deadLoopRefusal
	for i := range faces {
		for li := range faces[i].loops {
			l := &faces[i].loops[li]
			ring := w.WeldRingID(l.pts, l.srcV)
			collapsed, survived := collapseDeadLoop(l, ring, weld)
			if survived >= 0 {
				refused = append(refused, deadLoopRefusal{face: i, loop: li, survived: survived, welded: len(ring)})
			}
			rings[i] = append(rings[i], collapsed)
		}
	}
	return rings, refused
}

// recordDeadLoopRefusals stamps the builder with one Defect per loop weldRings could not collapse to a
// valid ring. The detail names the offending face and loop and the surviving vertex count, so the
// reader can reproduce it — the CLAUDE.md exception-message rule — rather than reading a generic
// "invalid solid" from a downstream Validate. No refusals is a no-op.
func recordDeadLoopRefusals(bld *topo.Builder, refused []deadLoopRefusal) {
	for _, r := range refused {
		bld.Diagnose(diag.Diagnostic{
			Code:     CodeAssembleDeadLoopCollapse,
			Severity: diag.Defect,
			Detail: fmt.Sprintf("face %d loop %d: dead-segment collapse would leave %d vertices (welded %d), below the "+
				"3 a loop needs; the loop has collapsed to a point or an edge and cannot bound a face", r.face, r.loop, r.survived, r.welded),
		})
	}
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
//
// survived is -1 when the loop is fine (collapsed or clean); when removing the dead segments would
// leave fewer than three vertices it is that surviving count (< 3), signalling the caller to refuse
// the loop with a named defect (#3389) instead of deferring the malformed body to a later Validate.
func collapseDeadLoop(l *filletLoop, ring []int, weld float64) (result []int, survived int) {
	n := len(ring)
	outR := make([]int, 0, n)
	var pts []math.Point3
	var srcV, srcE []uint64
	var curves []geom.Curve3
	for k := range n {
		if ring[k] == ring[(k+1)%n] && deadCurve(curveAt(l.curves, k), weld) {
			continue // dead self-loop: drop segment k; the vertex survives via the next segment's start
		}
		outR = append(outR, ring[k])
		pts = append(pts, l.pts[k])
		srcV = append(srcV, probe.SrcIDAt(l.srcV, k))
		curves = append(curves, curveAt(l.curves, k))
		srcE = append(srcE, probe.SrcIDAt(l.srcE, k))
	}
	if len(outR) < 3 {
		// Collapsing would degenerate the loop. Keep the original ring so the builder still assembles
		// (a partial body beats a nil), but report the surviving count so the caller refuses it loudly
		// with a named defect rather than shipping it silently for a later Validate to reject.
		return ring, len(outR)
	}
	l.pts, l.srcV, l.curves, l.srcE = pts, srcV, curves, srcE
	return outR, -1
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
