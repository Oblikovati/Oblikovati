// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"fmt"

	"oblikovati/kernel/exchange/step/part21"
)

// STEP wraps the real edge geometry inside several "carrier" curve entities that add no
// shape of their own — they reference a basis 3D curve plus representation metadata
// (pcurves, trim parameters). Most kernels other than SolidWorks (OpenCASCADE, Parasolid)
// emit edges this way, so unwrapping them is what makes a real-world STEP importable:
//
//	SURFACE_CURVE/SEAM_CURVE/INTERSECTION_CURVE(name, curve_3d, (pcurves…), master)
//	TRIMMED_CURVE(name, basis_curve, trim_1, trim_2, sense_agreement, master)
//
// In both, the geometric curve is parameter index 1 (after the name). The edge's own
// vertices already bound the curve (build_edge trims start→end), so the basis curve alone
// is what we need; pcurves and trim params are ignored.

// wrappedCurve maps a carrier curve by resolving its basis curve at parameter index i and
// mapping that. Recurses through nested carriers (a TRIMMED_CURVE of a SURFACE_CURVE …).
func wrappedCurve(g *part21.EntityGraph, ent *part21.RawEntity, i int, scale float64) (MappedCurve, error) {
	ref, err := refParam(ent.Params, i)
	if err != nil {
		return MappedCurve{}, fmt.Errorf("geommap: %s basis curve: %w", ent.Keyword, err)
	}
	return Curve(g, ref, scale)
}
