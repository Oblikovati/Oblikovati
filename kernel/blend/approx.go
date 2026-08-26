// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// canalGridU/V are the section-station and cross-section sample counts for the general canal fit —
// oversampled ~2× the control net so the least-squares fit smooths (Piegl-Tiller §9.4.1). A single
// blend segment sweeps a modest angle, so these suffice for a bicubic within tolerance.
const (
	canalGridU = 15
	canalGridV = 11
	canalCtrlU = 8
	canalCtrlV = 7
)

// fitCanal builds the blend surface for a NON-primitive centre curve (an ellipse or a marched NURBS
// curve, where analyticBlendSurface has no closed form): it samples the section arc of radius r at
// each station along the centre curve's parameter span [t0,t1] and least-squares fits a bicubic
// B-spline (u along the guide, v across the arc). It returns StatusTwistedSurface when a station has
// no valid section (the canal folds — the pipe-embedding bound r·κ_centre ≥ 1 of ADR-0050 P4), and
// StatusWalkingFailed if the fit's linear solve is singular.
func fitCanal(centre geom.Curve3, a, b geom.Surface, r, t0, t1, tol float64, inside func(math.Point3) bool) (geom.Surface, ErrorStatus) {
	pts := make([]math.Point3, 0, canalGridU*canalGridV)
	us := make([]float64, 0, canalGridU*canalGridV)
	vs := make([]float64, 0, canalGridU*canalGridV)
	for iu := range canalGridU {
		u := float64(iu) / (canalGridU - 1)
		arc, ok := sectionAt(centre.PointAt(t0+u*(t1-t0)), a, b, r, tol, inside)
		if !ok {
			return nil, StatusTwistedSurface
		}
		for iv := range canalGridV {
			v := float64(iv) / (canalGridV - 1)
			pts = append(pts, arc.PointAt(v))
			us = append(us, u)
			vs = append(vs, v)
		}
	}
	surf, err := geom.ApproximateSurfaceLS(pts, us, vs, 3, 3, canalCtrlU, canalCtrlV)
	if err != nil {
		return nil, StatusWalkingFailed
	}
	return surf, StatusOk
}
