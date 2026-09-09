// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
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
// two ellipses that cross at pinch points; the SSI tracer follows each ellipse straight through those
// pinches (Oblikovati#1404), so this returns the two closed loops there too, and the four-lobe bicylinder
// SOLID is assembled from them by the same trim and stitch as every other pair. It used to need a bespoke
// constructor beside this file, plus a snap ceiling here that declined the near-pinch band to it; both are
// deleted (ADR-0061 stage 4), which is why nothing in this file classifies a pair any more.

// CodeImprintUnclosedChain marks an SSI imprint that dropped a traced chain because it did not close into
// a loop — the typed signal (#1404) that replaces silently discarding it, so a caller/test can SEE the
// imprint degraded (and the boolean will fall back) rather than discover it later downstream.
const CodeImprintUnclosedChain diag.Code = "imprint.unclosed-chain"

// CodeImprintFallbackContour marks an imprint whose SSI curves came from the fixed-grid
// marching-squares fallback, not the continuation tracer (#1597). Fallback loops are contour-quality
// (sub-grid features lost, tangencies invisible), so the imprint proceeds but the degradation is
// recorded instead of silent.
const CodeImprintFallbackContour diag.Code = "imprint.fallback-contour"

// imprintTraceLoops returns the closed loops where base and other cross, over the marching window —
// the shared imprint step of every curved pair (cylinder∩cylinder, cone∩cylinder, cone∩cone). It
// prefers the EXACT closed-form section (exactImprintLoops) and marches only the pairs that have none,
// so a result body's edges are exact wherever the intersector can make them so (#3489). The two paths
// return the same loops in the same window; they differ only in whether each carries an achieved
// tolerance.
func imprintTraceLoops(base, other geom.Surface, window geom.SurfaceGrid, res geom.Resolution, rec *diag.Recorder) []geom.Curve3 {
	if exact, ok := exactImprintLoops(base, other, window, res); ok {
		return exact
	}
	return keepImprintLoops(geom.TraceSurfaceIntersection(base, other, window), res, rec)
}

// keepImprintLoops keeps the closed loops of a trace result, recording the fallback-contour
// diagnostic when the curves came from marching squares rather than the tracer (#1597). Split from
// imprintTraceLoops so the recording branch is unit-testable without forcing a live fallback.
func keepImprintLoops(tr geom.SurfaceIntersection, res geom.Resolution, rec *diag.Recorder) []geom.Curve3 {
	if tr.ViaFallback {
		rec.Recordf(CodeImprintFallbackContour, diag.Defect,
			"SSI tracer found no curve; %d contour(s) supplied by the marching-squares fallback", len(tr.Curves))
	}
	return closedTraceLoops(tr, res, rec)
}

// closedTraceLoops keeps the traced polylines that close into a loop (first point meets last), building a
// geom.Polyline from each. An open chain that is more than a single tangency marker but never closed — the
// tracer broke at a pinch it could not cross — is dropped from the watertight boundary, but no longer
// silently: it raises a CodeImprintUnclosedChain diagnostic on rec so the degradation is visible (#1404).
// Single/short point markers (an isolated tangential contact) are not chains and are skipped quietly.
//
// Each kept loop carries the trace's ACHIEVED deviation (tr.Deviation): the imprint is a chord
// approximation of the true intersection, and every edge stitched from it inherits that tolerance so the
// result body can report how exact its boundary is instead of claiming exactness (#3489).
//
// Each loop is carried as a *geom.Polyline (a POINTER): the arrangement's run-merge compares edge curves
// with `==` to fuse consecutive same-curve edges, and a geom.Polyline VALUE is uncomparable (it holds a
// slice), so a marched loop must travel by identity. The same pointers reach both operand sides, so each
// loop's edges merge and the two sides emit the SAME curve for the shared imprint (#1403). An exact
// section arc needs no such wrapper — it is a comparable value.
func closedTraceLoops(tr geom.SurfaceIntersection, res geom.Resolution, rec *diag.Recorder) []geom.Curve3 {
	var out []geom.Curve3
	for _, pts := range tr.Curves {
		if len(pts) >= 4 && !samePoint(pts[0], pts[len(pts)-1], res) {
			rec.Recordf(CodeImprintUnclosedChain, diag.Defect,
				"SSI traced a %d-point chain that did not close (endpoint gap %g > weld %g): dropped from the imprint",
				len(pts), float64(pts[0].DistanceTo(pts[len(pts)-1])), res.Weld())
			continue
		}
		if len(pts) < 4 || !samePoint(pts[0], pts[len(pts)-1], res) {
			continue
		}
		if pl, err := geom.NewMarchedPolyline(pts, tr.Deviation); err == nil {
			out = append(out, &pl)
		}
	}
	return out
}
