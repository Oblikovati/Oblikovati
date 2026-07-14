// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// extractTrihedral turns a solved planar-trihedral cornerBlend into the 3-sided RailLoop the exact
// sphere tier recognizes (three concentric equal-radius great-circle arcs on the corner sphere). Each
// Side's Curve is the boundary arc between successive arm tangent points, lying on cb.sphere; Adjacent
// is the sphere itself (recognition is rails-only, so Adjacent is not load-bearing here) and Cont is
// G1. This is the PLANAR strangler path: it reuses the existing arc geometry (no setback
// recomputation) so the downstream face stays byte-for-byte. ok=false if an arc cannot be
// reconstructed → honest-reject, letting spherePatchFace fall back to cb.sphere (do-no-harm).
//
// Usage: loop, ok := extractTrihedral(cb); if ok { patch, ok := resolveBlend(loop, scale) }.
func extractTrihedral(cb *cornerBlend) (RailLoop, bool) {
	if len(cb.arcs) != 3 {
		return RailLoop{}, false
	}
	sides := make([]Side, 3)
	for i, a := range cb.arcs {
		arc, ok := sphereBoundaryArc(cb.sphere, a)
		if !ok {
			return RailLoop{}, false
		}
		sides[i] = Side{Curve: arc, Adjacent: cb.sphere, Cont: G1}
	}
	return RailLoop{Sides: sides, Provenance: topo.Lineage{}}, true
}

// sphereBoundaryArc reconstructs the exact geom.Arc3d for one blendArc on cb.sphere: the great-circle
// arc through ta→mid→tb (its circumcircle recovers the same center/radius as cb.sphere because the
// three tangent points lie on a great circle). sph is unused today — recognition reads the arc alone —
// but is kept in the signature to document the surface the arc belongs to. ok=false on collinear
// points (no circle is determined), which propagates to extractTrihedral's honest-reject.
func sphereBoundaryArc(sph geom.Sphere, a blendArc) (geom.Arc3d, bool) {
	_ = sph
	arc, err := geom.Arc3dByThreePoints(a.ta, a.mid, a.tb)
	if err != nil {
		return geom.Arc3d{}, false
	}
	return arc, true
}
