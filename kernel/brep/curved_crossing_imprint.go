// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
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

// crossingCylinderImprint returns the intersection loops of two bare cylinder bodies as closed polylines,
// or ok=false when either body is not a bare cylinder or no closed loop is traced. The trace window spans
// the first body's axial extent (the cylinders cross within it), and the periodic angular direction is
// resolved by the tracer automatically. A non-nil rec receives a diagnostic for any traced chain that
// failed to close (#1404).
func crossingCylinderImprint(a, b *topo.Body, rec *diag.Recorder) ([]geom.Polyline, bool) {
	ca, baseA, heightA, okA := cylinderSolidParams(facesOfAny(a))
	cb, _, _, okB := cylinderSolidParams(facesOfAny(b))
	if !okA || !okB {
		return nil, false
	}
	res := geom.ResolutionForBox(a.RangeBox().Union(b.RangeBox())) // model-relative loop-closure weld (#1399)
	loops := closedTraceLoops(geom.IntersectSurfaceSurface(ca, cb, cylinderTraceWindow(ca, baseA, heightA)), res, rec)
	if len(loops) == 0 {
		return nil, false
	}
	return loops, true
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
