// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
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
// two ellipses that cross at pinch points where the tracer's continuation struggles; that degenerate
// (Steinmetz) case is left to a later slice, which can fit those planar loops to exact ellipses.

// crossingCylinderImprint returns the intersection loops of two bare cylinder bodies as closed polylines,
// or ok=false when either body is not a bare cylinder or no closed loop is traced. The trace window spans
// the first body's axial extent (the cylinders cross within it), and the periodic angular direction is
// resolved by the tracer automatically.
func crossingCylinderImprint(a, b *topo.Body) ([]geom.Polyline, bool) {
	ca, baseA, heightA, okA := cylinderSolidParams(facesOfAny(a))
	cb, _, _, okB := cylinderSolidParams(facesOfAny(b))
	if !okA || !okB {
		return nil, false
	}
	loops := closedTraceLoops(geom.IntersectSurfaceSurface(ca, cb, cylinderTraceWindow(ca, baseA, heightA)))
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
// geom.Polyline from each. An open chain — where the tracer broke at a tangency or pinch — is dropped, so
// the imprint carries only watertight boundary loops.
func closedTraceLoops(raw [][]math.Point3) []geom.Polyline {
	var out []geom.Polyline
	for _, pts := range raw {
		if len(pts) < 4 || !samePoint(pts[0], pts[len(pts)-1]) {
			continue
		}
		if pl, err := geom.NewPolyline(pts); err == nil {
			out = append(out, pl)
		}
	}
	return out
}
