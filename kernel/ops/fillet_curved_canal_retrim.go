// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The canal-aware HOST retrims (M6' C4 W3, architecture: canal-armweld-architecture.md §"The 4-boundary
// weld + host retrims" and §"Far-runout reuse"). Two host families are re-clipped for the canal corner:
//
//   - the CORNER roll host the canal touches directly (N7: the cylinder WALL R=50) is re-clipped so its
//     bitten loop FOLLOWS the shared foot-locus curve (feet[0]) — retrimCanalHost, the canal analogue of
//     the single-ball retrimCurvedHost but keyed on the shared foot-locus instead of w.center; and
//   - every FAR-RUNOUT bitten host (the y=0 cut, the exit caps) is spliced by the EXISTING
//     farArcsBiting/farRunoutFace path VERBATIM — canalFarOrPassthrough calls the same leaf functions the
//     single-ball curvedHostFace does, so the two produce byte-identical far faces (ADR: verbatim reuse).
//
// WATERTIGHTNESS RISK (architect-flagged, W3): for the foot-locus splice to ANCHOR, the foot-locus
// ENDPOINTS must lie on the host's original bitten loop within res.Weld·r. retrimCanalHost VERIFIES this
// (footLocusOnLoop) and HONEST-DECLINES with the measured gap when it does not — it never loosens the
// splice tolerance to force a false weld. On the real N7 wall the foot-locus endpoints (the corner
// vertices V0/V3) are the ARM-RAIL junction points, ~37 units interior to the wall loop, so the splice
// cannot anchor and the weld floors honestly; see .superpowers/sdd/armweld-w3-report.md for the measured
// anchor gaps escalated to the geometry-math-advisor.

// retrimCanalHost re-clips one CORNER roll host so its bitten loop follows the shared canal foot-locus:
// bittenLoop picks the loop the canal opens (keyed on a point ON the foot-locus, not w.center — the canal
// touches this host along the foot-locus, so a mid-foot point is the reliable bite key), then the
// foot-locus is spliced in as the bite via the existing insertSplits + spliceCornerBite. The spliced edge
// carries the SAME curve object the arm/corner samples (footLocus), so the retrimmed host and the canal
// patch land on identical points (ADR-C4-2). It HONEST-DECLINES (ok=false) when the bitten loop is
// ambiguous, the foot-locus endpoints do not anchor on that loop (the watertightness guard — never a
// loosened tolerance), or the splice fails. Example:
//
//	ff, ok := retrimCanalHost(wallFace, boundaries.feet[0], w, res)
//	if !ok { /* decline the weld — do-no-harm (see canalWallDeclineReason for the gap) */ }
func retrimCanalHost(host *topo.Face, footLocus geom.Curve3, w cornerWeld, res Resolution) (filletFace, bool) {
	tol := res.Weld() * w.radius
	star, ok := bittenLoop(host, footLocusMid(footLocus), tol)
	if !ok {
		return filletFace{}, false // no unambiguous bitten loop — do-no-harm
	}
	if !footLocusOnLoop(star, footLocus, tol) {
		return filletFace{}, false // anchor gap: foot-locus endpoints not on the bitten loop (see report)
	}
	spliced, ok := spliceCornerBite(segsFromLoop(star), footLocusBite(footLocus), tol)
	if !ok {
		return filletFace{}, false // the foot-locus bite could not be spliced onto the bitten loop
	}
	loops := append([]filletLoop{loopFromSegs(spliced)}, loopsExcept(host, star)...)
	return filletFace{surface: host.Geometry(), loops: loops, parent: host.Lineage()}, true
}

// footLocusMid is a point at the middle of the foot-locus — a point genuinely ON the host the canal
// touches, so bittenLoop keys on the opened wire (the analogue of the single-ball retrim's w.center key).
func footLocusMid(c geom.Curve3) math.Point3 {
	lo, hi := c.Domain()
	return c.PointAt((lo + hi) / 2)
}

// footLocusBite turns the foot-locus into the bite endSeg spliceCornerBite substitutes: its from/to are
// the foot-locus endpoints and its curve is the foot-locus ITSELF (the shared object, ADR-C4-2), so the
// retrimmed host loop carries the same curve the arm/corner neighbour does. arc=false: a foot-locus is a
// canal BSpline isocurve, not a circular Arc3d, so spliceCornerBite treats it as a general curved bite.
func footLocusBite(c geom.Curve3) endSeg {
	lo, hi := c.Domain()
	return endSeg{from: c.PointAt(lo), to: c.PointAt(hi), curve: c}
}

// footLocusOnLoop is the WATERTIGHTNESS ANCHOR guard: both foot-locus endpoints must lie on the bitten
// loop within tol for the splice to anchor. It NEVER loosens tol to force a weld — a failing anchor is an
// honest decline (the caller reports the measured gap for escalation).
func footLocusOnLoop(loop *topo.Loop, footLocus geom.Curve3, tol float64) bool {
	g0, g1 := footLocusAnchorGaps(loop, footLocus)
	return g0 <= tol && g1 <= tol
}

// footLocusAnchorGaps measures the distance from each foot-locus ENDPOINT to the nearest point on the
// bitten loop's edges — the escalation evidence the architect flagged. Returned raw (not thresholded) so
// the decline diagnostic and the W3 anchor test can report the exact gaps per host/endpoint.
func footLocusAnchorGaps(loop *topo.Loop, footLocus geom.Curve3) (float64, float64) {
	lo, hi := footLocus.Domain()
	return nearestOnLoopEdges(loop, footLocus.PointAt(lo)), nearestOnLoopEdges(loop, footLocus.PointAt(hi))
}

