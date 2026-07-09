// SPDX-License-Identifier: GPL-2.0-only

package sketch

import stdmath "math"

// Elliptical-arc angle convention (#1829). An elliptical arc is authored with Autodesk Inventor's
// convention — the TRUE geometric angle θ, the angle at the centre from the major axis to the ray
// through an endpoint — but stored and sampled with the PARAMETRIC (eccentric-anomaly) angle a,
// where the point is P(a) = C + Rmaj·cos(a)·û + Rmin·sin(a)·v̂. For a non-circular ellipse θ ≠ a; they
// relate by a = atan2(Rmaj·sinθ, Rmin·cosθ). The authoring boundary (EllipticalArcs.Add, reached by
// the wire and GUI) converts θ→a here; importers that already supply the parametric angle (DXF/DWG
// ELLIPSE params are eccentric-anomaly) use AddParametric and skip the conversion.

// trueToParamAngle maps a true geometric angle θ to the parametric angle a for an ellipse with the
// given semi-axes. Rmin ≤ 0 (a degenerate/circle-authored arc) falls back to θ unchanged.
func trueToParamAngle(theta, majorR, minorR float64) float64 {
	if minorR <= 0 || majorR <= 0 {
		return theta
	}
	return stdmath.Atan2(majorR*stdmath.Sin(theta), minorR*stdmath.Cos(theta))
}

// paramArcFromTrue converts an elliptical arc given by TRUE start/end angles into the parametric
// start/end angles the arc stores, preserving the sweep's direction AND magnitude across the atan2
// branch cut. θ→a is a continuous, monotonically increasing reparametrisation (mod 2π), so the
// parametric sweep has the same sign as the true sweep; this only unwraps it back into that
// revolution when atan2 wrapped an endpoint to the other branch.
func paramArcFromTrue(startTrue, endTrue, majorR, minorR float64) (startParam, endParam float64) {
	aStart := trueToParamAngle(startTrue, majorR, minorR)
	aEnd := trueToParamAngle(endTrue, majorR, minorR)
	sweepTrue := endTrue - startTrue
	sweepParam := aEnd - aStart
	for sweepTrue > 0 && sweepParam <= 0 {
		sweepParam += 2 * stdmath.Pi
	}
	for sweepTrue < 0 && sweepParam >= 0 {
		sweepParam -= 2 * stdmath.Pi
	}
	return aStart, aStart + sweepParam
}
