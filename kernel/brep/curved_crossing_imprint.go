// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Crossing-cylinder imprint (M2 Phase 2, Oblikovati/Oblikovati#1335). Phase 1 imprints are analytic
// conics on a cutting plane (a half-space). Two crossing cylinders meet in a curve that is generally NOT
// analytic — a quartic "saddle" the predictor–corrector SSI tracer (geom.IntersectSurfaceSurface) marches
// as a closed polyline. This is the imprint stage of the curved∩curved boolean: it returns the loops
// where the two cylinder surfaces cross, each a closed polyline lying on BOTH surfaces to tolerance — the
// foundation the split/classify/stitch slices build the watertight result on.
//
// Scope note: a thinner cylinder crossing a fatter one (radii unequal) gives clean, well-separated closed
// loops (a rod's entry/exit through the fat wall). Two EQUAL-radius perpendicular cylinders intersect in
// two ellipses that cross at pinch points; the SSI tracer now follows each ellipse straight through those
// pinches (Oblikovati#1404), so this returns the two closed loops there too. The bicylinder SOLID is still
// assembled by the exact analytic constructor (curved_steinmetz.go) because its four-lobe topology differs
// from the rod-band result this file builds — full unification onto the traced loops is tracked under the
// general curved∩curved pipeline (Oblikovati#1403).

// CodeImprintUnclosedChain marks an SSI imprint that dropped a traced chain because it did not close into
// a loop — the typed signal (#1404) that replaces silently discarding it, so a caller/test can SEE the
// imprint degraded (and the boolean will fall back) rather than discover it later downstream.
const CodeImprintUnclosedChain diag.Code = "imprint.unclosed-chain"

// CodeImprintFallbackContour marks an imprint whose SSI curves came from the fixed-grid
// marching-squares fallback, not the continuation tracer (#1597). Fallback loops are contour-quality
// (sub-grid features lost, tangencies invisible), so the imprint proceeds but the degradation is
// recorded instead of silent.
const CodeImprintFallbackContour diag.Code = "imprint.fallback-contour"

// CodeImprintNearPinchDeclined marks a crossing-cylinder imprint declined in the near-pinch RESIDUAL band —
// radii unequal by more than the snap ceiling, but whose two intersection loops have a neck so narrow
// (gap/chord below nearPinchGapChords, #1781) that the (u,v) arrangement fuses the two lens caps: the
// boolean takes the deterministic faceted fallback and records the degradation instead of assembling
// silently wrong analytic topology. Radii closer than the snap ceiling do NOT record this — they snap to the
// exact Steinmetz constructor (#1780); folding this residual band onto the analytic path is #1818.
const CodeImprintNearPinchDeclined diag.Code = "imprint.near-pinch-declined"

// imprintTraceLoops traces base∩other over window and keeps the loops that close — the shared trace
// step of every curved-imprint pair (cylinder∩cylinder, cone∩cylinder, cone∩cone).
func imprintTraceLoops(base, other geom.Surface, window geom.SurfaceGrid, res geom.Resolution, rec *diag.Recorder) []geom.Polyline {
	return keepImprintLoops(geom.TraceSurfaceIntersection(base, other, window), res, rec)
}

// keepImprintLoops keeps the closed loops of a trace result, recording the fallback-contour
// diagnostic when the curves came from marching squares rather than the tracer (#1597). Split from
// imprintTraceLoops so the recording branch is unit-testable without forcing a live fallback.
func keepImprintLoops(tr geom.SurfaceIntersection, res geom.Resolution, rec *diag.Recorder) []geom.Polyline {
	if tr.ViaFallback {
		rec.Recordf(CodeImprintFallbackContour, diag.Defect,
			"SSI tracer found no curve; %d contour(s) supplied by the marching-squares fallback", len(tr.Curves))
	}
	return closedTraceLoops(tr.Curves, res, rec)
}