// canalAnchorSamples is the per-edge chord count used to measure the foot-locus↔bitten-loop anchor gap.
// A dimensionless sampling density (not a length), so ADR-0042's model-relative rule does not apply; 64
// resolves the nearest-point distance far below any weld tolerance the anchor is compared against.
const canalAnchorSamples = 64

// nearestOnLoopEdges is the minimum distance from p to any point sampled along the loop's edges (each arc
// sampled on its true circle, each straight edge on its chord) — the on-loop anchor metric.
func nearestOnLoopEdges(loop *topo.Loop, p math.Point3) float64 {
	best := stdmath.Inf(1)
	for _, s := range segsFromLoop(loop) {
		for _, q := range edgeSamplePoints(s, canalAnchorSamples) {
			if d := float64(p.DistanceTo(q)); d < best {
				best = d
			}
		}
	}
	return best
}

// edgeSamplePoints samples an endSeg into n+1 points along its true geometry (the arc's circle, or the
// straight chord), for the nearest-point anchor measure.
func edgeSamplePoints(s endSeg, n int) []math.Point3 {
	pts := make([]math.Point3, 0, n+1)
	if arc, ok := s.curve.(geom.Arc3d); ok && s.arc {
		for k := 0; k <= n; k++ {
			pts = append(pts, arc.PointAt(float64(k)/float64(n)))
		}
		return pts
	}
	for k := 0; k <= n; k++ {
		pts = append(pts, s.from.Lerp(s.to, math.Scalar(float64(k)/float64(n))))
	}
	return pts
}

// canalHostFaces retrims every body face for the canal corner (the canal analogue of curvedHostFaces):
// the WALL roll host (the cylinder the two wall-sharing arms roll on) is re-clipped to follow the shared
// wall foot-locus feet[0] via retrimCanalHost; every far-runout bitten host is spliced by the EXISTING
// farArcsBiting/farRunoutFace path VERBATIM (canalFarOrPassthrough); any face neither touches passes
// through transformFace unchanged. Declines with a diagnostic reason (carrying the anchor gap on a wall
// decline) on any retrim failure. bundles carries each arm's far cross-section arc (canalArmBundles), the
// rail shared with the far-runout hosts. NOTE (W3): the s_10 foot-locus feet[1] welds to the s_10 ARM
// face (W2), not to a distinct body host, and the two plane corner hosts are bitten by the arms' host
// rails (no canal foot-locus) — those seams are the W4 whole-body-assembly concern (see the report).
func canalHostFaces(body *topo.Body, arms []edgeFillet, w cornerWeld, boundaries canalBoundaries, bundles []armRails, res Resolution) ([]filletFace, string) {
	wall := canalWallHost(arms)
	tol := res.Weld() * w.radius
	out := make([]filletFace, 0, len(body.Faces()))
	for _, f := range body.Faces() {
		ff, reason := canalHostFace(f, wall, boundaries, bundles, w, res, tol)
		if reason != "" {
			return nil, reason
		}
		out = append(out, ff)
	}
	return out, ""
}

// canalHostFace routes one host face to its treatment: the wall roll host to the foot-locus retrim, every
// other face to the verbatim far-runout / pass-through path.
func canalHostFace(f, wall *topo.Face, boundaries canalBoundaries, bundles []armRails, w cornerWeld, res Resolution, tol float64) (filletFace, string) {
	if f == wall {
		ff, ok := retrimCanalHost(f, boundaries.feet[0], w, res)
		if !ok {
			return filletFace{}, canalWallDeclineReason(f, boundaries.feet[0], tol)
		}
		return ff, ""
	}
	return canalFarOrPassthrough(f, bundles, tol)
}

// canalWallHost is the cylinder roll host the wall-sharing arms roll on (feet[0]'s host), or nil when the
// corner has no single cylinder wall (then no body face matches and the wall retrim is skipped).
func canalWallHost(arms []edgeFillet) *topo.Face {
	wallFace, _, ok := tangentCornerWall(arms)
	if !ok {
		return nil
	}
	return wallFace
}

// canalWallDeclineReason names WHY the wall foot-locus retrim declined, carrying the measured
// foot-locus-endpoint→bitten-loop ANCHOR gaps — the escalation evidence for a non-anchoring splice.
func canalWallDeclineReason(host *topo.Face, footLocus geom.Curve3, tol float64) string {
	star, ok := bittenLoop(host, footLocusMid(footLocus), tol)
	if !ok {
		return fmt.Sprintf("canal wall host retrim declined: no unambiguous bitten loop (tol %.3e)", tol)
	}
	g0, g1 := footLocusAnchorGaps(star, footLocus)
	return fmt.Sprintf("canal wall host retrim declined: foot-locus endpoints %.4e / %.4e off the bitten loop (watertightness anchor, tol %.3e)", g0, g1, tol)
}

// canalFarOrPassthrough is the VERBATIM far-runout / pass-through branch, mirroring the far half of the
// single-ball curvedHostFace: a face bitten by any arm's far cross-section arc is spliced by the EXISTING
// farRunoutFace; an untouched face is carried through by transformFace. It calls the same leaf functions
// the single-ball path does (no reimplementation), so the far faces are byte-identical.
func canalFarOrPassthrough(f *topo.Face, bundles []armRails, tol float64) (filletFace, string) {
	bites := farArcsBiting(f, bundles, tol)
	if len(bites) == 0 {
		return transformFace(f, nil, nil, nil, nil), "" // untouched by the corner — carried through verbatim
	}
	ff, ok := farRunoutFace(f, bites, tol)
	if !ok {
		return filletFace{}, fmt.Sprintf("canal far-runout retrim declined (%T, %d bites)", f.Geometry(), len(bites))
	}
	return ff, ""
}
