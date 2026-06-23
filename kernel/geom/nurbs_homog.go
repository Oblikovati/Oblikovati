// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// hpoint4 is a control point in homogeneous (projective) form: the weight-scaled
// coordinates (w·X, w·Y, w·Z) together with the weight w. The NURBS refinement and
// degree-change algorithms (Piegl & Tiller, "The NURBS Book", ch. 5) are affine
// combinations of these homogeneous points, so they run identically for rational
// and non-rational data; the rational position is recovered by the perspective
// divide A/w in [hpoint4.point3] / [hpoint4.point2]. Working in homogeneous space
// is why a single core implements curve, 2D-curve and surface refinement (M36-F01).
type hpoint4 struct {
	x, y, z, w float64
}

// hpoint4FromCurve maps a 3D control point and its weight to homogeneous form.
func hpoint4FromCurve(p math.Point3, w float64) hpoint4 {
	return hpoint4{p.X * w, p.Y * w, p.Z * w, w}
}

// hpoint4FromCurve2d maps a 2D control point and its weight to homogeneous form (z = 0).
func hpoint4FromCurve2d(p math.Point2, w float64) hpoint4 {
	return hpoint4{p.X * w, p.Y * w, 0, w}
}

// point3 recovers the rational 3D position (A/w) and the weight w.
func (h hpoint4) point3() (math.Point3, float64) {
	return math.P3(h.x/h.w, h.y/h.w, h.z/h.w), h.w
}

// point2 recovers the rational 2D position (A/w) and the weight w.
func (h hpoint4) point2() (math.Point2, float64) {
	return math.P2(h.x/h.w, h.y/h.w), h.w
}

// lerp returns (1−a)·h + a·o, the affine blend used by knot insertion (A5.1):
// lerp(Rw[i], Rw[i+1], α) = α·Rw[i+1] + (1−α)·Rw[i].
func (h hpoint4) lerp(o hpoint4, a float64) hpoint4 {
	return hpoint4{
		h.x + a*(o.x-h.x),
		h.y + a*(o.y-h.y),
		h.z + a*(o.z-h.z),
		h.w + a*(o.w-h.w),
	}
}

// add returns the component-wise sum, for the weighted blends of degree elevation.
func (h hpoint4) add(o hpoint4) hpoint4 {
	return hpoint4{h.x + o.x, h.y + o.y, h.z + o.z, h.w + o.w}
}

// sub returns the component-wise difference, used by knot removal's back-substitution.
func (h hpoint4) sub(o hpoint4) hpoint4 {
	return hpoint4{h.x - o.x, h.y - o.y, h.z - o.z, h.w - o.w}
}

// scale returns h scaled by s.
func (h hpoint4) scale(s float64) hpoint4 {
	return hpoint4{h.x * s, h.y * s, h.z * s, h.w * s}
}

// dist returns the Euclidean distance in homogeneous 4-space, the deviation
// measure knot removal and degree reduction compare against a tolerance (A5.8/A5.11).
func (h hpoint4) dist(o hpoint4) float64 {
	dx, dy, dz, dw := h.x-o.x, h.y-o.y, h.z-o.z, h.w-o.w
	return stdmath.Sqrt(dx*dx + dy*dy + dz*dz + dw*dw)
}

// curveToHomog converts a rational curve's (ctrl, weights) to homogeneous points.
func curveToHomog(ctrl []math.Point3, weights []float64) []hpoint4 {
	pw := make([]hpoint4, len(ctrl))
	for i := range ctrl {
		pw[i] = hpoint4FromCurve(ctrl[i], weights[i])
	}
	return pw
}

// curveFromHomog recovers the (ctrl, weights) of a rational curve from homogeneous points.
func curveFromHomog(pw []hpoint4) (ctrl []math.Point3, weights []float64) {
	ctrl = make([]math.Point3, len(pw))
	weights = make([]float64, len(pw))
	for i := range pw {
		ctrl[i], weights[i] = pw[i].point3()
	}
	return ctrl, weights
}

// curve2dToHomog converts a 2D rational curve's (ctrl, weights) to homogeneous points.
func curve2dToHomog(ctrl []math.Point2, weights []float64) []hpoint4 {
	pw := make([]hpoint4, len(ctrl))
	for i := range ctrl {
		pw[i] = hpoint4FromCurve2d(ctrl[i], weights[i])
	}
	return pw
}

// curve2dFromHomog recovers the (ctrl, weights) of a 2D rational curve from homogeneous points.
func curve2dFromHomog(pw []hpoint4) (ctrl []math.Point2, weights []float64) {
	ctrl = make([]math.Point2, len(pw))
	weights = make([]float64, len(pw))
	for i := range pw {
		ctrl[i], weights[i] = pw[i].point2()
	}
	return ctrl, weights
}
