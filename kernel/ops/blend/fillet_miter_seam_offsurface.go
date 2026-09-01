// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// The curved-miter seam's OWN off-surface do-no-harm floor — split out of fillet_miter_curved_weld.go
// (which owns the arm-side solve + shared-face retrim, plus miterWeldMeshDefect) to keep that file
// under the 500-line/one-responsibility rule. This file's one job: measure whether curvedMiterBody's
// assembled result carries a boundary edge that strays off the very face it bounds, at a scale the
// legitimate seam-chord sagitta could never reach (curvedMiterSeamOffSurfaceBand's derivation).

// curvedMiterSeamOffSurfaceBand is the worst-boundary-edge-off-its-own-surface ceiling, as a FRACTION
// of the fillet radius r — the natural length scale that separates a legitimate seam-chord sagitta
// (the polyline walkCurvedSeam emits necessarily misses the true torus∩cylinder quartic BETWEEN its
// sample stations by a small amount that shrinks with chord count) from a genuine branch-selection
// defect (a discontinuous jump between two DIFFERENT, disconnected roots of the seam equation). Measured
// on the corpus's only two live data points (A1 investigation, simple/W3, W4): a healthy chord-sampled
// seam reads 0.0046·r (W3, 8 chords, r=0.2); a corner whose seam-bottom endpoint solver
// (miterSeamBottomCyl/nearestCircleRoot, fillet_miter_curved_cylouter.go) picks the WRONG one of up to
// four contact-circle crossings — a SEPARATE, unresolved defect this A1 slice found but does not own —
// reads 0.85·r (W4). 0.1·r sits two orders of magnitude above the sagitta and one below the jump, so it
// cannot be tripped by tightening the sampler (raising filletChordsPerTurn only shrinks the sagitta
// further) and cannot be evaded by a genuine jump (which does not shrink with more chords — the walk
// still lands on the wrong root, just via more, still-disconnected, intermediate stations).
const curvedMiterSeamOffSurfaceBand = 0.1

// curvedMiterSeamOffSurface is a SECOND do-no-harm floor alongside miterWeldMeshDefect, catching a
// defect self-cross/retrace cannot see: a boundary edge whose curve strays off the very face it is
// supposed to bound (never flagged by Validate — edge-use counts and orientation are unaffected; only
// visible as a fold/kink in the tessellated mesh, TestEveryLoopSegmentLiesOnItsFace's own corpus gate).
// r is the fillet radius, curvedMiterSeamOffSurfaceBand's scale. Example: a seam-bottom branch jump
// (fillet_miter_curved_cylouter.go, unresolved — see curvedMiterSeamOffSurfaceBand) declines simple/W4
// here rather than ship a torus face with a 0.85r fold in it.
func curvedMiterSeamOffSurface(b *topo.Body, r float64) string {
	worst, where := worstBoundaryEdgeOffSurface(b)
	if worst <= curvedMiterSeamOffSurfaceBand*r {
		return ""
	}
	return fmt.Sprintf("curved miter: assembled weld carries a boundary edge %.4g off its own face's surface "+
		"(%.4g%% of the fillet radius, budget %.4g%%; %s) — a seam-sampling defect, not the near-boundary "+
		"residual this weld already reconciles", worst, worst/r*100, curvedMiterSeamOffSurfaceBand*100, where)
}

// worstBoundaryEdgeOffSurface returns the largest distance from any face's boundary-edge curve to that
// face's OWN surface (17 stations across the curve's domain, matching the corpus's own
// TestEveryLoopSegmentLiesOnItsFace measurement), plus a description of the offender.
func worstBoundaryEdgeOffSurface(b *topo.Body) (float64, string) {
	worst, where := 0.0, "(none)"
	for _, f := range b.Faces() {
		s := f.Geometry()
		if s == nil {
			continue
		}
		for _, e := range boundaryEdgesOnce(f) {
			if d := curveWorstOffSurface(e.Geometry(), s); d > worst {
				worst = d
				where = fmt.Sprintf("face %d (%T) bounded by edge %d (%T)", f.ID(), s, e.ID(), e.Geometry())
			}
		}
	}
	return worst, where
}

// boundaryEdgesOnce returns each DISTINCT edge in f's loops (a seam edge shared with a neighbour is
// measured once, not once per use).
func boundaryEdgesOnce(f *topo.Face) []*topo.Edge {
	seen := map[uint64]bool{}
	var out []*topo.Edge
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			if e := u.Edge(); e != nil && !seen[e.ID()] {
				seen[e.ID()] = true
				out = append(out, e)
			}
		}
	}
	return out
}

// curveWorstOffSurface samples c at 17 stations and returns the largest distance from a sample to its
// closest point on s — 0 (not off) when c is nil (an id-0 chord carries no curve to check).
func curveWorstOffSurface(c geom.Curve3, s geom.Surface) float64 {
	if c == nil {
		return 0
	}
	lo, hi := c.Domain()
	const stations = 17
	worst := 0.0
	for i := 0; i <= stations; i++ {
		p := c.PointAt(lo + (hi-lo)*float64(i)/float64(stations))
		_, _, foot := geom.ClosestPointOnSurface(s, p)
		if d := float64(p.DistanceTo(foot)); d > worst {
			worst = d
		}
	}
	return worst
}