// crossingCylinderLoops traces the intersection loops of two bare cylinder bodies as closed polylines,
// declining (silently) only below the Steinmetz snap ceiling where the exact bicylinder constructor takes
// over (#1780). Unlike crossingCylinderImprint it does NOT decline the near-pinch band — the unified intersect
// driver (ruledCrossingIntersect) resolves that band robustly by trimming the fat wall per loop (#1818),
// so it needs the raw loops. Callers that cannot yet handle the near-pinch band (cut/join, cap-crossing) use
// crossingCylinderImprint, which adds the near-pinch decline.
func crossingCylinderLoops(a, b *topo.Body, rec *diag.Recorder) ([]geom.Polyline, bool) {
	ca, _, _, okA := cylinderSolidParams(facesOfAny(a))
	cb, _, _, okB := cylinderSolidParams(facesOfAny(b))
	if !okA || !okB {
		return nil, false
	}
	res := geom.ResolutionForBox(a.RangeBox().Union(b.RangeBox())) // model-relative loop-closure weld (#1399)
	// Near-equal radii below the snap ceiling are the Steinmetz pinch's noise-floor neighbourhood: the exact
	// bicylinder constructor snaps the radii to their mean and builds the pinched solid (#1780). Decline the
	// rod-band path SILENTLY here so dispatch falls through to it — there is no degradation to record.
	if ca.Radius != cb.Radius && stdmath.Abs(ca.Radius-cb.Radius) <= steinmetzSnapCeiling(res) {
		return nil, false
	}
	// The trace is the general curved-crossing imprint (ADR-0058 phase 3); this wrapper keeps only the
	// both-cylinder guard and the Steinmetz snap-ceiling conditioning gate above.
	return curvedImprintLoops(a, b, rec)
}

// crossingCylinderImprint returns the crossing-cylinder loops for callers that decline the near-pinch band to
// their CSG fallback (cut/join, cap-crossing). It is crossingCylinderLoops plus the near-pinch gate: where the
// two loops' neck is narrow relative to the imprint chord, the fat wall's holed-wall trim would fuse the two
// lens holes (a conditioning wall, δ~√Δr vs facet error ~h²/√Δr — #1781), so it records the degradation and
// declines. The INTERSECT driver bypasses this (per-loop fat trim handles the band, #1818); folding cut/join
// onto the analytic path (the holed-wall neck bridge) is the remaining #1818 work.
func crossingCylinderImprint(a, b *topo.Body, rec *diag.Recorder) ([]geom.Polyline, bool) {
	loops, ok := crossingCylinderLoops(a, b, rec)
	if !ok {
		return nil, false
	}
	// Equal-radius loops touch at the Steinmetz pinches (neck 0) — that is the exact bicylinder, handled by
	// steinmetzGeneral, NOT a near-pinch decline. Only the UNEQUAL near-pinch band declines here.
	if unequalRadiusCrossing(a, b) && nearPinchLoops(loops) {
		rec.Recordf(CodeImprintNearPinchDeclined, diag.Defect,
			"crossing cylinders with a narrow imprint neck (gap/chord = %.2g) decline the analytic imprint: near-pinch lens fusion, falling back", loopGapChordRatio(loops))
		return nil, false
	}
	return loops, true
}

// unequalRadiusCrossing reports whether two bare cylinder bodies have different radii — the near-pinch decline
// applies only to the UNEQUAL band; equal radii are the exact Steinmetz pinch handled by steinmetzGeneral.
func unequalRadiusCrossing(a, b *topo.Body) bool {
	ca, _, _, okA := cylinderSolidParams(facesOfAny(a))
	cb, _, _, okB := cylinderSolidParams(facesOfAny(b))
	return okA && okB && ca.Radius != cb.Radius
}

