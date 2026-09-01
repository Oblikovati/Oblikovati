// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Cone–cylinder imprint (M2 Phase 2, Oblikovati/Oblikovati#1335). The first slice of the cone∩cylinder
// boolean: a cone (or frustum) crossing a cylinder meet in a curve that is generally NOT analytic — a
// quartic the predictor–corrector SSI tracer (geom.IntersectSurfaceSurface) marches as a closed polyline,
// exactly as for two crossing cylinders (curved_crossing_imprint.go), but with one operand a cone. It
// returns the loops where the two surfaces cross, each a closed polyline lying on BOTH surfaces to
// tolerance — the foundation the later split/classify/stitch slices build the watertight result on.
//
// Scope note: a tapered rod (a narrow cone/frustum) crossing a fatter cylinder gives clean, well-separated
// closed loops (the rod's entry and exit). The trace is windowed to the cone's apex-distance band so the
// cone's finite extent clips any continuation beyond its caps.

// coneCylinderImprint returns the intersection loops of a cone body and a cylinder body as closed polylines,
// or ok=false when the two bodies are not one bare cone and one bare cylinder, or no closed loop is traced.
// The cone is the trace base, windowed to its apex-distance band [vMin, vMax]; the periodic angular
// direction is left to the tracer.
func coneCylinderImprint(a, b *topo.Body, rec *diag.Recorder) ([]geom.Curve3, bool) {
	if _, _, _, _, ok := coneAndCylinder(a, b); !ok {
		return nil, false
	}
	// The trace is the general curved-crossing imprint (ADR-0058 phase 3); this wrapper keeps only the
	// one-cone-one-cylinder guard (the dispatch classification kernel/ops keys on).
	return curvedImprintLoops(a, b, rec)
}

// coneAndCylinder resolves two bodies into one bare cone (with its apex-distance band) and one bare
// cylinder, in either order. ok=false unless exactly one is a cone solid and the other a cylinder solid.
func coneAndCylinder(a, b *topo.Body) (cone geom.Cone, cyl geom.Cylinder, vMin, vMax float64, ok bool) {
	if cn, lo, hi, okCone := coneSolidParams(facesOfAny(a)); okCone {
		if cy, _, _, okCyl := cylinderSolidParams(facesOfAny(b)); okCyl {
			return cn, cy, lo, hi, true
		}
	}
	if cn, lo, hi, okCone := coneSolidParams(facesOfAny(b)); okCone {
		if cy, _, _, okCyl := cylinderSolidParams(facesOfAny(a)); okCyl {
			return cn, cy, lo, hi, true
		}
	}
	return geom.Cone{}, geom.Cylinder{}, 0, 0, false
}
