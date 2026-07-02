// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Cone–cone imprint (M2 Phase 2, Oblikovati/Oblikovati#1335). The last SSI imprint of Phase 2: two cones (or
// frustums) crossing — a tapered rod through a fatter tapered body. Their surfaces meet in a curve that is
// generally NOT analytic (a quartic), which the predictor–corrector SSI tracer (geom.IntersectSurfaceSurface)
// marches as a closed polyline, exactly as for two crossing cylinders (curved_crossing_imprint.go) and a
// cone crossing a cylinder (curved_cone_cylinder_imprint.go), but with BOTH operands cones. It returns the
// loops where the two surfaces cross, each a closed polyline lying on BOTH surfaces to tolerance — the
// foundation the later split/classify/stitch slices build the watertight result on.
//
// Scope note: a tapered rod (a narrow cone/frustum) crossing a fatter cone gives clean, well-separated closed
// loops (the rod's entry and exit). The trace is windowed to the FIRST cone's apex-distance band so that
// cone's finite extent clips any continuation beyond its caps.

// coneConeImprint returns the intersection loops of two bare cone bodies as closed polylines, or ok=false
// when either body is not a bare cone (a cone + a cylinder is the cone–cylinder case, handled elsewhere), or
// no closed loop is traced. The first body's cone is the trace base, windowed to its apex-distance band
// [vMin, vMax]; the periodic angular direction is left to the tracer.
func coneConeImprint(a, b *topo.Body, rec *diag.Recorder) ([]geom.Polyline, bool) {
	ca, vMin, vMax, okA := coneSolidParams(facesOfAny(a))
	cb, _, _, okB := coneSolidParams(facesOfAny(b))
	if !okA || !okB {
		return nil, false
	}
	window := geom.SurfaceGrid{VMin: vMin, VMax: vMax}
	res := geom.ResolutionForBox(a.RangeBox().Union(b.RangeBox())) // model-relative loop-closure weld (#1399)
	loops := imprintTraceLoops(ca, cb, window, res, rec)
	if len(loops) == 0 {
		return nil, false
	}
	return loops, true
}
