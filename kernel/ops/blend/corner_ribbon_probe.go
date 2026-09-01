// SPDX-License-Identifier: GPL-2.0-only
package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

// ribbonSeamNonFolding is the F2 runtime probe. For every G1 side it asserts the MATCHED fill's
// into-patch cross-derivative still agrees with the BASE Coons interior cross-derivative there — the
// sign-sensitive test creaseAngle omits — then requires the whole patch to pass the anti-fold sweep.
// It reconstructs the base from the rails (identical to the base used during the match). G0 /
// ribbon-less sides are skipped.
//
// WHY NOT compare the fill normal to the ribbon normal: for a VMin↔VMin Order-1 match the operator
// forces F_v(boundary) = −dir EXACTLY, so the fill normal nf = −nr IDENTICALLY for BOTH orientations —
// a boundary nf·nr test is tautological, blind to the fold (proven + empirically confirmed in the F2
// wave). The base's interior cross-derivative depends only on rail POSITIONS (ribbon-independent), so
// matched-vs-base DOES flip sign with the ribbon orientation and is the real discriminator.
func ribbonSeamNonFolding(fill geom.BSplineSurface, rails [4]geom.BSplineCurve, sides [4]geom.FillSide, scale tol.Resolution) bool {
	base, err := geom.CoonsFill(rails[0], rails[1], rails[2], rails[3])
	if err != nil {
		return false
	}
	edges := coons4Edges()
	for i, e := range edges {
		if sides[i].Order <= 0 {
			continue
		}
		if !matchedCrossPointsInward(fill, base, e, scale) {
			return false
		}
	}
	return obstacleNoFold(fill, scale)
}

// matchedCrossPointsInward compares the matched fill's into-patch cross-derivative at edge e's
// midpoint against the base Coons interior cross-derivative there. Correct (outward) ribbon: the
// matched cross-derivative lands back inside the patch, agreeing with base (dot > 0). Inward/folded
// ribbon: it flips (dot < 0). Boundary-local and exact — catches even a shallow fold (no 24×24
// sampling luck). Abstains (true) when either derivative is degenerate below the model-scaled weld floor.
func matchedCrossPointsInward(fill, base geom.BSplineSurface, e fillEdge, scale tol.Resolution) bool {
	cf, cb := inwardCrossAt(fill, e), inwardCrossAt(base, e)
	if cf.Length() < scale.Weld() || cb.Length() < scale.Weld() {
		return true
	}
	return cf.Dot(cb) > 0
}

// inwardCrossAt returns the into-patch cross-derivative at fill edge e's midpoint, reusing the
// obstacle certify sign convention (+∂/∂v at VMin, −∂/∂v at VMax, +∂/∂u at UMin, −∂/∂u at UMax) so it
// matches the awayRef anchor coons4Sides/obstacleSides build against.
func inwardCrossAt(s geom.BSplineSurface, e fillEdge) math.Vector3 {
	switch e {
	case edgeVMin:
		return inwardCrossV(s, false)
	case edgeVMax:
		return inwardCrossV(s, true)
	case edgeUMin:
		return inwardCrossU(s, false)
	default: // edgeUMax
		return inwardCrossU(s, true)
	}
}