// nearPinchLoops reports whether a pair of imprint loops is in the near-pinch band the (u,v) arrangement
// cannot resolve watertight: two loops whose mutual closest approach (the neck) is below nearPinchGapChords
// times the imprint's own typical chord. The criterion is on the loops' GEOMETRY (a model-relative,
// scale-free ratio), not on the radius gap — a resolved-but-narrow neck is the actual failure driver (#1781,
// grounded in the conditioning analysis: the two lens loops separate as O(√Δr) while their faceted cusps
// deviate as O(h²/√Δr), so the arrangement fuses them once the neck falls toward a few chord lengths).
func nearPinchLoops(loops []geom.Polyline) bool {
	return len(loops) == 2 && loopGapChordRatio(loops) < nearPinchGapChords
}

// nearPinchGapChords is the neck-to-chord ratio below which the near-pinch lens fusion sets in. Calibrated so
// the analytic path ships only where its tessellated result is verified watertight (freeEdgeCount==0) and
// declines to the faceted route below (#1781); it is dimensionless, so it is model-scale invariant.
const nearPinchGapChords = 0.95

// loopGapChordRatio is the two loops' mutual closest approach (the neck) divided by their typical chord — the
// dimensionless separation the near-pinch gate reads. +Inf when there are not exactly two loops.
func loopGapChordRatio(loops []geom.Polyline) float64 {
	if len(loops) != 2 {
		return stdmath.Inf(1)
	}
	chord := typicalLoopChord(loops)
	if chord <= 0 {
		return stdmath.Inf(1)
	}
	return interLoopMinDistance(loops[0], loops[1]) / chord
}

// interLoopMinDistance is the minimum distance between any vertex of one loop and any vertex of the other —
// the resolved neck of a two-loop near-pinch imprint.
func interLoopMinDistance(a, b geom.Polyline) float64 {
	min := stdmath.Inf(1)
	for _, pa := range a.Vertices {
		for _, pb := range b.Vertices {
			if d := float64(pa.DistanceTo(pb)); d < min {
				min = d
			}
		}
	}
	return min
}

// typicalLoopChord is the mean consecutive-vertex chord length over both loops — the imprint's own length
// scale, which the neck is measured against so the gate is spacing- and model-scale independent.
func typicalLoopChord(loops []geom.Polyline) float64 {
	var sum float64
	var n int
	for _, lp := range loops {
		for i := 1; i < len(lp.Vertices); i++ {
			sum += float64(lp.Vertices[i-1].DistanceTo(lp.Vertices[i]))
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

// cylinderTraceWindow is the (u, v) window for tracing on a cylinder base: the full periodic angle (left
// to the tracer) and the body's own axial span, from the base cap up by its height.
func cylinderTraceWindow(c geom.Cylinder, base math.Point3, height float64) geom.SurfaceGrid {
	vLo := float64(c.Origin.VectorTo(base).Dot(c.AxisDir.AsVector()))
	return geom.SurfaceGrid{VMin: vLo, VMax: vLo + height}
}

// closedTraceLoops keeps the traced polylines that close into a loop (first point meets last), building a
// geom.Polyline from each. An open chain that is more than a single tangency marker but never closed — the
// tracer broke at a pinch it could not cross — is dropped from the watertight boundary, but no longer
// silently: it raises a CodeImprintUnclosedChain diagnostic on rec so the degradation is visible (#1404).
// Single/short point markers (an isolated tangential contact) are not chains and are skipped quietly.
func closedTraceLoops(raw [][]math.Point3, res geom.Resolution, rec *diag.Recorder) []geom.Polyline {
	var out []geom.Polyline
	for _, pts := range raw {
		if len(pts) >= 4 && !samePoint(pts[0], pts[len(pts)-1], res) {
			rec.Recordf(CodeImprintUnclosedChain, diag.Defect,
				"SSI traced a %d-point chain that did not close (endpoint gap %g > weld %g): dropped from the imprint",
				len(pts), float64(pts[0].DistanceTo(pts[len(pts)-1])), res.Weld())
			continue
		}
		if len(pts) < 4 || !samePoint(pts[0], pts[len(pts)-1], res) {
			continue
		}
		if pl, err := geom.NewPolyline(pts); err == nil {
			out = append(out, pl)
		}
	}
	return out
}
